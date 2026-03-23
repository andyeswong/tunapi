package types

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
)

// WSMessage is the base WebSocket message envelope
type WSMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type HelloPayload struct {
	AgentID string `json:"agentId"`
	Token   string `json:"token"`
}

type HelloAckPayload struct {
	AgentID string `json:"agentId"`
	OK      bool   `json:"ok"`
}

type OpenStreamPayload struct {
	StreamID  string            `json:"streamId"`
	LocalPort int               `json:"localPort"`
	Host      string            `json:"host"`
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body,omitempty"`
}

type StreamOpenedPayload struct {
	StreamID string `json:"streamId"`
}

type StreamDataPayload struct {
	StreamID string `json:"streamId"`
	Data     string `json:"data"`
	More     bool   `json:"more"`
}

type StreamClosePayload struct {
	StreamID string `json:"streamId"`
}

type StreamErrPayload struct {
	StreamID string `json:"streamId"`
	Error    string `json:"error"`
}

// GenerateID creates a prefixed random ID
func GenerateID(prefix string) string {
	b := make([]byte, 12)
	rand.Read(b)
	return prefix + "_" + base64.RawURLEncoding.EncodeToString(b)[:16]
}

// HashToken returns the SHA-256 hash of a token
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
