package main

import (
	"strings"
	"testing"

	"gsmsniffer/internal/lab"
)

func TestReceiverEnvironmentDefaultsAndOverrides(t *testing.T) {
	for _, tt := range []struct {
		name, args, gain, ppm string
		wantGain              float64
		wantPPM               int
	}{
		{"defaults", "", "", "", 24, 0},
		{"usrp", "uhd,type=b200,serial=SERIAL", "31.5", "-2", 31.5, -2},
		{"zero", "", "0", "0", 0, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GSMSNIFFER_DEVICE_ARGS", tt.args)
			t.Setenv("GSMSNIFFER_RX_GAIN", tt.gain)
			t.Setenv("GSMSNIFFER_PPM", tt.ppm)
			opts, err := loadReceiverOptions()
			if err != nil || opts.RXGain == nil || *opts.RXGain != tt.wantGain || opts.PPM != tt.wantPPM || opts.DeviceArgs != tt.args {
				t.Fatalf("receiver options mismatch: %v", err)
			}
			m, err := lab.New(opts)
			if err != nil {
				t.Fatal(err)
			}
			m.Close()
		})
	}
}

func TestReceiverEnvironmentRejectsInvalidNumbers(t *testing.T) {
	for _, key := range []string{"GSMSNIFFER_RX_GAIN", "GSMSNIFFER_PPM"} {
		for _, value := range []string{"NaN", "+Inf", "-Inf", "2000", "-2000", "24\n", "abc", strings.Repeat("0", 33)} {
			t.Run(key+"/"+value, func(t *testing.T) {
				t.Setenv("GSMSNIFFER_RX_GAIN", "24")
				t.Setenv("GSMSNIFFER_PPM", "0")
				t.Setenv(key, value)
				if _, err := loadReceiverOptions(); err == nil || !strings.Contains(err.Error(), key) {
					t.Fatalf("invalid numeric configuration accepted: %v", err)
				}
			})
		}
	}
}

func TestReceiverStartupRejectsDeviceConfiguration(t *testing.T) {
	t.Setenv("GSMSNIFFER_TOKEN", "")
	t.Setenv("GSMSNIFFER_TOKEN_FILE", "")
	t.Setenv("GSMSNIFFER_DEVICE_ARGS", "uhd\nserial=SERIAL")
	t.Setenv("GSMSNIFFER_RX_GAIN", "24")
	t.Setenv("GSMSNIFFER_PPM", "0")
	t.Setenv("GSMSNIFFER_MAX_DURATION_SECONDS", "300")
	if err := run(); err == nil || !strings.Contains(err.Error(), "GSMSNIFFER_DEVICE_ARGS") {
		t.Fatalf("invalid receiver configuration accepted at startup: %v", err)
	}
}

func TestReceiverEnvironmentRejectsFractionalPPM(t *testing.T) {
	t.Setenv("GSMSNIFFER_RX_GAIN", "24")
	for _, value := range []string{"1.5", "1.0", "1e1", "-1.5"} {
		t.Setenv("GSMSNIFFER_PPM", value)
		if _, err := loadReceiverOptions(); err == nil {
			t.Fatal("non-integer PPM accepted")
		}
	}
}
