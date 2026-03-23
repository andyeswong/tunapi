package types

// RouteMode describes how a route is served
type RouteMode string

const (
	ModeDirect RouteMode = "direct"
	ModeAgent  RouteMode = "agent"
)

// Route describes a subdomain mapping
type Route struct {
	Subdomain string     `json:"subdomain"`
	Mode      RouteMode `json:"mode"`
	// direct mode
	Target    string     `json:"target,omitempty"`
	Port      int        `json:"port,omitempty"`
	// agent mode
	AgentID   string     `json:"agentId,omitempty"`
	LocalPort int        `json:"localPort,omitempty"`
}
