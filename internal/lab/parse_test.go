package lab

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestARFCNZeroSerialization(t *testing.T) {
	o := Observation{Kind: "frequencies", ARFCN: 0, FrequencyMHz: 935.0}
	b, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"arfcn":0`) {
		t.Fatalf("valid channel zero omitted: %s", b)
	}
	var decoded Observation
	if err = json.Unmarshal(b, &decoded); err != nil || decoded.Kind != "frequencies" || decoded.ARFCN != 0 {
		t.Fatal("observation round trip failed")
	}
	b, err = json.Marshal(Observation{Kind: "imsi"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"arfcn"`) {
		t.Fatal("capture implied a channel it did not measure")
	}
}

func TestScanContract(t *testing.T) {
	o, ok := parseScan("ARFCN:    1, Freq:  935.2M, CID:     1, LAC:     1, MCC: 999, MNC:  99, Pwr: -42")
	if !ok || o.Kind != "frequencies" || o.MCC != "999" || o.FrequencyMHz != 935.2 || o.PowerDBM != -42 {
		t.Fatalf("parse %+v %v", o, ok)
	}
	for _, s := range []string{"ARFCN: 975, Freq: 925.2M, CID: 1, LAC: 1, MCC: 999, MNC: 99, Pwr: -42", "ARFCN: 0, Freq: 935.0M, CID: 1, LAC: 1, MCC: 999, MNC: 99, Pwr: -42"} {
		if _, ok := parseScan(s); !ok {
			t.Fatal("E-GSM / ARFCN 0 rejected")
		}
	}
	for _, s := range []string{"diagnostic", "ARFCN: 2048, Freq: 935.2M, CID: 1, LAC: 1, MCC: 999, MNC: 99, Pwr: -42", "ARFCN: 1, Freq: 100.2M, CID: 1, LAC: 1, MCC: 999, MNC: 99, Pwr: -42"} {
		if _, ok := parseScan(s); ok {
			t.Fatal("accepted invalid scanner output")
		}
	}
}
func TestCaptureContract(t *testing.T) {
	o, ok := parseCapture("1750000000.125\t999990000000001", "imsi", 935.2)
	if !ok || o.Identity != "999**********01" || o.Timestamp != "2025-06-15T15:06:40.125Z" {
		t.Fatalf("parse %+v %v", o, ok)
	}
	o, ok = parseCapture("1750000000.125", "sms", 935.2)
	if !ok || o.Text != "[redacted]" {
		t.Fatal("SMS redaction")
	}
	for _, s := range []string{"bad\t999990000000001", "NaN\t999990000000001", "+Inf\t999990000000001", "-1\t999990000000001", "1750000000\tnot-digits", "1750000000\t999990000000001\textra"} {
		if _, ok := parseCapture(s, "imsi", 935.2); ok {
			t.Fatal("accepted invalid capture output")
		}
	}
	if _, ok := parseCapture("1750000000", "other", 935.2); ok {
		t.Fatal("unknown mode")
	}
}
func TestBoundedLines(t *testing.T) {
	var out []string
	l := newLines(func(s string) { out = append(out, s) })
	_, _ = l.Write([]byte("a"))
	_, _ = l.Write([]byte("b\r\n"))
	_, _ = l.Write([]byte(strings.Repeat("x", 20000) + "\nnext\n"))
	if len(out) != 2 || out[0] != "ab" || out[1] != "next" {
		t.Fatalf("lines %+v", out)
	}
	if len(l.buffer) > 16384 {
		t.Fatal("line memory unbounded")
	}
}
func TestSanitization(t *testing.T) {
	for _, id := range []string{"1", "12345", "123456789", "[masked]"} {
		o := sanitize(Observation{Identity: id, Text: "private"})
		if o.Identity != "[masked]" || o.Text != "[redacted]" {
			t.Fatal(o)
		}
		if sanitize(o) != o {
			t.Fatal("sanitization is not idempotent")
		}
	}
}
