package lab

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
)

func gainPointer(v float64) *float64 { return &v }

func TestReceiverDefaultsAndExplicitZero(t *testing.T) {
	for _, tt := range []struct {
		name string
		opts Options
		want []string
	}{
		{"default", Options{}, []string{"-g", "24", "-p", "0"}},
		{"zero gain", Options{RXGain: gainPointer(0)}, []string{"-g", "0", "-p", "0"}},
		{"upper limits", Options{RXGain: gainPointer(76), PPM: 200}, []string{"-g", "76", "-p", "200"}},
		{"negative ppm", Options{PPM: -200}, []string{"-g", "24", "-p", "-200"}},
		{"one device argument", Options{DeviceArgs: "uhd,type=b200,serial=SERIAL,name=lab fixture; --other"}, []string{"-g", "24", "-p", "0", "--args=uhd,type=b200,serial=SERIAL,name=lab fixture; --other"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := manager(t, tt.opts)
			if got := m.receiver.args(); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("args=%q, want %q", got, tt.want)
			}
		})
	}
}

func TestReceiverRejectsInvalidDeploymentSettings(t *testing.T) {
	for _, args := range []string{strings.Repeat("x", 513), "uhd\nserial=SERIAL", "uhd\r", "uhd\t", "uhd\x00", "uhd\x7f", "uhd\u0085", "uhd\u2028", "uhd\u2029", "uhd\u200b", "uhd\xff"} {
		if _, err := New(Options{DeviceArgs: args}); !errors.Is(err, ErrInvalid) {
			t.Fatalf("invalid device args accepted: error=%v", err)
		}
	}
	for _, gain := range []float64{-0.1, 76.1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := New(Options{RXGain: gainPointer(gain)}); !errors.Is(err, ErrInvalid) {
			t.Fatalf("invalid gain accepted: error=%v", err)
		}
	}
	for _, ppm := range []int{-201, 201} {
		if _, err := New(Options{PPM: ppm}); !errors.Is(err, ErrInvalid) {
			t.Fatalf("invalid ppm accepted: error=%v", err)
		}
	}
	m := manager(t, Options{DeviceArgs: strings.Repeat("x", 512)})
	if len(m.receiver.deviceArgs) != 512 {
		t.Fatal("valid boundary configuration changed")
	}
}

func TestReceiverSnapshotAndHTTPBoundary(t *testing.T) {
	gain := 30.0
	m := manager(t, Options{RXGain: &gain})
	gain = 75
	if m.receiver.gain != 30 {
		t.Fatal("caller-owned gain pointer changed active manager")
	}
	for _, field := range []string{"device_args", "rx_gain", "ppm"} {
		decoder := json.NewDecoder(strings.NewReader(`{"` + field + `":0}`))
		decoder.DisallowUnknownFields()
		var config Config
		if err := decoder.Decode(&config); err == nil {
			t.Fatalf("deployment-only field %s accepted by job config", field)
		}
	}
}
