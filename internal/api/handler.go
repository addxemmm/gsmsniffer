// Package api exposes the optionally authenticated laboratory management contract.
package api

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"gsmsniffer/internal/lab"
)

//go:embed openapi.json
var OpenAPI []byte

type Options struct {
	Token, Mode, Version, Revision string
	MaxDurationSeconds             int
	StartedAt                      time.Time
}

type Handler struct {
	manager   *lab.Manager
	opts      Options
	tokenHash [32]byte
}
type envelope struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data"`
	RequestID string `json:"request_id"`
}

func New(manager *lab.Manager, opts Options) http.Handler {
	if opts.StartedAt.IsZero() {
		opts.StartedAt = time.Now()
	}
	return &Handler{manager: manager, opts: opts, tokenHash: sha256.Sum256([]byte(opts.Token))}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		http.Error(w, "entropy unavailable", 503)
		return
	}
	id := hex.EncodeToString(raw[:])
	w.Header().Set("X-Request-ID", id)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	send := func(status int, code, message string, data any) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(envelope{code, message, data, id})
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			slog.Info("management request", "request_id", id, "method", auditMethod(r.Method), "route", auditRoute(r.URL.Path), "status", status)
		}
	}
	methodError := func(allow string) {
		w.Header().Set("Allow", allow)
		send(405, "method_not_allowed", "Method not allowed", nil)
	}
	if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
		if r.Method != http.MethodGet {
			methodError("GET")
			return
		}
		send(200, "ok", "Service is ready", map[string]bool{"ready": true})
		return
	}
	if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
		send(404, "not_found", "Route not found", nil)
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || !strings.EqualFold(u.Host, r.Host) || u.Path != "" || u.RawQuery != "" || u.User != nil || u.Fragment != "" {
			send(403, "origin_forbidden", "Cross-origin management requests are disabled", nil)
			return
		}
	}
	if r.URL.Path == "/api/v1/auth" {
		if r.Method != http.MethodGet {
			methodError("GET")
			return
		}
		send(200, "ok", "Authentication configuration", map[string]bool{"required": h.opts.Token != ""})
		return
	}
	if h.opts.Token != "" {
		provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		providedHash := sha256.Sum256([]byte(provided))
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") || subtle.ConstantTimeCompare(providedHash[:], h.tokenHash[:]) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="gsmsniffer"`)
			send(401, "unauthorized", "Valid Bearer token required", nil)
			return
		}
	}
	fail := func(err error) {
		switch {
		case errors.Is(err, lab.ErrInvalid):
			send(400, "invalid_request", err.Error(), nil)
		case errors.Is(err, lab.ErrBusy):
			send(409, "busy", "Stop the active job before this operation", nil)
		case errors.Is(err, lab.ErrNotFound):
			send(404, "not_found", "Job not found", nil)
		case errors.Is(err, lab.ErrUnavailable):
			send(503, "unavailable", "Laboratory runtime prerequisites are unavailable", nil)
		default:
			slog.Error("management failure", "request_id", id, "error", err)
			send(500, "internal_error", "Internal operation failed", nil)
		}
	}
	switch r.URL.Path {
	case "/api/v1/openapi.json":
		if r.Method != http.MethodGet {
			methodError("GET")
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write(OpenAPI)
	case "/api/v1/status":
		if r.Method != http.MethodGet {
			methodError("GET")
			return
		}
		send(200, "ok", "Service status", map[string]any{"mode": h.opts.Mode, "version": h.opts.Version, "revision": h.opts.Revision, "auth_required": h.opts.Token != "", "uptime_seconds": int(time.Since(h.opts.StartedAt).Seconds()), "active_job": h.manager.Active()})
	case "/api/v1/capabilities":
		if r.Method != http.MethodGet {
			methodError("GET")
			return
		}
		bins := map[string]bool{}
		for _, name := range []string{"grgsm_scanner", "grgsm_livemon_headless", "tshark"} {
			_, err := exec.LookPath(name)
			bins[name] = err == nil
		}
		send(200, "ok", "Runtime capabilities; binary availability is not RF validation", map[string]any{"mode": h.opts.Mode, "bands": []string{"GSM900", "DCS1800"}, "kinds": []string{"scan", "capture"}, "capture_modes": []string{"imsi", "sms"}, "max_duration_seconds": h.opts.MaxDurationSeconds, "shielded_ack_required": true, "identities_masked": true, "sms_redacted": true, "binaries": bins, "rf_validated": false})
	case "/api/v1/jobs":
		switch r.Method {
		case http.MethodGet:
			send(200, "ok", "Job history", map[string]any{"items": h.manager.Jobs()})
		case http.MethodPost:
			if strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]) != "application/json" {
				send(415, "unsupported_media_type", "Content-Type must be application/json", nil)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, 4096)
			dec := json.NewDecoder(r.Body)
			dec.DisallowUnknownFields()
			var cfg lab.Config
			if err := dec.Decode(&cfg); err != nil {
				var tooLarge *http.MaxBytesError
				if errors.As(err, &tooLarge) {
					send(413, "payload_too_large", "Request exceeds 4096 bytes", nil)
				} else {
					send(400, "invalid_json", "Expected one JSON object with known fields", nil)
				}
				return
			}
			if err := dec.Decode(new(any)); err != io.EOF {
				var tooLarge *http.MaxBytesError
				if errors.As(err, &tooLarge) {
					send(413, "payload_too_large", "Request exceeds 4096 bytes", nil)
				} else {
					send(400, "invalid_json", "Exactly one JSON object is required", nil)
				}
				return
			}
			if !cfg.ShieldedAck {
				send(400, "invalid_request", "Confirm own test devices inside a shielded room or enclosure", nil)
				return
			}
			job, err := h.manager.Start(cfg)
			if err != nil {
				fail(err)
				return
			}
			w.Header().Set("Location", "/api/v1/jobs/"+job.ID)
			send(202, "accepted", "Laboratory job accepted", job)
		default:
			methodError("GET, POST")
		}
	case "/api/v1/frequencies":
		if r.Method != http.MethodGet {
			methodError("GET")
			return
		}
		if r.URL.RawQuery != "" {
			send(400, "invalid_request", "Frequency catalog does not accept query parameters", nil)
			return
		}
		items := h.manager.Frequencies()
		send(200, "ok", "Scanned frequencies in the current runtime mode", map[string]any{"items": items, "total": len(items)})
	case "/api/v1/observations":
		switch r.Method {
		case http.MethodGet:
			query := r.URL.Query()
			kind := query.Get("kind")
			if kind != "" && kind != "frequencies" && kind != "imsi" && kind != "sms" {
				send(400, "invalid_request", "Unknown observation kind", nil)
				return
			}
			for key, values := range query {
				if (key != "kind" && key != "limit" && key != "offset") || len(values) != 1 {
					send(400, "invalid_request", "Unknown or repeated query parameter", nil)
					return
				}
			}
			limit, offset := 100, 0
			if value, ok := query["limit"]; ok {
				v, err := strconv.Atoi(value[0])
				if err != nil || v < 1 || v > 500 {
					send(400, "invalid_request", "limit must be between 1 and 500", nil)
					return
				}
				limit = v
			}
			if value, ok := query["offset"]; ok {
				v, err := strconv.Atoi(value[0])
				if err != nil || v < 0 {
					send(400, "invalid_request", "offset must be a nonnegative integer", nil)
					return
				}
				offset = v
			}
			items := h.manager.Observations(kind)
			total := len(items)
			from := min(offset, total)
			end := from + min(limit, total-from)
			page := items[from:end]
			if page == nil {
				page = []lab.Observation{}
			}
			send(200, "ok", "Redacted laboratory observations", map[string]any{"items": page, "total": total, "limit": limit, "offset": offset})
		case http.MethodDelete:
			if err := h.manager.Clear(); err != nil {
				fail(err)
				return
			}
			send(200, "ok", "Observations cleared", nil)
		default:
			methodError("GET, DELETE")
		}
	default:
		if strings.HasPrefix(r.URL.Path, "/api/v1/jobs/") {
			jobID := strings.TrimPrefix(r.URL.Path, "/api/v1/jobs/")
			decodedID, decodeErr := hex.DecodeString(jobID)
			if decodeErr != nil || len(decodedID) != 16 || jobID != strings.ToLower(jobID) {
				send(404, "not_found", "Job not found", nil)
				return
			}
			switch r.Method {
			case http.MethodGet:
				job, ok := h.manager.Get(jobID)
				if !ok {
					send(404, "not_found", "Job not found", nil)
					return
				}
				send(200, "ok", "Job status", job)
			case http.MethodDelete:
				job, err := h.manager.Stop(jobID)
				if err != nil {
					fail(err)
					return
				}
				send(200, "ok", "Job stopped", job)
			default:
				methodError("GET, DELETE")
			}
			return
		}
		send(404, "not_found", "Route not found", nil)
	}
}

// Audit only fixed labels, never arbitrary paths, identifiers, bodies or headers.
func auditMethod(method string) string {
	switch method {
	case "GET", "HEAD", "POST", "DELETE", "PUT", "PATCH", "OPTIONS":
		return method
	default:
		return "OTHER"
	}
}
func auditRoute(path string) string {
	switch path {
	case "/healthz", "/readyz", "/api/v1/auth", "/api/v1/status", "/api/v1/capabilities", "/api/v1/jobs", "/api/v1/frequencies", "/api/v1/observations", "/api/v1/openapi.json":
		return path
	default:
		if strings.HasPrefix(path, "/api/v1/jobs/") {
			return "/api/v1/jobs/{id}"
		}
		return "unmatched"
	}
}
