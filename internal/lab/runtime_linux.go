//go:build linux

package lab

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

func shieldedAvailable(c Config) error {
	names := []string{"grgsm_scanner"}
	if c.Kind == "capture" {
		names = []string{"grgsm_livemon_headless", "tshark"}
	}
	for _, name := range names {
		if _, err := exec.LookPath(name); err != nil {
			return fmt.Errorf("%w: %s is not installed", ErrUnavailable, name)
		}
	}
	return nil
}

// Each direct child has its own process group. Only these owned groups are signalled.
// Context cancellation sends TERM; WaitDelay and final group KILL bound teardown.
func command(ctx context.Context, name string, args []string, out io.Writer) *exec.Cmd {
	c := exec.CommandContext(ctx, name, args...)
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Stdout = out
	c.Stderr = io.Discard
	c.Env = append(os.Environ(), "LC_ALL=C", "PYTHONUNBUFFERED=1")
	c.Cancel = func() error {
		err := syscall.Kill(-c.Process.Pid, syscall.SIGTERM)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	c.WaitDelay = 2 * time.Second
	return c
}

func runShielded(ctx context.Context, c Config, emit func(Observation)) error {
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var commands []*exec.Cmd
	if c.Kind == "scan" {
		commands = []*exec.Cmd{command(childCtx, "grgsm_scanner", []string{"-b", c.Band}, newLines(func(s string) {
			if o, ok := parseScan(s); ok {
				emit(o)
			}
		}))}
	} else {
		filter := "e212.imsi"
		if c.Mode == "sms" {
			filter = "gsm_sms"
		}
		// The SMS text field is intentionally NOT requested. No raw pcap or stderr is retained.
		args := []string{"-n", "-p", "-l", "-i", "lo", "-f", "udp port 4729", "-Y", filter, "-T", "fields", "-E", "occurrence=f", "-e", "frame.time_epoch"}
		if c.Mode == "imsi" {
			args = append(args, "-e", "e212.imsi")
		}
		commands = []*exec.Cmd{
			command(childCtx, "tshark", args, newLines(func(s string) {
				if o, ok := parseCapture(s, c.Mode, c.FrequencyMHz); ok {
					emit(o)
				}
			})),
			command(childCtx, "grgsm_livemon_headless", []string{"-f", strconv.FormatFloat(c.FrequencyMHz, 'f', 1, 64) + "M"}, io.Discard),
		}
	}
	type result struct{ err error }
	results := make(chan result, len(commands))
	started := 0
	for _, cmd := range commands {
		if err := cmd.Start(); err != nil {
			cancel()
			for i := 0; i < started; i++ {
				<-results
			}
			return err
		}
		started++
		go func(cmd *exec.Cmd) {
			err := cmd.Wait()
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			results <- result{err}
		}(cmd)
	}
	var first error
	select {
	case <-ctx.Done():
		first = ctx.Err()
	case r := <-results:
		started--
		first = r.err
		if ctx.Err() != nil {
			// A signalled process may report its exit concurrently with ctx.Done.
			first = ctx.Err()
		} else if c.Kind == "capture" && first == nil {
			first = errors.New("capture process exited before deadline")
		}
	}
	cancel()
	for i := 0; i < started; i++ {
		<-results
	}
	return first
}
