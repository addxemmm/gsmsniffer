package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"

	"gsmsniffer/internal/api"
	"gsmsniffer/internal/lab"
	"gsmsniffer/internal/webui"
)

var version = "2.1"
var revision = "development"

const defaultWebAddress = ":18083"
const defaultAPIAddress = ":8083"

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Printf("gsmsniffer %s (%s)\n", version, revision)
			return
		case "healthcheck":
			if err := checkHealth(); err != nil {
				os.Exit(1)
			}
			return
		default:
			fmt.Fprintln(os.Stderr, "usage: gsmsniffer [version|healthcheck]")
			os.Exit(2)
		}
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

// loadToken disables authentication only when neither token source is configured.
// An explicitly configured file must remain valid, even if the environment token is valid.
func loadToken() (string, error) {
	raw := os.Getenv("GSMSNIFFER_TOKEN")
	configured := raw != ""
	if path := os.Getenv("GSMSNIFFER_TOKEN_FILE"); path != "" {
		configured = true
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read token file: %w", err)
		}
		raw = string(data)
	}
	if !configured {
		return "", nil
	}
	token := strings.TrimSpace(raw)
	if len(token) < 32 || strings.ContainsFunc(token, unicode.IsSpace) {
		return "", errors.New("configured GSMSNIFFER_TOKEN_FILE or GSMSNIFFER_TOKEN must contain at least 32 non-whitespace characters")
	}
	return token, nil
}

func run() error {
	token, err := loadToken()
	if err != nil {
		return err
	}
	if token == "" {
		slog.Warn("management authentication disabled; all clients with network access can manage this service; use only on a trusted network")
	}
	maxDuration, err := strconv.Atoi(env("GSMSNIFFER_MAX_DURATION_SECONDS", "300"))
	if err != nil || maxDuration < 1 || maxDuration > 3600 {
		return errors.New("GSMSNIFFER_MAX_DURATION_SECONDS must be 1..3600")
	}
	mode := env("GSMSNIFFER_MODE", "demo")
	manager, err := lab.New(lab.Options{Mode: mode, DataDir: env("GSMSNIFFER_DATA_DIR", "./data"), MaxDurationSeconds: maxDuration})
	if err != nil {
		return err
	}
	defer manager.Close()
	apiHandler := api.New(manager, api.Options{Token: token, Mode: mode, Version: version, Revision: revision, MaxDurationSeconds: maxDuration})
	servers := newServers(apiHandler, webui.New())
	listeners := []net.Listener{}
	defer func() {
		for _, l := range listeners {
			_ = l.Close()
		}
	}()
	for _, s := range servers {
		l, err := net.Listen("tcp", s.Addr)
		if err != nil {
			return err
		}
		listeners = append(listeners, l)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errorsCh := make(chan error, len(servers))
	for i, s := range servers {
		go func(s *http.Server, l net.Listener) { errorsCh <- s.Serve(l) }(s, listeners[i])
		slog.Info("management listener started", "address", s.Addr, "mode", mode, "version", version, "revision", revision)
	}
	var result error
	select {
	case <-ctx.Done():
	case result = <-errorsCh:
		if errors.Is(result, http.ErrServerClosed) {
			result = nil
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, s := range servers {
		if err := s.Shutdown(shutdownCtx); err != nil {
			_ = s.Close()
		}
	}
	return result
}

func newServers(apiHandler, ui http.Handler) []*http.Server {
	web := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			apiHandler.ServeHTTP(w, r)
			return
		}
		ui.ServeHTTP(w, r)
	})
	server := func(addr string, handler http.Handler) *http.Server {
		return &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024}
	}
	return []*http.Server{
		server(env("GSMSNIFFER_ADDR", defaultWebAddress), web),
		server(env("GSMSNIFFER_API_ADDR", defaultAPIAddress), apiHandler),
	}
}

func healthcheckURLs() ([]string, error) {
	if override := os.Getenv("GSMSNIFFER_HEALTHCHECK_URL"); override != "" {
		return []string{override}, nil
	}
	urls := make([]string, 0, 2)
	for _, addr := range []string{env("GSMSNIFFER_ADDR", defaultWebAddress), env("GSMSNIFFER_API_ADDR", defaultAPIAddress)} {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid healthcheck listener address: %w", err)
		}
		if host == "" || host == "0.0.0.0" {
			host = "127.0.0.1"
		} else if host == "::" {
			host = "::1"
		}
		urls = append(urls, "http://"+net.JoinHostPort(host, port)+"/healthz")
	}
	return urls, nil
}

func checkHealth() error {
	urls, err := healthcheckURLs()
	if err != nil {
		return err
	}
	client := http.Client{Timeout: 3 * time.Second}
	for _, target := range urls {
		res, err := client.Get(target)
		if err != nil {
			return err
		}
		_ = res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("listener healthcheck returned %d", res.StatusCode)
		}
	}
	return nil
}
