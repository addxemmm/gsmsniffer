package lab

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func scanForCapture(t *testing.T, m *Manager) Frequency {
	t.Helper()
	j, err := m.Start(fixture())
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for len(m.Frequencies()) == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if _, err = m.Stop(j.ID); err != nil {
		t.Fatal(err)
	}
	rows := m.Frequencies()
	if len(rows) == 0 || !rows[0].Selectable {
		t.Fatal("scan produced no selectable frequencies")
	}
	return rows[0]
}

func captureFrom(f Frequency, mode string) Config {
	return Config{Kind: "capture", ScanJobID: f.ScanJobID, Band: f.Band, FrequencyMHz: f.FrequencyMHz, Mode: mode, DurationSeconds: 1, ShieldedAck: true}
}

func TestScanCaptureWorkflowAndRestart(t *testing.T) {
	dir := t.TempDir()
	m := manager(t, Options{DataDir: dir})
	f := scanForCapture(t, m)
	for _, mode := range []string{"imsi", "sms"} {
		j, err := m.Start(captureFrom(f, mode))
		if err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(3 * time.Second)
		for len(m.Observations(mode)) == 0 && time.Now().Before(deadline) {
			time.Sleep(time.Millisecond)
		}
		if _, err = m.Stop(j.ID); err != nil {
			t.Fatal(err)
		}
		obs := m.Observations(mode)
		if len(obs) == 0 || obs[0].JobID != j.ID || obs[0].Band != f.Band || obs[0].Source != "demo" {
			t.Fatalf("missing capture provenance: %+v", obs)
		}
	}
	m.Close()
	r := manager(t, Options{DataDir: dir})
	if rows := r.Frequencies(); len(rows) != 1 || rows[0] != f {
		t.Fatalf("catalog not retained: %+v", rows)
	}
	j, err := r.Start(captureFrom(f, "imsi"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.Stop(j.ID); err != nil {
		t.Fatal(err)
	}
	if err = r.Clear(); err != nil {
		t.Fatal(err)
	}
	if len(r.Frequencies()) != 0 {
		t.Fatal("cleared catalog retained")
	}
	if _, err = r.Start(captureFrom(f, "sms")); !errors.Is(err, ErrInvalid) {
		t.Fatalf("stale selection accepted: %v", err)
	}
}

func TestCaptureProvenanceValidation(t *testing.T) {
	m := manager(t, Options{})
	base := captureFrom(Frequency{ScanJobID: strings.Repeat("1", 32), Band: "GSM900", FrequencyMHz: 935.2}, "imsi")
	if _, err := m.Start(base); !errors.Is(err, ErrInvalid) {
		t.Fatal("capture without scan accepted", err)
	}
	f := scanForCapture(t, m)
	for name, edit := range map[string]func(*Config){
		"missing":   func(c *Config) { c.ScanJobID = "" },
		"unknown":   func(c *Config) { c.ScanJobID = strings.Repeat("1", 32) },
		"malformed": func(c *Config) { c.ScanJobID = "bad-id" },
		"frequency": func(c *Config) { c.FrequencyMHz = 935.4 },
		"band":      func(c *Config) { c.Band = "DCS1800"; c.FrequencyMHz = 1805.2 },
	} {
		t.Run(name, func(t *testing.T) {
			c := captureFrom(f, "sms")
			edit(&c)
			if _, err := m.Start(c); !errors.Is(err, ErrInvalid) {
				t.Fatal("invalid provenance accepted", err)
			}
		})
	}
	c := fixture()
	c.ScanJobID = f.ScanJobID
	if _, err := m.Start(c); !errors.Is(err, ErrInvalid) {
		t.Fatal("scan accepted parent scan", err)
	}
	j, err := m.Start(fixture())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Start(captureFrom(f, "imsi")); !errors.Is(err, ErrBusy) {
		t.Fatal("concurrent capture accepted", err)
	}
	if _, err = m.Stop(j.ID); err != nil {
		t.Fatal(err)
	}
}

func TestFrequencyCatalogFilteringDedupAndState(t *testing.T) {
	m := manager(t, Options{})
	id := strings.Repeat("a", 32)
	m.jobs = []Job{{ID: id, Kind: "scan", State: "running", Config: fixture()}}
	valid := Observation{Kind: "frequencies", JobID: id, Band: "GSM900", Source: "demo", FrequencyMHz: 935.2, Timestamp: "old", PowerDBM: -50}
	m.observations = []Observation{valid}
	latest := valid
	latest.Timestamp, latest.PowerDBM = "new", -40
	m.observations = append(m.observations, latest)
	for _, edit := range []func(*Observation){
		func(o *Observation) { o.JobID = "" },
		func(o *Observation) { o.JobID = strings.Repeat("b", 32) },
		func(o *Observation) { o.Kind = "imsi" },
		func(o *Observation) { o.Source = "shielded" },
		func(o *Observation) { o.Band = "DCS1800" },
		func(o *Observation) { o.FrequencyMHz = 935.3 },
	} {
		o := valid
		edit(&o)
		m.observations = append(m.observations, o)
	}
	rows := m.Frequencies()
	if len(rows) != 1 || rows[0].Selectable || rows[0].Timestamp != "new" || rows[0].PowerDBM != -40 || rows[0].ARFCN != 0 {
		t.Fatalf("catalog filtering: %+v", rows)
	}
	for _, state := range []string{"running", "finished", "cancelled", "failed"} {
		m.jobs[0].State = state
		if m.Frequencies()[0].Selectable != (state == "finished" || state == "cancelled") {
			t.Fatal("wrong selectable state", state)
		}
		if state == "running" || state == "failed" {
			if _, err := m.Start(captureFrom(rows[0], "imsi")); !errors.Is(err, ErrInvalid) {
				t.Fatal("unfinished/failed result accepted", err)
			}
		}
	}
	m.observations = []Observation{valid, latest}
	m.opts.Mode = "shielded"
	if len(m.Frequencies()) != 0 {
		t.Fatal("mixed runtime mode catalog")
	}
	if _, err := m.Start(captureFrom(rows[0], "imsi")); !errors.Is(err, ErrInvalid) {
		t.Fatal("cross-mode selection accepted", err)
	}
	m.opts.Mode = "demo"
	m.jobs = nil
	if len(m.Frequencies()) != 0 {
		t.Fatal("evicted scan remained selectable")
	}
}

func TestLegacyCaptureHistoryLoadsWithoutScan(t *testing.T) {
	dir := t.TempDir()
	c := captureFrom(Frequency{Band: "GSM900", FrequencyMHz: 935.2}, "imsi")
	state := persisted{
		Jobs:         []Job{{ID: strings.Repeat("a", 32), Kind: "capture", State: "finished", Config: c}},
		Observations: []Observation{{Kind: "frequencies", Source: "demo", FrequencyMHz: 935.2}, {Kind: "imsi", Source: "demo", Identity: "999990000000001"}},
	}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "state.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	m := manager(t, Options{DataDir: dir})
	if len(m.Jobs()) != 1 || len(m.Observations("")) != 2 || len(m.Frequencies()) != 0 {
		t.Fatal("legacy history lost or selectable")
	}
	if m.Observations("imsi")[0].Identity != "999**********01" {
		t.Fatal("legacy identity unmasked")
	}
}

func TestFrequencyCatalogDedupScopeAndOrder(t *testing.T) {
	m := manager(t, Options{})
	first, second := strings.Repeat("a", 32), strings.Repeat("b", 32)
	m.jobs = []Job{
		{ID: first, Kind: "scan", State: "finished", Config: fixture()},
		{ID: second, Kind: "scan", State: "cancelled", Config: fixture()},
	}
	m.observations = []Observation{
		{Kind: "frequencies", JobID: first, Band: "GSM900", Source: "demo", FrequencyMHz: 935.2, CellID: "old"},
		{Kind: "frequencies", JobID: first, Band: "GSM900", Source: "demo", FrequencyMHz: 935.4},
		{Kind: "frequencies", JobID: second, Band: "GSM900", Source: "demo", FrequencyMHz: 935.2},
		{Kind: "frequencies", JobID: first, Band: "GSM900", Source: "demo", FrequencyMHz: 935.2, CellID: "new"},
	}
	rows := m.Frequencies()
	if len(rows) != 3 || rows[0].CellID != "new" || rows[0].ScanJobID != first || rows[1].ScanJobID != second || rows[2].FrequencyMHz != 935.4 {
		t.Fatalf("dedup scope/order: %+v", rows)
	}
	// A capture job and a scan with no surviving observations cannot serve as
	// provenance, even when their IDs are otherwise well formed.
	m.jobs[0].Kind = "capture"
	if rows = m.Frequencies(); len(rows) != 1 || rows[0].ScanJobID != second {
		t.Fatalf("capture job exposed as scan: %+v", rows)
	}
	m.observations = nil
	if _, err := m.Start(captureFrom(Frequency{ScanJobID: second, Band: "GSM900", FrequencyMHz: 935.2}, "imsi")); !errors.Is(err, ErrInvalid) {
		t.Fatal("empty scan accepted", err)
	}
}
