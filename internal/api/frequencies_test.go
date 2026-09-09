package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gsmsniffer/internal/lab"
)

func TestFrequencyCatalogAuthenticationAndMethods(t *testing.T) {
	for _, token := range []string{"", testToken} {
		h := setupWithToken(t, token)
		w := request(h, "GET", "/api/v1/frequencies", "", "")
		want := 200
		if token != "" {
			want = 401
		}
		if w.Code != want {
			t.Fatalf("auth: %d", w.Code)
		}
		w = request(h, "GET", "/api/v1/frequencies", "", token)
		if w.Code != 200 || !json.Valid(w.Body.Bytes()) {
			t.Fatalf("catalog: %d %s", w.Code, w.Body)
		}
		for _, method := range []string{"POST", "DELETE", "PUT"} {
			w = request(h, method, "/api/v1/frequencies", "", token)
			if w.Code != 405 || w.Header().Get("Allow") != "GET" {
				t.Fatalf("method: %d", w.Code)
			}
		}
		w = request(h, "GET", "/api/v1/frequencies?unknown=1", "", token)
		if w.Code != 400 {
			t.Fatal("query validation missing")
		}
		r := httptest.NewRequest("GET", "http://example.com/api/v1/frequencies", nil)
		r.Header.Set("Origin", "http://evil.invalid")
		r.Header.Set("Authorization", "Bearer "+token)
		w = httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal("catalog cross-origin allowed")
		}
	}
}

func TestScanThenCaptureAPI(t *testing.T) {
	h := setup(t)
	post := func(c lab.Config, status int) lab.Job {
		t.Helper()
		body, _ := json.Marshal(c)
		w := request(h, "POST", "/api/v1/jobs", string(body), testToken)
		if w.Code != status {
			t.Fatalf("job: %d %s", w.Code, w.Body)
		}
		var got struct{ Data lab.Job }
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		return got.Data
	}
	get := func() []lab.Frequency {
		t.Helper()
		w := request(h, "GET", "/api/v1/frequencies", "", testToken)
		var got struct {
			Data struct {
				Items []lab.Frequency `json:"items"`
				Total int             `json:"total"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || w.Code != http.StatusOK || got.Data.Total != len(got.Data.Items) || got.Data.Items == nil {
			t.Fatal("catalog envelope", w.Body)
		}
		return got.Data.Items
	}
	c := lab.Config{Kind: "capture", Band: "GSM900", Mode: "imsi", FrequencyMHz: 935.2, DurationSeconds: 10, ShieldedAck: true}
	post(c, 400)
	scan := post(lab.Config{Kind: "scan", Band: "GSM900", DurationSeconds: 10, ShieldedAck: true}, 202)
	deadline := time.Now().Add(3 * time.Second)
	for len(get()) == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	rows := get()
	if len(rows) != 1 || rows[0].Selectable {
		t.Fatalf("live rows: %+v", rows)
	}
	c.ScanJobID = scan.ID
	post(c, 409)
	w := request(h, "DELETE", "/api/v1/jobs/"+scan.ID, "", testToken)
	if w.Code != 200 || !get()[0].Selectable {
		t.Fatal("stopped scan not selectable")
	}
	for _, mode := range []string{"imsi", "sms"} {
		c.Mode = mode
		j := post(c, 202)
		w = request(h, "DELETE", "/api/v1/jobs/"+j.ID, "", testToken)
		if w.Code != 200 || j.Config.ScanJobID != scan.ID {
			t.Fatal(fmt.Sprintf("capture %s: %d", mode, w.Code))
		}
	}
	w = request(h, "DELETE", "/api/v1/observations", "", testToken)
	if w.Code != 200 || len(get()) != 0 {
		t.Fatal("clear catalog failed")
	}
	post(c, 400)
}
