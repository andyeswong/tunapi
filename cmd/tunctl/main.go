package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type routeReq struct {
	Subdomain string `json:"subdomain"`
	Target    string `json:"target"`
	Port      int    `json:"port"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	baseURL, secret := loadDefaults()

	switch cmd {
	case "health":
		must(doHealth(baseURL))
	case "list":
		must(doList(baseURL, secret))
	case "register":
		must(doRegister(baseURL, secret, os.Args[2:]))
	case "delete":
		must(doDelete(baseURL, secret, os.Args[2:]))
	case "agent":
		must(doAgent(baseURL, secret, os.Args[2:]))
	case "publish":
		must(doPublish(baseURL, secret, os.Args[2:]))
	case "unpublish":
		must(doUnpublish(baseURL, secret, os.Args[2:]))
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`tunctl - TunAPI admin CLI

Usage:
  tunctl health
  tunctl list
  tunctl register --subdomain demo --target 127.0.0.1 --port 8080
  tunctl delete --subdomain demo

  tunctl agent create --name web01
  tunctl agent list
  tunctl agent delete --name web01

  tunctl publish --subdomain app --agent web01 --local-port 3000
  tunctl unpublish --subdomain app

Environment:
  TUNAPI_URL      default: http://127.0.0.1:8443
  TUNAPI_SECRET   secret for protected endpoints
`)
}

func loadDefaults() (string, string) {
	url := strings.TrimSpace(os.Getenv("TUNAPI_URL"))
	if url == "" {
		url = "http://127.0.0.1:8443"
	}
	secret := strings.TrimSpace(os.Getenv("TUNAPI_SECRET"))
	return strings.TrimRight(url, "/"), secret
}

func client() *http.Client {
	return &http.Client{Timeout: 20 * time.Second}
}

func req(method, url, secret string, body []byte) (*http.Response, error) {
	var r io.Reader
	if len(body) > 0 {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		return nil, err
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	if secret != "" {
		req.Header.Set("X-Secret", secret)
	}
	return client().Do(req)
}

func printResp(resp *http.Response) error {
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	fmt.Println(string(b))
	return nil
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func doHealth(baseURL string) error {
	resp, err := req(http.MethodGet, baseURL+"/health", "", nil)
	if err != nil {
		return err
	}
	return printResp(resp)
}

func doList(baseURL, secret string) error {
	resp, err := req(http.MethodGet, baseURL+"/list", secret, nil)
	if err != nil {
		return err
	}
	return printResp(resp)
}

func doRegister(baseURL, secret string, args []string) error {
	fs := flag.NewFlagSet("register", flag.ExitOnError)
	url := fs.String("url", baseURL, "TunAPI base URL")
	sec := fs.String("secret", secret, "Secret")
	subdomain := fs.String("subdomain", "", "Subdomain")
	target := fs.String("target", "", "Target host")
	port := fs.Int("port", 0, "Target port")
	fs.Parse(args)

	payload, _ := json.Marshal(routeReq{Subdomain: *subdomain, Target: *target, Port: *port})
	resp, err := req(http.MethodPost, strings.TrimRight(*url, "/")+"/register", *sec, payload)
	if err != nil {
		return err
	}
	return printResp(resp)
}

func doDelete(baseURL, secret string, args []string) error {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	url := fs.String("url", baseURL, "TunAPI base URL")
	sec := fs.String("secret", secret, "Secret")
	subdomain := fs.String("subdomain", "", "Subdomain to delete")
	fs.Parse(args)

	u := strings.TrimRight(*url, "/") + "/register?subdomain=" + *subdomain
	resp, err := req(http.MethodDelete, u, *sec, nil)
	if err != nil {
		return err
	}
	return printResp(resp)
}

// ---- agent commands ----

func doAgent(baseURL, secret string, args []string) error {
	if len(args) < 1 {
		agentUsage()
		os.Exit(1)
	}
	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "create":
		return agentCreate(baseURL, secret, subArgs)
	case "list":
		return agentList(baseURL, secret, subArgs)
	case "delete":
		return agentDelete(baseURL, secret, subArgs)
	default:
		agentUsage()
		os.Exit(1)
	}
	return nil
}

func agentUsage() {
	fmt.Println(`tunctl agent create --name <name>
tunctl agent list
tunctl agent delete --name <name>`)
}

func agentCreate(baseURL, secret string, args []string) error {
	fs := flag.NewFlagSet("agent create", flag.ExitOnError)
	url := fs.String("url", baseURL, "TunAPI URL")
	sec := fs.String("secret", secret, "Secret")
	name := fs.String("name", "", "Agent name")
	fs.Parse(args)

	if *name == "" {
		return fmt.Errorf("--name is required")
	}

	payload, _ := json.Marshal(map[string]string{"name": *name})
	resp, err := req(http.MethodPost, strings.TrimRight(*url, "/")+"/agent/create", *sec, payload)
	if err != nil {
		return err
	}
	return printResp(resp)
}

func agentList(baseURL, secret string, args []string) error {
	fs := flag.NewFlagSet("agent list", flag.ExitOnError)
	url := fs.String("url", baseURL, "TunAPI URL")
	sec := fs.String("secret", secret, "Secret")
	fs.Parse(args)

	resp, err := req(http.MethodGet, strings.TrimRight(*url, "/")+"/agent/list", *sec, nil)
	if err != nil {
		return err
	}
	return printResp(resp)
}

func agentDelete(baseURL, secret string, args []string) error {
	fs := flag.NewFlagSet("agent delete", flag.ExitOnError)
	url := fs.String("url", baseURL, "TunAPI URL")
	sec := fs.String("secret", secret, "Secret")
	name := fs.String("name", "", "Agent name")
	fs.Parse(args)

	if *name == "" {
		return fmt.Errorf("--name is required")
	}

	u := strings.TrimRight(*url, "/") + "/agent/delete?name=" + *name
	resp, err := req(http.MethodDelete, u, *sec, nil)
	if err != nil {
		return err
	}
	return printResp(resp)
}

// ---- publish commands ----

func doPublish(baseURL, secret string, args []string) error {
	fs := flag.NewFlagSet("publish", flag.ExitOnError)
	url := fs.String("url", baseURL, "TunAPI URL")
	sec := fs.String("secret", secret, "Secret")
	subdomain := fs.String("subdomain", "", "Subdomain")
	agent := fs.String("agent", "", "Agent name")
	localPort := fs.Int("local-port", 0, "Local port on agent")
	fs.Parse(args)

	if *subdomain == "" || *agent == "" || *localPort == 0 {
		return fmt.Errorf("--subdomain, --agent, and --local-port are required")
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"subdomain": *subdomain,
		"agent":     *agent,
		"localPort": *localPort,
	})
	resp, err := req(http.MethodPost, strings.TrimRight(*url, "/")+"/publish", *sec, payload)
	if err != nil {
		return err
	}
	return printResp(resp)
}

func doUnpublish(baseURL, secret string, args []string) error {
	fs := flag.NewFlagSet("unpublish", flag.ExitOnError)
	url := fs.String("url", baseURL, "TunAPI URL")
	sec := fs.String("secret", secret, "Secret")
	subdomain := fs.String("subdomain", "", "Subdomain")
	fs.Parse(args)

	if *subdomain == "" {
		return fmt.Errorf("--subdomain is required")
	}

	u := strings.TrimRight(*url, "/") + "/publish?subdomain=" + *subdomain
	resp, err := req(http.MethodDelete, u, *sec, nil)
	if err != nil {
		return err
	}
	return printResp(resp)
}
