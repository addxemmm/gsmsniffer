package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"gsmsniffer/internal/lab"
)

const testToken = "synthetic-test-token-not-for-deployment-0000"

func setup(t *testing.T) http.Handler {
	t.Helper()
	return setupWithToken(t, testToken)
}

func setupWithToken(t *testing.T, token string) http.Handler {
	t.Helper()
	m, err := lab.New(lab.Options{Mode: "demo", DataDir: t.TempDir(), MaxDurationSeconds: 10})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.Close)
	return New(m, Options{Token: token, Mode: "demo", Version: "test", MaxDurationSeconds: 10})
}

func TestAuthenticationDiscovery(t *testing.T) {
	for _, token := range []string{"", testToken} {
		h := setupWithToken(t, token)
		w := request(h, "GET", "/api/v1/auth", "", "")
		var got struct {
			Data map[string]bool `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || len(got.Data) != 1 || got.Data["required"] != (token != "") || strings.Contains(w.Body.String(), testToken) {
			t.Fatalf("unexpected authentication discovery: %d %s", w.Code, w.Body)
		}
		for _, method := range []string{"POST", "PUT", "DELETE", "HEAD", "OPTIONS"} {
			w = request(h, method, "/api/v1/auth", "", "")
			if w.Code != 405 || w.Header().Get("Allow") != "GET" {
				t.Fatalf("auth method %s: %d", method, w.Code)
			}
		}
		r := httptest.NewRequest("GET", "http://example.com/api/v1/auth", nil)
		r.Header.Set("Origin", "http://evil.invalid")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatalf("cross-origin discovery: %d", w.Code)
		}
	}
}

func TestOptionalAuthenticationPreservesValidation(t *testing.T) {
	h := setupWithToken(t, "")
	for _, path := range []string{"/api/v1/status", "/api/v1/jobs", "/api/v1/capabilities", "/api/v1/observations", "/api/v1/openapi.json"} {
		w := request(h, "GET", path, "", "")
		if w.Code != 200 || w.Header().Get("WWW-Authenticate") != "" {
			t.Fatalf("anonymous %s: %d %s", path, w.Code, w.Body)
		}
	}
	w := request(h, "GET", "/api/v1/status", "", "")
	if !strings.Contains(w.Body.String(), `"auth_required":false`) {
		t.Fatal("missing disabled auth status")
	}
	for _, body := range []string{`{}`, `{"kind":"scan","band":"GSM900","duration_seconds":1,"shielded_ack":false}`, `{"unknown":true}`} {
		w = request(h, "POST", "/api/v1/jobs", body, "")
		if w.Code != 400 {
			t.Fatalf("anonymous invalid job: %d %s", w.Code, w.Body)
		}
	}
	for _, method := range []string{"GET", "POST", "DELETE"} {
		r := httptest.NewRequest(method, "http://example.com/api/v1/jobs", strings.NewReader(`{}`))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", "http://evil.invalid")
		w = httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatalf("anonymous cross-origin %s: %d", method, w.Code)
		}
	}
	w = request(h, "POST", "/api/v1/jobs", `{"kind":"scan","band":"GSM900","duration_seconds":1,"shielded_ack":true}`, "")
	if w.Code != 202 {
		t.Fatalf("anonymous valid demo job: %d %s", w.Code, w.Body)
	}
	var got struct {
		Data lab.Job `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	w = request(h, "DELETE", "/api/v1/jobs/"+got.Data.ID, "", "")
	if w.Code != 200 {
		t.Fatalf("anonymous stop: %d %s", w.Code, w.Body)
	}
}
func request(h http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestAuthenticationAndMethods(t *testing.T) {
	h := setup(t)
	for _, tt := range []struct {
		method, path, token string
		status              int
	}{{"GET", "/healthz", "", 200}, {"GET", "/readyz", "", 200}, {"POST", "/healthz", "", 405}, {"GET", "/api/v1/status", "", 401}, {"GET", "/api/v1/status", "wrong", 401}, {"GET", "/api/v1/status", testToken, 200}, {"POST", "/api/v1/status", testToken, 405}, {"GET", "/scanner", testToken, 404}, {"GET", "/api/v1/unknown", testToken, 404}, {"GET", "/api/v1/jobs/missing", testToken, 404}, {"GET", "/api/v1/jobs/", testToken, 404}} {
		t.Run(tt.method+tt.path+tt.token, func(t *testing.T) {
			w := request(h, tt.method, tt.path, "", tt.token)
			if w.Code != tt.status {
				t.Fatalf("status=%d body=%s", w.Code, w.Body)
			}
			var got envelope
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if len(got.RequestID) != 32 || w.Header().Get("X-Request-ID") != got.RequestID {
				t.Fatal("missing request ID")
			}
			if strings.Contains(w.Body.String(), testToken) {
				t.Fatal("token leaked")
			}
		})
	}
}
func TestStrictRequests(t *testing.T) {
	h := setup(t)
	for _, body := range []string{`null`, `[]`, `{}`, `{"kind":"scan","band":"GSM900","duration_seconds":1,"shielded_ack":false}`, `{"unknown":1}`, `{} {}`, `{"kind":"scan","band":"GSM900","duration_seconds":1,"shielded_ack":true,"mode":"sms"}`} {
		w := request(h, "POST", "/api/v1/jobs", body, testToken)
		if w.Code != 400 {
			t.Fatalf("body=%s status=%d", body, w.Code)
		}
	}
	w := request(h, "POST", "/api/v1/jobs", `{"kind":"`+strings.Repeat("a", 5000)+`"}`, testToken)
	if w.Code != 413 {
		t.Fatalf("oversize=%d", w.Code)
	}
	w = request(h, "POST", "/api/v1/jobs", "", testToken)
	if w.Code != 415 {
		t.Fatalf("content type=%d", w.Code)
	}
	for _, query := range []string{"limit=0", "limit=501", "limit=-1", "limit=abc", "offset=-1", "offset=9999999999999999999999", "kind=raw", "limit=2&limit=3", "arbitrary=1", "limit="} {
		w = request(h, "GET", "/api/v1/observations?"+query, "", testToken)
		if w.Code != 400 {
			t.Fatalf("query=%s status=%d", query, w.Code)
		}
	}
	w = request(h, "GET", "/api/v1/observations?offset=9223372036854775807", "", testToken)
	if w.Code != 200 {
		t.Fatalf("large offset=%d", w.Code)
	}
}
func TestJobLifecycle(t *testing.T) {
	h := setup(t)
	body := `{"kind":"scan","band":"GSM900","duration_seconds":10,"shielded_ack":true}`
	w := request(h, "POST", "/api/v1/jobs", body, testToken)
	if w.Code != 202 {
		t.Fatalf("start: %d %s", w.Code, w.Body)
	}
	var got struct {
		Data lab.Job `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/jobs/" + got.Data.ID
	if w.Header().Get("Location") != path {
		t.Fatal("missing location")
	}
	for _, tt := range []struct {
		method, path, body string
		status             int
	}{{"POST", "/api/v1/jobs", body, 409}, {"DELETE", "/api/v1/observations", "", 409}, {"GET", path, "", 200}, {"GET", "/api/v1/jobs", "", 200}, {"DELETE", path, "", 200}, {"DELETE", path, "", 200}, {"DELETE", "/api/v1/observations", "", 200}} {
		w = request(h, tt.method, tt.path, tt.body, testToken)
		if w.Code != tt.status {
			t.Fatalf("%s %s: %d %s", tt.method, tt.path, w.Code, w.Body)
		}
	}
}
func TestOriginPolicy(t *testing.T) {
	h := setup(t)
	for _, origin := range []string{"https://evil.invalid", "null", "http://example.com@evil.invalid", "http://example.com/path"} {
		r := httptest.NewRequest("GET", "http://example.com/api/v1/status", nil)
		r.Header.Set("Origin", origin)
		r.Header.Set("Authorization", "Bearer "+testToken)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatalf("origin %s: %d", origin, w.Code)
		}
	}
	r := httptest.NewRequest("GET", "http://example.com/api/v1/status", nil)
	r.Header.Set("Origin", "http://example.com")
	r.Header.Set("Authorization", "Bearer "+testToken)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
}
func TestSpecificationAndContract(t *testing.T) {
	doc, err := os.ReadFile("../../docs/api/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(doc, OpenAPI) {
		t.Fatal("published and embedded OpenAPI differ")
	}
	var spec struct {
		Openapi string
		Paths   map[string]map[string]json.RawMessage
	}
	if err := json.Unmarshal(OpenAPI, &spec); err != nil {
		t.Fatal(err)
	}
	if spec.Openapi != "3.0.3" {
		t.Fatal(spec.Openapi)
	}
	h := setup(t)
	for _, path := range []string{"/api/v1/status", "/api/v1/capabilities", "/api/v1/jobs", "/api/v1/observations", "/api/v1/openapi.json", "/healthz", "/readyz"} {
		if _, ok := spec.Paths[path]["get"]; !ok {
			t.Fatal("missing contract", path)
		}
		w := request(h, "GET", path, "", testToken)
		if w.Code != 200 {
			t.Fatal(fmt.Sprintf("%s: %d", path, w.Code))
		}
	}
	w := request(h, "GET", "/api/v1/openapi.json", "", testToken)
	if !bytes.Equal(w.Body.Bytes(), OpenAPI) {
		t.Fatal("spec response differs")
	}
}
