package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"tunapi/pkg/types"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

// WebSocketConn abstracts *websocket.Conn
type WebSocketConn interface {
	ReadMessage() (int, []byte, error)
	WriteMessage(int, []byte) error
	Close() error
}

// ---- WebSocket Agent Handler ----

func handleAgentConnect(s *Server, w http.ResponseWriter, r *http.Request) {
	agentID := r.Header.Get("X-Agent-ID")
	agentToken := r.Header.Get("X-Agent-Token")
	agentName := r.Header.Get("X-Agent-Name")

	if agentID == "" || agentToken == "" {
		http.Error(w, "missing auth headers", http.StatusUnauthorized)
		return
	}

	if !s.agents.ValidateToken(agentID, agentToken) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}

	s.agentHub.Register(agentID, conn)
	s.agents.Touch(agentID)
	s.agents.SaveToFile("")

	defer func() {
		s.agentHub.Unregister(agentID)
		conn.Close()
	}()

	// ping keepalive
	pingTicker := time.NewTicker(25 * time.Second)
	defer pingTicker.Stop()

	conn.SetReadLimit(65536)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	go func() {
		for range pingTicker.C {
			conn.WriteMessage(websocket.PingMessage, nil)
		}
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if strings.Contains(err.Error(), "close") {
				log.Printf("agent %s disconnected", agentName)
			} else {
				log.Printf("agent read error: %v", err)
			}
			return
		}

		s.handleAgentMsg(agentID, msg)
	}
}

func (s *Server) handleAgentMsg(agentID string, msg []byte) {
	var m types.WSMessage
	if err := json.Unmarshal(msg, &m); err != nil {
		return
	}

	switch m.Type {
	case "pong":
		// keepalive ack, nothing to do

	case "stream_opened":
		var p types.StreamOpenedPayload
		if json.Unmarshal(m.Data, &p) == nil {
			s.agentHub.StreamOpened(agentID, p.StreamID)
			s.deliverStreamData(p.StreamID, nil, true, false, nil)
		}

	case "stream_data":
		var p types.StreamDataPayload
		if json.Unmarshal(m.Data, &p) == nil {
			data, _ := base64.StdEncoding.DecodeString(p.Data)
			s.deliverStreamData(p.StreamID, data, false, p.More, nil)
			if !p.More {
				s.deliverStreamData(p.StreamID, nil, false, false, nil)
			}
		}

	case "stream_close":
		var p types.StreamClosePayload
		if json.Unmarshal(m.Data, &p) == nil {
			s.deliverStreamData(p.StreamID, nil, false, false, nil)
			s.agentHub.RemoveStream(agentID, p.StreamID)
		}

	case "stream_error":
		var p types.StreamErrPayload
		if json.Unmarshal(m.Data, &p) == nil {
			s.deliverStreamData(p.StreamID, nil, false, false, fmt.Errorf("%s", p.Error))
		}
	}
}

