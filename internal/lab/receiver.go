package lab

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type receiverConfig struct {
	deviceArgs string
	gain       float64
	ppm        int
}

func newReceiverConfig(opts Options) (receiverConfig, error) {
	if len(opts.DeviceArgs) > 512 || !utf8.ValidString(opts.DeviceArgs) || strings.ContainsFunc(opts.DeviceArgs, func(r rune) bool {
		return unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || r == '\u2028' || r == '\u2029'
	}) {
		return receiverConfig{}, fmt.Errorf("%w: GSMSNIFFER_DEVICE_ARGS must be at most 512 bytes of UTF-8 without control characters", ErrInvalid)
	}
	gain := 24.0
	if opts.RXGain != nil {
		gain = *opts.RXGain
	}
	if math.IsNaN(gain) || math.IsInf(gain, 0) || gain < 0 || gain > 76 {
		return receiverConfig{}, fmt.Errorf("%w: GSMSNIFFER_RX_GAIN must be finite and 0..76", ErrInvalid)
	}
	if opts.PPM < -200 || opts.PPM > 200 {
		return receiverConfig{}, fmt.Errorf("%w: GSMSNIFFER_PPM must be an integer in -200..200", ErrInvalid)
	}
	return receiverConfig{deviceArgs: opts.DeviceArgs, gain: gain, ppm: opts.PPM}, nil
}

// One shared argv builder keeps scanning and capture on the same receiver.
// Device args stay a single argv value; no shell, tokenization or job overrides.
func (r receiverConfig) args() []string {
	args := []string{"-g", strconv.FormatFloat(r.gain, 'f', -1, 64), "-p", strconv.Itoa(r.ppm)}
	if r.deviceArgs != "" {
		args = append(args, "--args="+r.deviceArgs)
	}
	return args
}
