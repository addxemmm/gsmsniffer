package lab

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func fixture() Config {
	return Config{Kind: "scan", Band: "GSM900", DurationSeconds: 1, ShieldedAck: true}
}
func manager(t *testing.T, opts Options) *Manager {
	t.Helper()
	m, e := New(opts)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(m.Close)
	return m
}

func TestValidation(t *testing.T) {
	m := manager(t, Options{})
	for _, frequency := range []float64{925.2, 935.0, 959.8} {
		c := fixture()
		c.Kind = "capture"
		c.Mode = "imsi"
		c.FrequencyMHz = frequency
		if err := m.validate(c); err != nil {
			t.Fatal(err)
		}
	}
	cases := map[string]func(*Config){"kind": func(c *Config) { c.Kind = "scan;sh" }, "band": func(c *Config) { c.Band = "900;id" }, "zero_duration": func(c *Config) { c.DurationSeconds = 0 }, "long_duration": func(c *Config) { c.DurationSeconds = 301 }, "ack": func(c *Config) { c.ShieldedAck = false }, "nan": func(c *Config) { c.FrequencyMHz = math.NaN() }, "inf": func(c *Config) { c.FrequencyMHz = math.Inf(1) }, "scan_frequency": func(c *Config) { c.FrequencyMHz = 935.2 }, "scan_mode": func(c *Config) { c.Mode = "imsi" }, "mode": func(c *Config) { c.Kind = "capture"; c.FrequencyMHz = 935.2; c.Mode = "exec" }, "range": func(c *Config) { c.Kind = "capture"; c.Mode = "imsi"; c.FrequencyMHz = 100 }, "channel": func(c *Config) { c.Kind = "capture"; c.Mode = "imsi"; c.FrequencyMHz = 935.3 }}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) {
			c := fixture()
			edit(&c)
			if _, e := m.Start(c); !errors.Is(e, ErrInvalid) {
				t.Fatalf("want invalid, got %v", e)
			}
		})
	}
	for _, opts := range []Options{{Mode: "live"}, {MaxDurationSeconds: -1}, {MaxDurationSeconds: 3601}} {
		if _, e := New(opts); !errors.Is(e, ErrInvalid) {
			t.Errorf("expected invalid options: %v", e)
		}
	}
	for _, band := range []string{"GSM900", "DCS1800"} {
		for _, mode := range []string{"imsi", "sms"} {
			c := fixture()
			c.Kind = "capture"
			c.Band = band
			c.Mode = mode
			c.FrequencyMHz = 935.2
			if band == "DCS1800" {
				c.FrequencyMHz = 1805.2
			}
			if e := m.validate(c); e != nil {
				t.Fatal(e)
			}
		}
	}
}
func TestConcurrencyAndStop(t *testing.T) {
	m := manager(t, Options{})
	var success atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := m.Start(fixture())
			if e == nil {
				success.Add(1)
			} else if !errors.Is(e, ErrBusy) {
				t.Errorf("start: %v", e)
			}
		}()
	}
	wg.Wait()
	if success.Load() != 1 {
		t.Fatalf("started %d jobs", success.Load())
	}
	active := m.Active()
	if active == nil || active.State != "running" {
		t.Fatal("missing running job")
	}
	if !errors.Is(m.Clear(), ErrBusy) {
		t.Fatal("clear should reject active task")
	}
	j, e := m.Stop(active.ID)
	if e != nil || j.State != "cancelled" || j.EndedAt == nil {
		t.Fatalf("stop: %+v %v", j, e)
	}
	if m.Active() != nil {
		t.Fatal("active task not cleared")
	}
	if _, e = m.Stop(j.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = m.Stop("missing"); !errors.Is(e, ErrNotFound) {
		t.Fatal(e)
	}
	if _, ok := m.Get("missing"); ok {
		t.Fatal("unexpected job")
	}
}

