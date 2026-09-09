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

	"gsmsniffer/internal/api"
	"gsmsniffer/internal/lab"
	"gsmsniffer/internal/webui"
)

var version = "2.0.0"
var revision = "development"

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
			client := http.Client{Timeout: 3 * time.Second}
			res, err := client.Get(env("GSMSNIFFER_HEALTHCHECK_URL", "http://127.0.0.1:8080/healthz"))
			if err != nil {
				os.Exit(1)
			}
			_ = res.Body.Close()
			if res.StatusCode != 200 {
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

func run() error {
	token := strings.TrimSpace(os.Getenv("GSMSNIFFER_TOKEN"))
	if path := os.Getenv("GSMSNIFFER_TOKEN_FILE"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read token file: %w", err)
		}
		token = strings.TrimSpace(string(data))
	}
	if len(token) < 32 || strings.ContainsAny(token, " \r\n\t") {
		return errors.New("configure GSMSNIFFER_TOKEN_FILE or GSMSNIFFER_TOKEN with at least 32 non-whitespace characters")
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
	ui := webui.New()
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
	servers := []*http.Server{server(env("GSMSNIFFER_ADDR", ":8080"), web)}
	if addr := os.Getenv("GSMSNIFFER_API_ADDR"); addr != "" {
		servers = append(servers, server(addr, apiHandler))
	}
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
