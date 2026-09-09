//go:build linux

package lab

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Executable test fixtures stand in for dependencies; no SDR commands run in tests.
func TestMain(m *testing.M) {
	if os.Getenv("GSMLAB_TEST_HELPER") == "1" {
		name := filepath.Base(os.Args[0])
		behavior := os.Getenv("GSMLAB_TEST_BEHAVIOR")
		if behavior == "fail" || (behavior == "fail-one" && name == "tshark") {
			os.Exit(7)
		}
		if name == "grgsm_scanner" && behavior != "hang" {
			fmt.Println("ARFCN:    1, Freq:  935.2M, CID:     1, LAC:     1, MCC: 999, MNC:  99, Pwr: -42")
			os.Exit(0)
		}
		if name == "tshark" {
			args := strings.Join(os.Args[1:], " ")
			if !strings.Contains(args, " -p ") {
				os.Exit(98)
			}
			if strings.Contains(args, "gsm_sms.sms_text") || strings.Contains(args, " -w ") {
				os.Exit(99)
			}
			if strings.Contains(args, "e212.imsi") {
				fmt.Println("1750000000.125\t999990000000001")
			} else {
				fmt.Println("1750000000.125")
			}
		}
		signal.Ignore(syscall.SIGTERM)
		time.Sleep(60 * time.Second)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func helperRuntime(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"grgsm_scanner", "grgsm_livemon_headless", "tshark"} {
		if err := os.Symlink(exe, filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
	t.Setenv("GSMLAB_TEST_HELPER", "1")
}

func TestShieldedFixtureScan(t *testing.T) {
	helperRuntime(t)
	m := manager(t, Options{Mode: "shielded"})
	j, err := m.Start(fixture())
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for m.Active() != nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	j, _ = m.Get(j.ID)
	if j.State != "finished" {
		t.Fatalf("%+v", j)
	}
	if len(m.Observations("frequencies")) != 1 {
		t.Fatal("scanner fixture not parsed")
	}
}
func TestShieldedFixtureFailure(t *testing.T) {
	helperRuntime(t)
	t.Setenv("GSMLAB_TEST_BEHAVIOR", "fail")
	m := manager(t, Options{Mode: "shielded"})
	j, err := m.Start(fixture())
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for m.Active() != nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	j, _ = m.Get(j.ID)
	if j.State != "failed" || j.Error == "" {
		t.Fatalf("%+v", j)
	}
}
func TestShieldedOwnedProcessStopBounded(t *testing.T) {
	helperRuntime(t)
	m := manager(t, Options{Mode: "shielded"})
	c := fixture()
	c.ScanJobID = scanForCapture(t, m).ScanJobID
	c.Kind = "capture"
	c.Mode = "imsi"
	c.FrequencyMHz = 935.2
	c.DurationSeconds = 30
	j, err := m.Start(c)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for len(m.Observations("imsi")) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	start := time.Now()
	j, err = m.Stop(j.ID)
	if err != nil || j.State != "cancelled" {
		t.Fatalf("%+v %v", j, err)
	}
	if time.Since(start) > 4*time.Second {
		t.Fatal("process teardown exceeded limit")
	}
	obs := m.Observations("imsi")
	if len(obs) != 1 || obs[0].Identity != "999**********01" {
		t.Fatal("missing masked fixture")
	}
}
func TestShieldedUnavailable(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if err := shieldedAvailable(fixture()); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := runShielded(ctx, fixture(), func(Observation) {}); err == nil {
		t.Fatal("missing executable accepted")
	}
}

func TestShieldedFirstFailureNotHiddenByTeardownDeadline(t *testing.T) {
	helperRuntime(t)
	t.Setenv("GSMLAB_TEST_BEHAVIOR", "fail-one")
	m := manager(t, Options{Mode: "shielded"})
	c := fixture()
	c.ScanJobID = scanForCapture(t, m).ScanJobID
	c.Kind = "capture"
	c.Mode = "imsi"
	c.FrequencyMHz = 935.2
	j, err := m.Start(c)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for m.Active() != nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	j, _ = m.Get(j.ID)
	if j.State != "failed" {
		t.Fatalf("failure hidden by cleanup deadline: %+v", j)
	}
}
