package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"tunapi/pkg/types"
)

var (
	routes          Routes
	routesMu        sync.RWMutex
	routesFile      string
	agentsFile      string
	sharedSecret    string
	allowedTargets  map[string]struct{}
	baseDomain     string
	publicScheme   string
	serverInstance *Server
)

var listenPort string

var subdomainRe = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

func main() {
	routesFile = getEnv("TUNAPI_ROUTES_FILE", "/etc/tunapi/routes.json")
	agentsFile = getEnv("TUNAPI_AGENTS_FILE", "/etc/tunapi/agents.json")
	sharedSecret = getEnv("TUNAPI_SECRET", "changeme")
	listenPort = getEnv("TUNAPI_PORT", "8443")
	baseDomain = strings.TrimSpace(getEnv("TUNAPI_BASE_DOMAIN", "tunapi.local"))
	publicScheme = strings.TrimSpace(getEnv("TUNAPI_PUBLIC_SCHEME", "https"))
	allowedTargets = parseAllowTargets(getEnv("TUNAPI_ALLOWED_TARGETS", "127.0.0.1,localhost"))

	if sharedSecret == "changeme" {
		log.Fatal("TUNAPI_SECRET is using insecure default 'changeme'; set a strong secret")
	}
	if baseDomain == "" {
		log.Fatal("TUNAPI_BASE_DOMAIN is required")
	}
	if publicScheme != "http" && publicScheme != "https" {
		log.Fatal("TUNAPI_PUBLIC_SCHEME must be http or https")
	}

	loadRoutes()

	srv := NewServer()
	serverInstance = srv

	// HTTP API routes
	http.HandleFunc("/register", handleRegisterOrDelete)
	http.HandleFunc("/list", handleList)
	http.HandleFunc("/health", handleHealth)

	// Agent management API
	http.HandleFunc("/agent/create", srv.handleAgentCreate)
	http.HandleFunc("/agent/list", srv.handleAgentList)
	http.HandleFunc("/agent/delete", srv.handleAgentDelete)
	http.HandleFunc("/agent/connect", func(w http.ResponseWriter, r *http.Request) {
		handleAgentConnect(srv, w, r)
	})

	// Publish/unpublish via agent
	http.HandleFunc("/publish", srv.handlePublish)
	http.HandleFunc("/route/", srv.handleUnpublish)

	// Catch-all: proxy by subdomain
	http.HandleFunc("/", handleProxy)

	addr := fmt.Sprintf(":%s", listenPort)
	log.Printf("tunapi listening on %s", addr)
	log.Printf("routes file: %s", routesFile)
	log.Printf("public base domain: %s://*.%s", publicScheme, baseDomain)
	log.Printf("allowed targets: %s", strings.Join(sortedAllowedTargets(), ","))

	srvHTTP := &http.Server{
		Addr:              addr,
		Handler:           nil,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Fatal(srvHTTP.ListenAndServe())
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseAllowTargets(raw string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, p := range strings.Split(raw, ",") {
		s := strings.ToLower(strings.TrimSpace(p))
		if s == "" {
			continue
		}
		out[s] = struct{}{}
	}
	return out
}

func sortedAllowedTargets() []string {
	res := make([]string, 0, len(allowedTargets))
	for k := range allowedTargets {
		res = append(res, k)
	}
	for i := 0; i < len(res); i++ {
		for j := i + 1; j < len(res); j++ {
			if res[j] < res[i] {
				res[i], res[j] = res[j], res[i]
			}
		}
	}
	return res
}

func loadRoutes() {
	routesMu.Lock()
	defer routesMu.Unlock()

	data, err := os.ReadFile(routesFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("warning: cannot read routes file: %v", err)
		}
		routes = Routes{Version: 1, Routes: []types.Route{}}
		return
	}

	if err := json.Unmarshal(data, &routes); err != nil {
		log.Fatalf("invalid routes JSON: %v", err)
	}
	log.Printf("loaded %d routes", len(routes.Routes))
}

func saveRoutes() error {
	routesMu.Lock()
	defer routesMu.Unlock()

	data, err := json.MarshalIndent(routes, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(routesFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(routesFile, data, 0644)
}

func findRoute(subdomain string) *types.Route {
	routesMu.RLock()
	defer routesMu.RUnlock()

	for i := range routes.Routes {
		r := &routes.Routes[i]
		if r.Subdomain == subdomain {
			return r
		}
	}
	return nil
}

func checkSecret(r *http.Request) bool {
	secret := r.Header.Get("X-Secret")
	if secret == "" {
		secret = r.URL.Query().Get("secret")
	}
	return secret == sharedSecret
}

func validateTarget(target string, port int) error {
	t := strings.ToLower(strings.TrimSpace(target))
	if t == "" {
		return fmt.Errorf("target is required")
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("port out of range")
	}

	if _, err := strconv.Atoi(t); err == nil {
		return fmt.Errorf("numeric host without dots is not allowed")
	}

	if len(allowedTargets) > 0 {
		if _, ok := allowedTargets[t]; !ok {
			return fmt.Errorf("target not allowed; set TUNAPI_ALLOWED_TARGETS")
		}
	}

	if ip := net.ParseIP(t); ip != nil {
		if ip.IsMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("invalid target ip")
		}
	}

	return nil
}

func handleRegisterOrDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && r.URL.Path == "/register" {
		handleRegister(w, r)
		return
	}
	if r.Method == http.MethodDelete && r.URL.Path == "/register" {
		subdomain := r.URL.Query().Get("subdomain")
		if subdomain != "" {
			handleDeleteBySubdomain(w, r, subdomain)
			return
		}
		http.Error(w, "subdomain required", http.StatusBadRequest)
		return
	}
	http.NotFound(w, r)
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if !checkSecret(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var req types.Route
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	req.Subdomain = strings.ToLower(strings.TrimSpace(req.Subdomain))
	req.Target = strings.TrimSpace(req.Target)

	// Default to direct mode if not specified
	if req.Mode == "" {
		req.Mode = types.ModeDirect
	}

	if req.Subdomain == "" || req.Target == "" || req.Port == 0 {
		http.Error(w, "subdomain, target, and port are required", http.StatusBadRequest)
		return
	}
	if !subdomainRe.MatchString(req.Subdomain) {
		http.Error(w, "invalid subdomain format", http.StatusBadRequest)
		return
	}
	if req.Mode == types.ModeDirect {
		if err := validateTarget(req.Target, req.Port); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	routesMu.Lock()
	for i := range routes.Routes {
		if routes.Routes[i].Subdomain == req.Subdomain {
			routesMu.Unlock()
			http.Error(w, "subdomain already exists", http.StatusConflict)
			return
		}
	}

	routes.Routes = append(routes.Routes, req)
	routesMu.Unlock()

	if err := saveRoutes(); err != nil {
		http.Error(w, "failed to save routes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fullDomain := fmt.Sprintf("%s.%s", req.Subdomain, baseDomain)
	resp := map[string]interface{}{
		"url":       fmt.Sprintf("%s://%s", publicScheme, fullDomain),
		"subdomain": req.Subdomain,
		"target":    fmt.Sprintf("%s:%d", req.Target, req.Port),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
	log.Printf("registered route: %s -> %s:%d", req.Subdomain, req.Target, req.Port)
}

func handleDeleteBySubdomain(w http.ResponseWriter, r *http.Request, subdomain string) {
	if !checkSecret(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	routesMu.Lock()
	found := false
	newRoutes := make([]types.Route, 0, len(routes.Routes))
	for _, rt := range routes.Routes {
		if rt.Subdomain != subdomain {
			newRoutes = append(newRoutes, rt)
		} else {
			found = true
		}
	}
	routes.Routes = newRoutes
	routesMu.Unlock()

	if !found {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if err := saveRoutes(); err != nil {
		http.Error(w, "failed to save routes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"deleted": subdomain})
	log.Printf("deleted route: %s", subdomain)
}

func handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	if !checkSecret(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	routesMu.RLock()
	defer routesMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(routes)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleProxy(w http.ResponseWriter, req *http.Request) {
	host := req.Host

	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}

	host = strings.ToLower(strings.TrimSpace(host))
	if !strings.HasSuffix(host, "."+baseDomain) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	subdomain := strings.TrimSuffix(host, "."+baseDomain)
	if subdomain == "" || strings.Contains(subdomain, ".") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// skip API paths
	if strings.HasPrefix(req.URL.Path, "/register") ||
		strings.HasPrefix(req.URL.Path, "/list") ||
		strings.HasPrefix(req.URL.Path, "/health") ||
		strings.HasPrefix(req.URL.Path, "/agent") ||
		strings.HasPrefix(req.URL.Path, "/publish") {
		http.NotFound(w, req)
		return
	}

	route := findRoute(subdomain)
	if route == nil {
		http.Error(w, "route not found: "+subdomain, http.StatusNotFound)
		return
	}

	// Route via agent if ModeAgent
	if route.Mode == types.ModeAgent {
		serverInstance.RouteHTTPViaAgent(route.AgentID, route.LocalPort, req.Method, req.URL.Path, dumpHeaders(req), nil, w)
		return
	}

	// Direct mode
	target := fmt.Sprintf("http://%s:%d", route.Target, route.Port)
	proxyURL, err := url.Parse(target)
	if err != nil {
		http.Error(w, "invalid target", http.StatusBadGateway)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(proxyURL)
	proxy.Transport = &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}
	proxy.ErrorHandler = func(rw http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error %s -> %s: %v", subdomain, target, err)
		http.Error(rw, "bad gateway", http.StatusBadGateway)
	}

	req.URL.Host = proxyURL.Host
	req.URL.Scheme = proxyURL.Scheme
	req.Host = proxyURL.Host

	log.Printf("proxying %s -> %s%s", subdomain, target, req.URL.Path)
	proxy.ServeHTTP(w, req)
}

func dumpHeaders(req *http.Request) map[string]string {
	out := make(map[string]string)
	for k, v := range req.Header {
		if len(v) > 0 {
			out[k] = v[0]
		}
	}
	// Override Host
	out["Host"] = req.Host
	return out
}