// RouteHTTPViaAgent proxies an HTTP request through an agent and writes the response
func (s *Server) RouteHTTPViaAgent(agentID string, localPort int, method, path string, headers map[string]string, body []byte, w http.ResponseWriter) {
	streamID := types.GenerateID("st")

	// register waiter
	respCh := make(chan *proxyResponse, 1)
	s.addStreamWaiter(streamID, respCh)
	defer s.removeStreamWaiter(streamID)

	// build request payload
	req := types.OpenStreamPayload{
		StreamID:  streamID,
		LocalPort: localPort,
		Host:     headers["Host"],
		Method:   method,
		Path:     path,
		Headers:  headers,
	}

	if len(body) > 0 {
		req.Body = base64.StdEncoding.EncodeToString(body)
	}

	// send open_stream to agent
	payloadJSON, _ := json.Marshal(req)
	msg := types.WSMessage{
		Type: "open_stream",
		Data: payloadJSON,
	}
	msgBytes, _ := json.Marshal(msg)
	if err := s.agentHub.SendToAgent(agentID, msgBytes); err != nil {
		http.Error(w, "agent unreachable: "+err.Error(), http.StatusBadGateway)
		return
	}

	// Collect response chunks
	var respBody []byte
	var statusCode int
	var respHeaders map[string]string
	var headersParsed bool
	var respWritten bool
	var bw *bufio.Writer
	var firstChunkHeaderEnd int

	for {
		select {
		case resp := <-respCh:
			if resp.Err != nil {
				if !respWritten {
					http.Error(w, "agent error: "+resp.Err.Error(), http.StatusBadGateway)
					return
				}
				return
			}
			if resp.StatusCode != 0 {
				statusCode = resp.StatusCode
				respHeaders = resp.Headers
				headersParsed = true
			}
			if len(resp.Body) > 0 {
				if !headersParsed {
					// First chunk: might contain full HTTP response (headers + body) or just headers
					headerEnd := 0
					for i := 0; i < len(resp.Body)-3; i++ {
						if resp.Body[i] == '\r' && resp.Body[i+1] == '\n' &&
							resp.Body[i+2] == '\r' && resp.Body[i+3] == '\n' {
							headerEnd = i + 4
							break
						}
					}
					if headerEnd > 0 {
						// Parse status line from headers portion
						headerStr := string(resp.Body[:headerEnd])
						lines := strings.Split(headerStr, "\r\n")
						if len(lines) > 0 && strings.HasPrefix(lines[0], "HTTP/") {
							parts := strings.Split(lines[0], " ")
							if len(parts) >= 2 {
								if sc, err := strconv.Atoi(parts[1]); err == nil {
									statusCode = sc
								}
							}
							respHeaders = make(map[string]string)
							for _, line := range lines[1:] {
								if idx := strings.Index(line, ": "); idx > 0 {
									respHeaders[strings.ToLower(line[:idx])] = line[idx+2:]
								}
							}
						}
						headersParsed = true
						firstChunkHeaderEnd = headerEnd
					}
				}

				if headersParsed && !respWritten {
					// Write headers
					status := statusCode
					if status == 0 {
						status = 200
					}
					for k, v := range respHeaders {
						w.Header().Set(k, v)
					}
					w.WriteHeader(status)
					respWritten = true
					bw = bufio.NewWriter(w)
				}

				// Write body chunk
				if headersParsed {
					bodyStart := firstChunkHeaderEnd
					if bw != nil {
						bw.Write(resp.Body[bodyStart:])
						bw.Flush()
					} else {
						w.Write(resp.Body[bodyStart:])
					}
				} else {
					respBody = append(respBody, resp.Body...)
				}
			} else if headersParsed {
				// Clean close - write any remaining body
				if len(respBody) > 0 {
					if bw != nil {
						bw.Write(respBody)
						bw.Flush()
					} else {
						w.Write(respBody)
					}
				}
				return
			}
		case <-time.After(120 * time.Second):
			if len(respBody) > 0 && !respWritten {
				w.Write(respBody)
			}
			return
		}
	}
}

// ---- stream response channel registry ----

func (s *Server) addStreamWaiter(streamID string, ch chan *proxyResponse) {
	s.streamMu.Lock()
	s.streamResp[streamID] = ch
	s.streamMu.Unlock()
}

func (s *Server) removeStreamWaiter(streamID string) {
	s.streamMu.Lock()
	delete(s.streamResp, streamID)
	s.streamMu.Unlock()
}

func (s *Server) deliverStreamData(streamID string, data []byte, opened, more bool, streamErr error) {
	s.streamMu.RLock()
	ch, ok := s.streamResp[streamID]
	s.streamMu.RUnlock()
	if !ok {
		return
	}

	if opened {
		// just signal that stream is open (connection established)
		return
	}

	if streamErr != nil {
		ch <- &proxyResponse{Err: streamErr}
		return
	}

	if data == nil && !more {
		// stream closed cleanly
		ch <- &proxyResponse{}
		return
	}

	// send data chunk
	ch <- &proxyResponse{Body: data}
}

func (s *Server) closeStreamResp(streamID string) {
	s.streamMu.RLock()
	ch, ok := s.streamResp[streamID]
	s.streamMu.RUnlock()
	if !ok {
		return
	}
	ch <- &proxyResponse{}
}
