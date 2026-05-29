package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"local-router/internal/router"
)

type Config struct {
	BaseURL string
	Stdout  io.Writer
	Stderr  io.Writer
}

func Run(args []string, cfg Config) int {
	if cfg.Stdout == nil {
		cfg.Stdout = os.Stdout
	}
	if cfg.Stderr == nil {
		cfg.Stderr = os.Stderr
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = envOrDefault("LOCAL_ROUTER_URL", "http://127.0.0.1")
	}
	if len(args) == 0 || isHelpArg(args[0]) {
		printHelp(cfg.Stdout)
		return 0
	}
	if len(args) > 1 && isHelpArg(args[1]) {
		if printCommandHelp(cfg.Stdout, args[0]) {
			return 0
		}
		printHelp(cfg.Stdout)
		return 2
	}
	client := &Client{BaseURL: strings.TrimRight(cfg.BaseURL, "/"), HTTP: &http.Client{Timeout: 5 * time.Second}}
	var err error
	switch args[0] {
	case "serve":
		printCommandHelp(cfg.Stdout, "serve")
		return 0
	case "status":
		err = status(client, cfg.Stdout)
	case "register":
		err = register(client, args[1:], cfg.Stdout)
	case "unregister":
		err = unregister(client, args[1:], cfg.Stdout)
	case "routes":
		err = routes(client, cfg.Stdout)
	case "pin":
		err = setPinned(client, args[1:], true, cfg.Stdout)
	case "unpin":
		err = setPinned(client, args[1:], false, cfg.Stdout)
	default:
		fmt.Fprintf(cfg.Stderr, "unknown command %q\n\n", args[0])
		printHelp(cfg.Stderr)
		return 2
	}
	if err != nil {
		fmt.Fprintln(cfg.Stderr, friendlyError(err))
		return 1
	}
	return 0
}

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

func (c *Client) List() ([]router.RouteView, error) {
	var routes []router.RouteView
	return routes, c.do(http.MethodGet, "/router/routes", nil, &routes)
}

func (c *Client) Register(name string, req router.RegisterRequest, force bool) (router.RouteView, error) {
	path := "/router/routes/" + url.PathEscape(name)
	if force {
		path += "?force=true"
	}
	var route router.RouteView
	return route, c.do(http.MethodPut, path, req, &route)
}

func (c *Client) Delete(name string) error {
	return c.do(http.MethodDelete, "/router/routes/"+url.PathEscape(name), nil, nil)
}

func (c *Client) SetPinned(name string, pinned bool) (router.RouteView, error) {
	var route router.RouteView
	return route, c.do(http.MethodPatch, "/router/routes/"+url.PathEscape(name), map[string]bool{"pinned": pinned}, &route)
}

func (c *Client) do(method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		buf := new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			return err
		}
		reader = buf
	}
	req, err := http.NewRequest(method, c.BaseURL+path, reader)
	if err != nil {
		return err
	}
	req.Host = router.PrimaryControlHost
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return errRouterNotRunning
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		if apiErr.Error == "" {
			apiErr.Error = resp.Status
		}
		return fmt.Errorf("%s", apiErr.Error)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

var errRouterNotRunning = errors.New("router service is not running")

func status(client *Client, out io.Writer) error {
	routes, err := client.List()
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "local-router is running (%d routes)\n", len(routes))
	return nil
}

func register(client *Client, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("register requires a route name")
	}
	name := args[0]
	req := router.RegisterRequest{}
	force := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--port":
			i++
			if i >= len(args) {
				return errors.New("--port requires a value")
			}
			port, err := strconv.Atoi(args[i])
			if err != nil {
				return errors.New("--port must be a number")
			}
			req.Port = port
		case "--title":
			i++
			if i >= len(args) {
				return errors.New("--title requires a value")
			}
			req.Title = args[i]
		case "--target-host":
			i++
			if i >= len(args) {
				return errors.New("--target-host requires a value")
			}
			req.TargetHost = args[i]
		case "--exec":
			i++
			if i >= len(args) {
				return errors.New("--exec requires a value")
			}
			req.Exec = args[i]
		case "--heartbeat-path":
			i++
			if i >= len(args) {
				return errors.New("--heartbeat-path requires a value")
			}
			req.HeartbeatPath = args[i]
		case "--pinned":
			req.Pinned = true
		case "--force":
			force = true
		default:
			return fmt.Errorf("unknown option %s", args[i])
		}
	}
	if req.Port == 0 {
		return errors.New("register requires --port")
	}
	view, err := client.Register(name, req, force)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, view.URL)
	return nil
}

func unregister(client *Client, args []string, out io.Writer) error {
	if len(args) != 1 {
		return errors.New("unregister requires a route name")
	}
	if err := client.Delete(args[0]); err != nil {
		return err
	}
	fmt.Fprintf(out, "unregistered %s\n", args[0])
	return nil
}

func routes(client *Client, out io.Writer) error {
	routes, err := client.List()
	if err != nil {
		return err
	}
	if len(routes) == 0 {
		fmt.Fprintln(out, "no routes registered")
		return nil
	}
	for _, route := range routes {
		fmt.Fprintf(out, "%s\t%s\t%s\t%s:%d\n", route.Name, route.URL, route.Status, route.TargetHost, route.Port)
	}
	return nil
}

func setPinned(client *Client, args []string, pinned bool, out io.Writer) error {
	if len(args) != 1 {
		return errors.New("pin/unpin requires a route name")
	}
	view, err := client.SetPinned(args[0], pinned)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s pinned=%t\n", view.Name, view.Pinned)
	return nil
}

func friendlyError(err error) string {
	if errors.Is(err, errRouterNotRunning) {
		return "local-router service is not running; start it with `local-router serve`"
	}
	return err.Error()
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
