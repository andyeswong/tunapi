package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"tunapi/pkg/types"
)

func isValidSubdomain(s string) bool {
	return subdomainRe.MatchString(s)
}

func baseDomainFromEnv() string {
	if v := os.Getenv("TUNAPI_BASE_DOMAIN"); v != "" {
		return strings.TrimSpace(v)
	}
	return "tunapi.local"
}

func publicSchemeFromEnv() string {
	if v := os.Getenv("TUNAPI_PUBLIC_SCHEME"); v != "" {
		return strings.TrimSpace(v)
	}
	return "https"
}

func scheduleReloadnginx() {
	// no-op on dev; called to reload nginx config after route changes
	log.Println("nginx reload scheduled")
}

// Server is the main tunapi server
type Server struct {
	agents      *AgentStore
	agentHub    *AgentHub
	streamMu    sync.RWMutex
	streamResp  map[string]chan *proxyResponse
}

// NewServer creates a new server instance
func NewServer() *Server {
	agents := NewAgentStore()
	SetAgentPersistFile(agentsFile)
	agents.LoadFromFile(agentPersistFile)
	hub := NewAgentHub(agents)
	s := &Server{
		agents:     agents,
		agentHub:   hub,
		streamResp: make(map[string]chan *proxyResponse),
	}
	return s
}

// MustMarshalJSON marshals or dies
func MustMarshalJSON(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// generateToken creates a random token
func generateToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// ---- Agent API endpoints ----

func (s *Server) handleAgentCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, `{"error":"name required"}`, http.StatusBadRequest)
		return
	}

	// only allow lowercase letters, numbers, dash
	for _, c := range req.Name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			http.Error(w, `{"error":"name must be lowercase letters, numbers, dash only"}`, http.StatusBadRequest)
			return
		}
	}

	rawToken := generateToken()
	tokenHash := types.HashToken(rawToken)
	agent := s.agents.Register(req.Name, tokenHash)
	s.agents.SaveToFile("")

	log.Printf("agent created: name=%s id=%s", req.Name, agent.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id":    agent.ID,
		"name":  agent.Name,
		"token": rawToken, // only returned once
	})
}

func (s *Server) handleAgentList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}

	list := s.agents.List()
	out := make([]map[string]interface{}, 0, len(list))
	for _, a := range list {
		session := s.agentHub.Session(a.ID)
		streams := 0
		if session != nil {
			streams = session.StreamCount()
		}
		out = append(out, map[string]interface{}{
			"id":       a.ID,
			"name":     a.Name,
			"online":   a.Online,
			"lastSeen": a.LastSeen,
			"streams":  streams,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"agents": out})
}

func (s *Server) handleAgentDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "DELETE only", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/agent/")
	if id == "" {
		http.Error(w, `{"error":"agent id required"}`, http.StatusBadRequest)
		return
	}

	agent := s.agents.Get(id)
	if agent == nil {
		http.Error(w, `{"error":"agent not found"}`, http.StatusNotFound)
		return
	}

	s.agentHub.Unregister(id)
	// Note: agent record stays in AgentStore but goes offline

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"deleted": agent.Name})
}

// handlePublish creates a route via agent
func (s *Server) handlePublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Subdomain string `json:"subdomain"`
		AgentName string `json:"agent"`
		LocalPort int    `json:"localPort"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	if req.Subdomain == "" || req.AgentName == "" || req.LocalPort == 0 {
		http.Error(w, `{"error":"subdomain, agent, localPort required"}`, http.StatusBadRequest)
		return
	}

	// validate subdomain
	if strings.Contains(req.Subdomain, ".") || !isValidSubdomain(req.Subdomain) {
		http.Error(w, `{"error":"invalid subdomain"}`, http.StatusBadRequest)
		return
	}

	// validate port
	if req.LocalPort < 1 || req.LocalPort > 65535 {
		http.Error(w, `{"error":"port must be 1-65535"}`, http.StatusBadRequest)
		return
	}

	agent := s.agents.GetByName(req.AgentName)
	if agent == nil {
		http.Error(w, `{"error":"agent not found"}`, http.StatusNotFound)
		return
	}

	// check if session is live
	session := s.agentHub.Session(agent.ID)
	if session == nil {
		http.Error(w, `{"error":"agent not connected"}`, http.StatusBadRequest)
		return
	}

	// find and update or add route
	found := false
	for i := range routes.Routes {
		if routes.Routes[i].Subdomain == req.Subdomain {
			routes.Routes[i].Mode = types.ModeAgent
			routes.Routes[i].AgentID = agent.ID
			routes.Routes[i].LocalPort = req.LocalPort
			found = true
			break
		}
	}
	if !found {
		routes.Routes = append(routes.Routes, types.Route{
			Subdomain: req.Subdomain,
			Mode:      types.ModeAgent,
			AgentID:   agent.ID,
			LocalPort: req.LocalPort,
		})
	}

	saveRoutes()
	scheduleReloadnginx()

	w.Header().Set("Content-Type", "application/json")
	baseDomain := baseDomainFromEnv()
	scheme := publicSchemeFromEnv()
	json.NewEncoder(w).Encode(map[string]string{
		"url": fmt.Sprintf("%s://%s.%s", scheme, req.Subdomain, baseDomain),
	})
}

// handleUnpublish removes an agent route
func (s *Server) handleUnpublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "DELETE only", http.StatusMethodNotAllowed)
		return
	}

	subdomain := strings.TrimPrefix(r.URL.Path, "/route/")
	if subdomain == "" {
		http.Error(w, `{"error":"subdomain required"}`, http.StatusBadRequest)
		return
	}

	found := false
	for i := range routes.Routes {
		if routes.Routes[i].Subdomain == subdomain && routes.Routes[i].Mode == types.ModeAgent {
			routes.Routes = append(routes.Routes[:i], routes.Routes[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		http.Error(w, `{"error":"route not found"}`, http.StatusNotFound)
		return
	}

	saveRoutes()
	scheduleReloadnginx()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"unpublished": subdomain})
}
