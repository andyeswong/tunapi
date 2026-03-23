package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	agentID    string
	agentToken string
	agentName  string
	serverURL  string
)

func getEnvOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

type agent struct {
	id      string
	token   string
	name    string
	server  string
	conn    *websocket.Conn
	streams map[string]*streamConn
	streamsMu sync.Mutex
	done    chan struct{}
}

type streamConn struct {
	id        string
	conn      net.Conn
	localIP   string
	localPort int
}

func main() {
	agentID = getEnvOr("TUNAPI_AGENT_ID", "ag_mUXw_Fn3HPih0jyw")
	agentToken = getEnvOr("TUNAPI_AGENT_TOKEN", "Wu5ya7gro2pO2Aapm5_fOQZ0782bFO4Wx1sSi0WY-Zs")
	serverURL = getEnvOr("TUNAPI_SERVER", "wss://tunapps.andres-wong.com/agent/connect")
	agentName = getEnvOr("TUNAPI_AGENT_NAME", "cliente-test")

	if agentID == "" || agentToken == "" || serverURL == "" || agentName == "" {
		fmt.Fprintln(os.Stderr, "Error: TUNAPI_AGENT_ID, TUNAPI_AGENT_TOKEN, TUNAPI_SERVER, TUNAPI_AGENT_NAME are required")
		os.Exit(1)
	}

	a := newAgent(agentID, agentToken, agentName, serverURL)
	if err := a.connect(); err != nil {
		log.Fatalf("connection failed: %v", err)
	}

	log.Printf("tunagent connected (name=%s id=%s)", agentName, agentID)
	log.Printf("Ctrl+C to stop")

	<-a.done
	a.shutdown()
}

func newAgent(id, token, name, server string) *agent {
	return &agent{
		id:      id,
		token:   token,
		name:    name,
		server:  server,
		streams: make(map[string]*streamConn),
		done:    make(chan struct{}),
	}
}

func (a *agent) connect() error {
	u, err := url.Parse(a.server)
	if err != nil {
		return fmt.Errorf("bad server URL: %w", err)
	}

	wsScheme := u.Scheme
	if wsScheme == "https" {
		wsScheme = "wss"
	} else if wsScheme == "http" {
		wsScheme = "ws"
	}
	wsURL := fmt.Sprintf("%s://%s%s", wsScheme, u.Host, u.Path)
	log.Printf("connecting to %s", wsURL)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	header := http.Header{}
	header.Set("X-Agent-ID", a.id)
	header.Set("X-Agent-Token", a.token)
	header.Set("X-Agent-Name", a.name)

	conn, resp, err := dialer.Dial(wsURL, header)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			resp.Body.Close()
			return fmt.Errorf("websocket dial failed (%d %s): %s", resp.StatusCode, resp.Status, strings.TrimSpace(string(body)))
		}
		return fmt.Errorf("websocket dial: %w", err)
	}

	if resp.StatusCode != http.StatusSwitchingProtocols {
		conn.Close()
		return fmt.Errorf("expected 101 got %d %s", resp.StatusCode, resp.Status)
	}

	log.Printf("agent connected successfully")
	a.conn = conn
	go a.readLoop()
	return nil
}

func (a *agent) readLoop() {
	defer close(a.done)
	for {
		_, msg, err := a.conn.ReadMessage()
		if err != nil {
			if strings.Contains(err.Error(), "close") {
				log.Println("server disconnected")
			} else {
				log.Printf("read error: %v", err)
			}
			return
		}

		var m wsMessage
		if err := json.Unmarshal(msg, &m); err != nil {
			log.Printf("bad message: %v", err)
			continue
		}

		a.handleMessage(m)
	}
}

type wsMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type openStreamPayload struct {
	StreamID  string            `json:"streamId"`
	LocalPort int               `json:"localPort"`
	Host      string            `json:"host"`
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body,omitempty"`
}

type streamDataPayload struct {
	StreamID string `json:"streamId"`
	Data     string `json:"data"`
	More     bool   `json:"more"`
}

type streamClosePayload struct {
	StreamID string `json:"streamId"`
}

type streamErrPayload struct {
	StreamID string `json:"streamId"`
	Error    string `json:"error"`
}

