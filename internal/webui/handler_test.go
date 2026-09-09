package webui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAssets(t *testing.T) {
	for _, tc := range []struct{ path, contentType string }{{"/", "text/html"}, {"/app.css", "text/css"}, {"/app.js", "text/javascript"}} {
		t.Run(tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			New().ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if w.Code != http.StatusOK || w.Body.Len() == 0 || !strings.HasPrefix(w.Header().Get("Content-Type"), tc.contentType) {
				t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
			}
			if w.Header().Get("Content-Security-Policy") == "" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatal("missing security headers")
			}
		})
	}
}

func TestUnknownAndMethods(t *testing.T) {
	for _, path := range []string{"/unknown", "/static/index.html", "/../handler.go", "/api/v1/status"} {
		w := httptest.NewRecorder()
		New().ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != http.StatusNotFound {
			t.Errorf("%s: got %d", path, w.Code)
		}
	}
	w := httptest.NewRecorder()
	New().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST: got %d", w.Code)
	}
	w = httptest.NewRecorder()
	New().ServeHTTP(w, httptest.NewRequest(http.MethodHead, "/", nil))
	if w.Code != http.StatusOK || w.Body.Len() != 0 {
		t.Error("HEAD must return OK without a body")
	}
}

func TestSelfContainedConsole(t *testing.T) {
	html, err := assets.ReadFile("static/index.html")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(html), `name="shielded_ack" required`) != 2 {
		t.Fatal("each start form must require explicit shielded acknowledgement")
	}
	js, err := assets.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"localStorage", "sessionStorage", "innerHTML", "document.write", "eval("} {
		if strings.Contains(string(js), forbidden) {
			t.Errorf("console must not use %s", forbidden)
		}
	}
	if strings.Contains(string(html), "https://") || strings.Contains(string(html), "http://") {
		t.Fatal("console assets must be self-contained")
	}
}