func TestDeadlineAndPrivacy(t *testing.T) {
	for _, mode := range []string{"imsi", "sms"} {
		t.Run(mode, func(t *testing.T) {
			m := manager(t, Options{DataDir: t.TempDir()})
			c := fixture()
			c.Kind = "capture"
			c.Mode = mode
			c.FrequencyMHz = 935.2
			j, e := m.Start(c)
			if e != nil {
				t.Fatal(e)
			}
			deadline := time.Now().Add(3 * time.Second)
			for m.Active() != nil && time.Now().Before(deadline) {
				time.Sleep(10 * time.Millisecond)
			}
			j, _ = m.Get(j.ID)
			if j.State != "finished" || j.EndedAt == nil {
				t.Fatalf("state %+v", j)
			}
			obs := m.Observations(mode)
			if len(obs) == 0 {
				t.Fatal("missing demo data")
			}
			for _, o := range obs {
				if o.Source != "demo" {
					t.Fatal("demo source missing")
				}
				if mode == "imsi" && o.Identity != "999**********01" {
					t.Fatalf("unmasked identity %q", o.Identity)
				}
				if mode == "sms" && o.Text != "[redacted]" {
					t.Fatal("unredacted SMS")
				}
			}
			b, e := os.ReadFile(filepath.Join(m.opts.DataDir, "state.json"))
			if e != nil {
				t.Fatal(e)
			}
			if strings.Contains(string(b), "999990000000001") || strings.Contains(string(b), "SYNTHETIC LAB DEMO") {
				t.Fatal("raw data persisted")
			}
			if runtime.GOOS != "windows" {
				info, _ := os.Stat(filepath.Join(m.opts.DataDir, "state.json"))
				if info.Mode().Perm() != 0600 {
					t.Fatalf("mode %v", info.Mode())
				}
			}
			if e = m.Clear(); e != nil {
				t.Fatal(e)
			}
			if len(m.Observations("")) != 0 {
				t.Fatal("clear failed")
			}
		})
	}
}
func TestHistoryLimitsAndRestore(t *testing.T) {
	dir := t.TempDir()
	m := manager(t, Options{DataDir: dir})
	for i := 0; i < maxJobs+3; i++ {
		j, e := m.Start(fixture())
		if e != nil {
			t.Fatal(e)
		}
		if _, e = m.Stop(j.ID); e != nil {
			t.Fatal(e)
		}
	}
	if len(m.Jobs()) != maxJobs {
		t.Fatal("unbounded jobs")
	}
	for i := 0; i < maxObservations+3; i++ {
		m.observe(Observation{Kind: "imsi", Identity: "999990000000001", Timestamp: fmt.Sprint(i)})
	}
	if len(m.Observations("")) != maxObservations {
		t.Fatal("unbounded observations")
	}
	if m.Observations("")[0].Timestamp != "3" {
		t.Fatal("old observations retained")
	}
	m.mu.Lock()
	if e := m.persistLocked(); e != nil {
		t.Fatal(e)
	}
	m.mu.Unlock()
	m.Close()
	restored := manager(t, Options{DataDir: dir})
	if len(restored.Jobs()) != maxJobs || len(restored.Observations("")) != maxObservations {
		t.Fatal("restore counts differ")
	}
	if restored.Observations("")[0].Identity != "999**********01" {
		t.Fatal("restore unmasked data")
	}
	j := restored.Jobs()[0]
	original := *j.EndedAt
	*j.EndedAt = time.Time{}
	actual, _ := restored.Get(j.ID)
	if !actual.EndedAt.Equal(original) {
		t.Fatal("returned time pointer mutates store")
	}
}
func TestRecoveryAndCorruptState(t *testing.T) {
	dir := t.TempDir()
	m := manager(t, Options{DataDir: dir})
	j, e := m.Start(fixture())
	if e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(filepath.Join(dir, "state.json"))
	if e != nil {
		t.Fatal(e)
	}
	m.Close()
	if e = os.WriteFile(filepath.Join(dir, "state.json"), b, 0600); e != nil {
		t.Fatal(e)
	}
	r := manager(t, Options{DataDir: dir})
	recovered, _ := r.Get(j.ID)
	if recovered.State != "failed" || recovered.EndedAt == nil {
		t.Fatal("interrupted job not recovered as failed")
	}
	r.Close()
	for _, bad := range []string{"broken", `{"jobs":[]} trailing`, strings.Repeat("x", maxStateBytes+1)} {
		if e = os.WriteFile(filepath.Join(dir, "state.json"), []byte(bad), 0600); e != nil {
			t.Fatal(e)
		}
		if _, e = New(Options{DataDir: dir}); e == nil {
			t.Fatal("accepted corrupt state")
		}
	}
}
func TestCloseUnavailableAndPersistenceFailure(t *testing.T) {
	m := manager(t, Options{})
	j, e := m.Start(fixture())
	if e != nil {
		t.Fatal(e)
	}
	m.Close()
	j, _ = m.Get(j.ID)
	if j.State != "cancelled" {
		t.Fatal(j.State)
	}
	if _, e = m.Start(fixture()); !errors.Is(e, ErrUnavailable) {
		t.Fatal(e)
	}
	dir := t.TempDir()
	m2 := manager(t, Options{DataDir: dir})
	if e = os.Remove(filepath.Join(dir, "state.json")); e != nil {
		t.Fatal(e)
	}
	if e = os.Mkdir(filepath.Join(dir, "state.json"), 0700); e != nil {
		t.Fatal(e)
	}
	if _, e = m2.Start(fixture()); !errors.Is(e, ErrUnavailable) {
		t.Fatalf("want persistence unavailable: %v", e)
	}
	if len(m2.Jobs()) != 0 {
		t.Fatal("failed start retained job")
	}
	if !errors.Is(m2.Clear(), ErrUnavailable) {
		t.Fatal("clear lost persistence error")
	}
}

func TestRestoreLowerLimitAndSource(t *testing.T) {
	dir := t.TempDir()
	m := manager(t, Options{DataDir: dir})
	c := fixture()
	c.DurationSeconds = 2
	j, err := m.Start(c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = m.Stop(j.ID); err != nil {
		t.Fatal(err)
	}
	m.observe(Observation{Kind: "imsi", Identity: "999990000000001"})
	m.mu.Lock()
	m.jobs[0].Error = "sensitive dependency stderr"
	m.observations = append(m.observations, Observation{Kind: "sms", Source: "untrusted", Text: "private SMS"})
	err = m.persistLocked()
	m.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	m.Close()
	r := manager(t, Options{Mode: "shielded", DataDir: dir, MaxDurationSeconds: 1})
	restored, _ := r.Get(j.ID)
	if restored.Error != "previous runtime error (details redacted)" {
		t.Fatal("unknown saved error retained")
	}
	obs := r.Observations("imsi")
	if len(obs) != 1 || obs[0].Source != "demo" {
		t.Fatal("historical demo relabelled as hardware")
	}
	sms := r.Observations("sms")
	if len(sms) != 1 || sms[0].Source != "unknown" || sms[0].Text != "[redacted]" {
		t.Fatal("unknown source or raw SMS preserved")
	}
}
