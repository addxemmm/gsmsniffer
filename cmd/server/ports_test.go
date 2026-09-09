package main

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"gsmsniffer/internal/api"
	"gsmsniffer/internal/lab"
	"gsmsniffer/internal/webui"
)

func TestDefaultPortsAndListenerIsolation(t *testing.T) {
	t.Setenv("GSMSNIFFER_ADDR", "")
	t.Setenv("GSMSNIFFER_API_ADDR", "")
	m, err := lab.New(lab.Options{Mode: "demo", DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	h := api.New(m, api.Options{Token: strings.Repeat("x", 32), Mode: "demo"})
	servers := newServers(h, webui.New())
	if len(servers) != 2 || servers[0].Addr != ":18083" || servers[1].Addr != ":8083" {
		t.Fatalf("wrong listener defaults: %+v", servers)
	}
	for i, s := range servers {
		w := httptest.NewRecorder()
		s.Handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		want := 200
		if i == 1 {
			want = 404
		}
		if w.Code != want {
			t.Fatalf("listener %d root: %d", i, w.Code)
		}
		for _, token := range []string{"", strings.Repeat("x", 32)} {
			r := httptest.NewRequest("GET", "/api/v1/status", nil)
			if token != "" {
				r.Header.Set("Authorization", "Bearer "+token)
			}
			w = httptest.NewRecorder()
			s.Handler.ServeHTTP(w, r)
			want = 401
			if token != "" {
				want = 200
			}
			if w.Code != want {
				t.Fatalf("listener %d authentication: %d", i, w.Code)
			}
		}
	}
	t.Setenv("GSMSNIFFER_ADDR", "127.0.0.1:19000")
	t.Setenv("GSMSNIFFER_API_ADDR", "127.0.0.1:19001")
	servers = newServers(h, webui.New())
	if servers[0].Addr != "127.0.0.1:19000" || servers[1].Addr != "127.0.0.1:19001" {
		t.Fatal("listener overrides ignored")
	}
}

func TestHealthcheckUsesBothConfiguredPorts(t *testing.T) {
	t.Setenv("GSMSNIFFER_ADDR", "")
	t.Setenv("GSMSNIFFER_API_ADDR", "")
	t.Setenv("GSMSNIFFER_HEALTHCHECK_URL", "")
	urls, err := healthcheckURLs()
	want := []string{"http://127.0.0.1:18083/healthz", "http://127.0.0.1:8083/healthz"}
	if err != nil || !reflect.DeepEqual(urls, want) {
		t.Fatalf("urls=%v err=%v", urls, err)
	}
	t.Setenv("GSMSNIFFER_ADDR", "0.0.0.0:19000")
	t.Setenv("GSMSNIFFER_API_ADDR", "[::]:19001")
	urls, err = healthcheckURLs()
	want = []string{"http://127.0.0.1:19000/healthz", "http://[::1]:19001/healthz"}
	if err != nil || !reflect.DeepEqual(urls, want) {
		t.Fatalf("urls=%v err=%v", urls, err)
	}
	t.Setenv("GSMSNIFFER_ADDR", "invalid")
	if _, err = healthcheckURLs(); err == nil {
		t.Fatal("invalid address accepted")
	}
	t.Setenv("GSMSNIFFER_HEALTHCHECK_URL", "http://127.0.0.1:19999/readyz")
	urls, err = healthcheckURLs()
	if err != nil || len(urls) != 1 || urls[0] != "http://127.0.0.1:19999/readyz" {
		t.Fatal("override ignored")
	}
}

func TestHealthcheckFailsWhenBackendFails(t *testing.T) {
	t.Setenv("GSMSNIFFER_HEALTHCHECK_URL", "")
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	defer web.Close()
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(503) }))
	defer apiServer.Close()
	t.Setenv("GSMSNIFFER_ADDR", strings.TrimPrefix(web.URL, "http://"))
	t.Setenv("GSMSNIFFER_API_ADDR", strings.TrimPrefix(apiServer.URL, "http://"))
	if err := checkHealth(); err == nil {
		t.Fatal("backend failure hidden by healthy web listener")
	}
	apiServer.Close()
	if err := checkHealth(); err == nil {
		t.Fatal("closed backend considered healthy")
	}
	t.Setenv("GSMSNIFFER_API_ADDR", strings.TrimPrefix(web.URL, "http://"))
	if err := checkHealth(); err != nil {
		t.Fatal(err)
	}
}
