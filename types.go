package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"tunapi/pkg/types"
)

// Routes wraps the route list
type Routes struct {
	Version int      `json:"version"`
	Routes  []types.Route `json:"routes"`
}

// Agent represents a registered tunagent
type Agent struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	TokenHash string    `json:"-"` // never expose in JSON
	Online    bool      `json:"online"`
	LastSeen  time.Time `json:"lastSeen"`
	Streams   int       `json:"streams"`
}

// AgentStore manages agent registration
type AgentStore struct {
	mu        sync.RWMutex
	agents    map[string]*Agent
	byName      map[string]*Agent
	nameIndex   map[string]string // name -> id
	persistFile string
}

var agentPersistFile string

func NewAgentStore() *AgentStore {
	return &AgentStore{
		agents:    make(map[string]*Agent),
		byName:    make(map[string]*Agent),
		nameIndex: make(map[string]string),
	}
}

func SetAgentPersistFile(path string) {
	agentPersistFile = path
}

func (s *AgentStore) Register(name, tokenHash string) *Agent {
	s.mu.Lock()
	defer s.mu.Unlock()

	if oldID, ok := s.nameIndex[name]; ok {
		delete(s.agents, oldID)
	}

	id := types.GenerateID("ag")
	agent := &Agent{
		ID:        id,
		Name:      name,
		TokenHash: tokenHash,
		Online:    true,
		LastSeen:  time.Now(),
	}
	s.agents[id] = agent
	s.byName[name] = agent
	s.nameIndex[name] = id
	return agent
}

func (s *AgentStore) SetOnline(id string, online bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a, ok := s.agents[id]; ok {
		a.Online = online
		a.LastSeen = time.Now()
	}
}

func (s *AgentStore) Get(id string) *Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.agents[id]
}

func (s *AgentStore) GetByName(name string) *Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.byName[name]
}

func (s *AgentStore) List() []*Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Agent, 0, len(s.agents))
	for _, a := range s.agents {
		out = append(out, a)
	}
	return out
}

func (s *AgentStore) ValidateToken(id, token string) bool {
	a := s.Get(id)
	if a == nil {
		return false
	}
	return types.HashToken(token) == a.TokenHash
}

// LoadFromFile loads agents from a JSON file
func (s *AgentStore) LoadFromFile(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("LoadFromFile: %v", err)
		}
		return
	}

	var stored struct {
		Agents []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			TokenHash string `json:"tokenHash"`
			Online    bool   `json:"online"`
			LastSeen  string `json:"lastSeen"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		log.Printf("LoadFromFile: invalid JSON: %v", err)
		return
	}

	for _, a := range stored.Agents {
		agent := &Agent{
			ID:        a.ID,
			Name:      a.Name,
			TokenHash: a.TokenHash,
			Online:    false,
		}
		s.agents[a.ID] = agent
		s.byName[a.Name] = agent
		s.nameIndex[a.Name] = a.ID
	}
	log.Printf("LoadFromFile: loaded %d agents from %s", len(stored.Agents), path)
}

// SaveToFile saves agents to a JSON file
func (s *AgentStore) SaveToFile(path string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Use global if path not provided
	if path == "" {
		path = agentPersistFile
	}
	if path == "" {
		return
	}

	var stored struct {
		Agents []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			TokenHash string `json:"tokenHash"`
			Online    bool   `json:"online"`
			LastSeen  string `json:"lastSeen"`
		} `json:"agents"`
	}
	for _, a := range s.agents {
		stored.Agents = append(stored.Agents, struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			TokenHash string `json:"tokenHash"`
			Online    bool   `json:"online"`
			LastSeen  string `json:"lastSeen"`
		}{
			ID:        a.ID,
			Name:      a.Name,
			TokenHash: a.TokenHash,
			Online:    a.Online,
			LastSeen:  a.LastSeen.Format(time.RFC3339),
		})
	}

	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		log.Printf("SaveToFile: marshal error: %v", err)
		return
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Printf("SaveToFile: write error: %v", err)
		return
	}
	log.Printf("SaveToFile: saved %d agents to %s", len(stored.Agents), path)
}

func (s *AgentStore) Touch(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a, ok := s.agents[id]; ok {
		a.LastSeen = time.Now()
	}
}

// ParseAgentToken returns the hash of a raw token
func ParseAgentToken(raw string) string {
	return types.HashToken(raw)
}

// ---- AgentSession ----

type AgentSession struct {
	AgentID   string
	Name      string
	Conn      WebSocketConn
	Streams   map[string]struct{}
	StreamsMu sync.Mutex
}

func (s *AgentSession) AddStream(id string) {
	s.StreamsMu.Lock()
	defer s.StreamsMu.Unlock()
	s.Streams[id] = struct{}{}
}

func (s *AgentSession) RemoveStream(id string) {
	s.StreamsMu.Lock()
	defer s.StreamsMu.Unlock()
	delete(s.Streams, id)
}

func (s *AgentSession) StreamCount() int {
	s.StreamsMu.Lock()
	defer s.StreamsMu.Unlock()
	return len(s.Streams)
}

// AgentHub manages live agent WebSocket sessions
type AgentHub struct {
	mu       sync.RWMutex
	sessions map[string]*AgentSession // agentID -> session
	agents   *AgentStore
}

func NewAgentHub(agents *AgentStore) *AgentHub {
	return &AgentHub{
		sessions: make(map[string]*AgentSession),
		agents:   agents,
	}
}

func (h *AgentHub) Register(agentID string, conn WebSocketConn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if old, ok := h.sessions[agentID]; ok {
		old.Conn.Close()
	}

	session := &AgentSession{
		AgentID: agentID,
		Name:    h.agents.Get(agentID).Name,
		Conn:    conn,
		Streams: make(map[string]struct{}),
	}
	h.sessions[agentID] = session
	h.agents.SetOnline(agentID, true)
}

func (h *AgentHub) Unregister(agentID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if s, ok := h.sessions[agentID]; ok {
		s.Conn.Close()
		delete(h.sessions, agentID)
	}
	h.agents.SetOnline(agentID, false)
}

func (h *AgentHub) Session(agentID string) *AgentSession {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.sessions[agentID]
}

func (h *AgentHub) RemoveStream(agentID, streamID string) {
	h.mu.RLock()
	s := h.sessions[agentID]
	h.mu.RUnlock()
	if s != nil {
		s.RemoveStream(streamID)
	}
}

func (h *AgentHub) StreamOpened(agentID, streamID string) {
	h.mu.RLock()
	s := h.sessions[agentID]
	h.mu.RUnlock()
	if s != nil {
		s.AddStream(streamID)
	}
}

func (h *AgentHub) SendToAgent(agentID string, msg []byte) error {
	h.mu.RLock()
	s, ok := h.sessions[agentID]
	h.mu.RUnlock()
	if !ok {
		return fmt.Errorf("agent not connected")
	}
	return s.Conn.WriteMessage(websocket.TextMessage, msg)
}

// proxyResponse is used internally by the agent routing system
type proxyResponse struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
	Err        error
}