func (a *agent) handleMessage(m wsMessage) {
	log.Printf("agent received: type=%s", m.Type)
	switch m.Type {
	case "open_stream":
		var p openStreamPayload
		if err := json.Unmarshal(m.Data, &p); err != nil {
			log.Printf("bad open_stream: %v", err)
			return
		}
		go a.openStream(p)
	case "ping":
		a.sendMsg("pong", nil)
	default:
		log.Printf("unknown message type: %s", m.Type)
	}
}

func (a *agent) openStream(p openStreamPayload) {
	a.streamsMu.Lock()
	if len(a.streams) >= 50 {
		a.streamsMu.Unlock()
		a.sendMsg("stream_error", streamErrPayload{StreamID: p.StreamID, Error: "max streams reached"})
		return
	}
	a.streamsMu.Unlock()

	addr := fmt.Sprintf("127.0.0.1:%d", p.LocalPort)
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		a.sendMsg("stream_error", streamErrPayload{StreamID: p.StreamID, Error: err.Error()})
		return
	}

	sc := &streamConn{id: p.StreamID, conn: conn, localIP: "127.0.0.1", localPort: p.LocalPort}

	a.streamsMu.Lock()
	a.streams[p.StreamID] = sc
	a.streamsMu.Unlock()

	log.Printf("openStream: stream opened successfully, streamId=%s", p.StreamID)
	a.sendMsg("stream_opened", map[string]string{"streamId": p.StreamID})

	// Build and send the HTTP request (method + path + headers [+ body])
	httpReq := a.buildHTTPRequest(p.Method, p.Path, p.Host, p.Headers, p.Body)
	log.Printf("openStream: sending HTTP request to 127.0.0.1:%d (%d bytes)", p.LocalPort, len(httpReq))
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Write([]byte(httpReq))
	if err != nil {
		log.Printf("openStream: write error: %v (wrote %d bytes)", err, n)
		conn.Close()
		return
	}
	log.Printf("openStream: request sent (%d bytes), waiting for response...", n)

	go a.copyStreamToWS(sc)
}

func (a *agent) buildHTTPRequest(method, path, host string, headers map[string]string, bodyB64 string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s %s HTTP/1.1\r\n", method, path))
	sb.WriteString(fmt.Sprintf("Host: %s\r\n", host))
	for k, v := range headers {
		// Skip Host as we already set it above; skip connection-level headers
		if k == "Host" || k == "Content-Length" {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	if bodyB64 != "" {
		b, _ := base64.StdEncoding.DecodeString(bodyB64)
		sb.WriteString(fmt.Sprintf("Content-Length: %d\r\n", len(b)))
	}
	sb.WriteString("\r\n")
	req := sb.String()
	if bodyB64 != "" {
		b, _ := base64.StdEncoding.DecodeString(bodyB64)
		req += string(b)
	}
	return req
}

func (a *agent) copyStreamToWS(sc *streamConn) {
	log.Printf("copyStreamToWS: starting for streamId=%s", sc.id)
	buf := make([]byte, 32*1024)
	for {
		n, err := sc.conn.Read(buf)
		if n > 0 {
			log.Printf("copyStreamToWS: read %d bytes from Apache, streamId=%s", n, sc.id)
			b := base64.StdEncoding.EncodeToString(buf[:n])
			a.sendMsg("stream_data", streamDataPayload{StreamID: sc.id, Data: b, More: true})
		}
		if err != nil {
			log.Printf("copyStreamToWS: error/close for streamId=%s: %v", sc.id, err)
			a.sendMsg("stream_close", streamClosePayload{StreamID: sc.id})
			break
		}
	}
}

func (a *agent) sendMsg(typ string, data interface{}) {
	var raw json.RawMessage
	if data != nil {
		b, _ := json.Marshal(data)
		raw = b
	}
	m := wsMessage{Type: typ, Data: raw}
	msg, _ := json.Marshal(m)
	if err := a.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		log.Printf("write error: %v", err)
	}
}

func (a *agent) shutdown() {
	a.streamsMu.Lock()
	for _, sc := range a.streams {
		sc.conn.Close()
	}
	a.streamsMu.Unlock()
	if a.conn != nil {
		a.conn.Close()
	}
}
