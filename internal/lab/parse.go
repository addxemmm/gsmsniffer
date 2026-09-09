package lab

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Verified upstream contracts:
// https://github.com/ptrkrysik/gr-gsm/blob/master/apps/grgsm_scanner (channel_info.__str__)
// https://github.com/ptrkrysik/gr-gsm/blob/master/apps/grgsm_livemon_headless.grc (-f; loopback UDP 4729)
// https://www.wireshark.org/docs/man-pages/tshark.html (-T fields, -e, -E occurrence=f)
// https://www.wireshark.org/docs/dfref/e/e212.html (e212.imsi)
// https://github.com/ptrkrysik/gr-gsm/blob/master/python/misc_utils/arfcn.py (GSM900 includes E-GSM)
// These contracts still require validation against the actual packaged hardware runtime.
var scanLine = regexp.MustCompile(`^ARFCN:\s*(\d{1,4}),\s*Freq:\s*(\d{3,4}\.\d)M,\s*CID:\s*(\d{1,5}),\s*LAC:\s*(\d{1,5}),\s*MCC:\s*(\d{1,3}),\s*MNC:\s*(\d{1,3}),\s*Pwr:\s*(-?\d{1,3})\s*$`)
var identityLine = regexp.MustCompile(`^\d{5,16}$`)

func parseScan(s string) (Observation, bool) {
	p := scanLine.FindStringSubmatch(s)
	if p == nil {
		return Observation{}, false
	}
	a, _ := strconv.Atoi(p[1])
	f, _ := strconv.ParseFloat(p[2], 64)
	power, _ := strconv.ParseFloat(p[7], 64)
	if a > 1023 || !((f >= 925.2 && f <= 959.8) || (f >= 1805.2 && f <= 1879.8)) {
		return Observation{}, false
	}
	return Observation{Kind: "frequencies", Timestamp: time.Now().UTC().Format(time.RFC3339Nano), ARFCN: a, FrequencyMHz: f, CellID: p[3], LAC: p[4], MCC: p[5], MNC: p[6], PowerDBM: power}, true
}
func parseCapture(s, mode string, frequency float64) (Observation, bool) {
	p := strings.Split(s, "\t")
	expected := 1
	if mode == "imsi" {
		expected = 2
	}
	if len(p) != expected {
		return Observation{}, false
	}
	epoch, err := strconv.ParseFloat(p[0], 64)
	if err != nil || math.IsNaN(epoch) || math.IsInf(epoch, 0) || epoch < 0 || epoch > 253402300799 {
		return Observation{}, false
	}
	seconds, frac := math.Modf(epoch)
	o := Observation{Kind: mode, Timestamp: time.Unix(int64(seconds), int64(frac*1e9)).UTC().Format(time.RFC3339Nano), FrequencyMHz: frequency}
	if mode == "imsi" {
		if !identityLine.MatchString(p[1]) {
			return Observation{}, false
		}
		o.Identity = p[1]
	} else if mode == "sms" {
		o.Text = "[redacted]"
	} else {
		return Observation{}, false
	}
	return sanitize(o), true
}

// lines bounds memory even if a dependency emits an unbounded line. Writes are
// handled by os/exec's single stdout copier. No raw output reaches disk or logs.
type lines struct {
	buffer  []byte
	discard bool
	emit    func(string)
}

func newLines(emit func(string)) *lines { return &lines{emit: emit} }
func (l *lines) Write(p []byte) (int, error) {
	for _, b := range p {
		if b == '\n' {
			if !l.discard {
				l.emit(strings.TrimSuffix(string(l.buffer), "\r"))
			}
			l.buffer = l.buffer[:0]
			l.discard = false
			continue
		}
		if l.discard {
			continue
		}
		if len(l.buffer) >= 16384 {
			l.buffer = l.buffer[:0]
			l.discard = true
			continue
		}
		l.buffer = append(l.buffer, b)
	}
	return len(p), nil
}
