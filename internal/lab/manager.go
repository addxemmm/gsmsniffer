// Package lab manages bounded experiments on owned equipment in RF-shielded labs.
// Demo observations are synthetic. Identifiers and message text are never retained raw.
package lab

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ErrBusy        = errors.New("another experiment is running")
	ErrNotFound    = errors.New("job not found")
	ErrInvalid     = errors.New("invalid experiment configuration")
	ErrUnavailable = errors.New("experiment runtime unavailable")
)

const (
	maxJobs         = 100
	maxObservations = 1000
	maxStateBytes   = 2 << 20
)

type Config struct {
	Kind            string  `json:"kind"`
	ScanJobID       string  `json:"scan_job_id,omitempty"`
	Band            string  `json:"band"`
	FrequencyMHz    float64 `json:"frequency_mhz"`
	Mode            string  `json:"mode"`
	DurationSeconds int     `json:"duration_seconds"`
	ShieldedAck     bool    `json:"shielded_ack"`
}
type Job struct {
	ID        string     `json:"id"`
	Kind      string     `json:"kind"`
	State     string     `json:"state"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	Error     string     `json:"error,omitempty"`
	Config    Config     `json:"config"`
}
type Observation struct {
	Kind         string  `json:"kind"`
	JobID        string  `json:"job_id,omitempty"`
	Band         string  `json:"band,omitempty"`
	Source       string  `json:"source"`
	Timestamp    string  `json:"timestamp"`
	Identity     string  `json:"identity,omitempty"`
	Text         string  `json:"text,omitempty"`
	ARFCN        int     `json:"arfcn,omitempty"`
	FrequencyMHz float64 `json:"frequency_mhz,omitempty"`
	CellID       string  `json:"cell_id,omitempty"`
	LAC          string  `json:"lac,omitempty"`
	MCC          string  `json:"mcc,omitempty"`
	MNC          string  `json:"mnc,omitempty"`
	PowerDBM     float64 `json:"power_dbm,omitempty"`
}

// ARFCN zero is a valid GSM900 channel, not a missing scanner result. Capture
// events do not request an ARFCN field and should not imply that channel zero was observed.
func (o Observation) MarshalJSON() ([]byte, error) {
	type wire Observation
	var arfcn *int
	if o.Kind == "frequencies" || o.ARFCN != 0 {
		value := o.ARFCN
		arfcn = &value
	}
	return json.Marshal(struct {
		wire
		ARFCN *int `json:"arfcn,omitempty"`
	}{wire(o), arfcn})
}

type Options struct {
	Mode, DataDir      string
	MaxDurationSeconds int
	// Receiver settings are deployment-only, never accepted in HTTP job config.
	DeviceArgs string
	RXGain     *float64 // nil selects 24 dB; a pointer permits explicit zero gain.
	PPM        int
}
type persisted struct {
	Jobs         []Job         `json:"jobs"`
	Observations []Observation `json:"observations"`
}
type running struct {
	id     string
	cancel context.CancelFunc
	done   chan struct{}
}
type Manager struct {
	mu           sync.Mutex
	opts         Options
	receiver     receiverConfig
	jobs         []Job
	observations []Observation
	active       *running
	closed       bool
}

func New(opts Options) (*Manager, error) {
	if opts.Mode == "" {
		opts.Mode = "demo"
	}
	if opts.Mode != "demo" && opts.Mode != "shielded" {
		return nil, fmt.Errorf("%w: runtime mode must be demo or shielded", ErrInvalid)
	}
	if opts.MaxDurationSeconds == 0 {
		opts.MaxDurationSeconds = 300
	}
	if opts.MaxDurationSeconds < 1 || opts.MaxDurationSeconds > 3600 {
		return nil, fmt.Errorf("%w: maximum duration must be 1..3600", ErrInvalid)
	}
	receiver, err := newReceiverConfig(opts)
	if err != nil {
		return nil, err
	}
	// Keep only the immutable normalized receiver values, not a caller-owned pointer.
	opts.RXGain = nil
	m := &Manager{opts: opts, receiver: receiver, jobs: []Job{}, observations: []Observation{}}
	if opts.DataDir != "" {
		if err := os.MkdirAll(opts.DataDir, 0700); err != nil {
			return nil, fmt.Errorf("prepare data directory: %w", err)
		}
		if err := m.restore(); err != nil {
			return nil, err
		}
		if err := m.persistLocked(); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func (m *Manager) validate(c Config) error {
	bad := func(s string) error { return fmt.Errorf("%w: %s", ErrInvalid, s) }
	if c.Kind != "scan" && c.Kind != "capture" {
		return bad("kind must be scan or capture")
	}
	if c.Band != "GSM900" && c.Band != "DCS1800" {
		return bad("band must be GSM900 or DCS1800")
	}
	if c.DurationSeconds < 1 || c.DurationSeconds > m.opts.MaxDurationSeconds {
		return bad("duration is outside configured limit")
	}
	if !c.ShieldedAck {
		return bad("acknowledge owned equipment and RF-shielded enclosure")
	}
	if math.IsNaN(c.FrequencyMHz) || math.IsInf(c.FrequencyMHz, 0) {
		return bad("frequency must be finite")
	}
	if c.Kind == "scan" {
		if c.Mode != "" || c.FrequencyMHz != 0 || c.ScanJobID != "" {
			return bad("scan accepts band only; omit mode, frequency and scan_job_id")
		}
		return nil
	}
	if c.ScanJobID != "" {
		decoded, err := hex.DecodeString(c.ScanJobID)
		if err != nil || len(decoded) != 16 || c.ScanJobID != strings.ToLower(c.ScanJobID) {
			return bad("scan_job_id must be a scan job ID")
		}
	}
	if c.Mode != "imsi" && c.Mode != "sms" {
		return bad("capture mode must be imsi or sms")
	}
	lo, hi := 925.2, 959.8
	if c.Band == "DCS1800" {
		lo, hi = 1805.2, 1879.8
	}
	if c.FrequencyMHz < lo || c.FrequencyMHz > hi {
		return bad("frequency outside selected GSM downlink band")
	}
	channel := (c.FrequencyMHz - lo) / 0.2
	if math.Abs(channel-math.Round(channel)) > 0.000001 {
		return bad("frequency must align to a 200 kHz channel")
	}
	return nil
}

func (m *Manager) Start(c Config) (Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return Job{}, ErrUnavailable
	}
	if err := m.validate(c); err != nil {
		return Job{}, err
	}
	if m.active != nil {
		return Job{}, ErrBusy
	}
	if c.Kind == "capture" {
		matched := false
		for _, frequency := range m.frequenciesLocked() {
			if frequency.Selectable && frequency.ScanJobID == c.ScanJobID && frequency.Band == c.Band && math.Abs(frequency.FrequencyMHz-c.FrequencyMHz) < 0.000001 {
				matched = true
				break
			}
		}
		if !matched {
			return Job{}, fmt.Errorf("%w: select a retained frequency from a completed scan in the current runtime mode", ErrInvalid)
		}
	}
	if m.opts.Mode == "shielded" {
		if err := shieldedAvailable(c); err != nil {
			return Job{}, err
		}
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return Job{}, fmt.Errorf("%w: ID generation", ErrUnavailable)
	}
	j := Job{ID: hex.EncodeToString(id[:]), Kind: c.Kind, State: "running", StartedAt: time.Now().UTC(), Config: c}
	oldJobs := m.jobs
	m.jobs = append(append([]Job(nil), m.jobs...), j)
	if len(m.jobs) > maxJobs {
		m.jobs = m.jobs[len(m.jobs)-maxJobs:]
	}
	if err := m.persistLocked(); err != nil {
		m.jobs = oldJobs
		return Job{}, fmt.Errorf("%w: state persistence failed", ErrUnavailable)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.DurationSeconds)*time.Second)
	r := &running{id: j.ID, cancel: cancel, done: make(chan struct{})}
	m.active = r
	go m.execute(ctx, r, c)
	return cloneJob(j), nil
}

func (m *Manager) execute(ctx context.Context, r *running, c Config) {
	var err error
	if m.opts.Mode == "demo" {
		err = m.demo(ctx, c)
	} else {
		err = runShielded(ctx, c, m.receiver, m.observe)
	}
	m.mu.Lock()
	state, message := "finished", ""
	if errors.Is(err, context.Canceled) {
		state = "cancelled"
	} else if !errors.Is(err, context.DeadlineExceeded) && err != nil {
		state, message = "failed", "runtime process failed; verify dependencies, capture permissions and attached SDR"
	}
	for i := range m.jobs {
		if m.jobs[i].ID == r.id {
			now := time.Now().UTC()
			m.jobs[i].State = state
			m.jobs[i].EndedAt = &now
			m.jobs[i].Error = message
		}
	}
	if e := m.persistLocked(); e != nil {
		for i := range m.jobs {
			if m.jobs[i].ID == r.id {
				m.jobs[i].State = "failed"
				m.jobs[i].Error = "state persistence failed"
			}
		}
	}
	m.active = nil
	close(r.done)
	m.mu.Unlock()
	r.cancel()
}

func (m *Manager) Stop(id string) (Job, error) {
	m.mu.Lock()
	if m.active != nil && (id == "" || m.active.id == id) {
		r := m.active
		r.cancel()
		m.mu.Unlock()
		<-r.done
		j, _ := m.Get(r.id)
		return j, nil
	}
	for _, j := range m.jobs {
		if j.ID == id {
			m.mu.Unlock()
			return cloneJob(j), nil
		}
	}
	m.mu.Unlock()
	return Job{}, ErrNotFound
}
func cloneJob(j Job) Job {
	if j.EndedAt != nil {
		t := *j.EndedAt
		j.EndedAt = &t
	}
	return j
}
func (m *Manager) Get(id string) (Job, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, j := range m.jobs {
		if j.ID == id {
			return cloneJob(j), true
		}
	}
	return Job{}, false
}
func (m *Manager) Jobs() []Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Job, 0, len(m.jobs))
	for i := len(m.jobs) - 1; i >= 0; i-- {
		out = append(out, cloneJob(m.jobs[i]))
	}
	return out
}
func (m *Manager) Active() *Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active == nil {
		return nil
	}
	for _, j := range m.jobs {
		if j.ID == m.active.id {
			c := cloneJob(j)
			return &c
		}
	}
	return nil
}
func (m *Manager) Observations(kind string) []Observation {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []Observation{}
	for _, o := range m.observations {
		if kind == "" || o.Kind == kind {
			out = append(out, o)
		}
	}
	return out
}
func (m *Manager) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.active != nil {
		return ErrBusy
	}
	old := m.observations
	m.observations = []Observation{}
	if err := m.persistLocked(); err != nil {
		m.observations = old
		return fmt.Errorf("%w: state persistence failed", ErrUnavailable)
	}
	return nil
}
func (m *Manager) Close() {
	m.mu.Lock()
	m.closed = true
	r := m.active
	if r != nil {
		r.cancel()
	}
	m.mu.Unlock()
	if r != nil {
		<-r.done
	}
}

func sanitize(o Observation) Observation {
	if o.Identity != "" {
		if o.Identity == "[masked]" {
			// Preserve the placeholder across repeated reads and restarts.
		} else if len(o.Identity) >= 10 && len(o.Identity) <= 16 {
			o.Identity = o.Identity[:3] + strings.Repeat("*", len(o.Identity)-5) + o.Identity[len(o.Identity)-2:]
		} else {
			o.Identity = "[masked]"
		}
	}
	if o.Text != "" || o.Kind == "sms" {
		o.Text = "[redacted]"
	}
	return o
}
func (m *Manager) observe(o Observation) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o = sanitize(o)
	o.Source = m.opts.Mode
	o.JobID, o.Band = "", ""
	if m.active != nil {
		for _, job := range m.jobs {
			if job.ID == m.active.id {
				o.JobID, o.Band = job.ID, job.Config.Band
				break
			}
		}
	}
	m.observations = append(m.observations, o)
	if len(m.observations) > maxObservations {
		copy(m.observations, m.observations[len(m.observations)-maxObservations:])
		m.observations = m.observations[:maxObservations]
	}
}
func (m *Manager) demo(ctx context.Context, c Config) error {
	emit := func() {
		o := Observation{Timestamp: time.Now().UTC().Format(time.RFC3339Nano), MCC: "999", MNC: "99"}
		if c.Kind == "scan" {
			o.Kind = "frequencies"
			o.FrequencyMHz = 935.2
			o.ARFCN = 1
			if c.Band == "DCS1800" {
				o.FrequencyMHz = 1805.2
				o.ARFCN = 512
			}
			o.CellID = "1"
			o.LAC = "1"
			o.PowerDBM = -42
		} else {
			o.Kind = c.Mode
			o.FrequencyMHz = c.FrequencyMHz
			if c.Mode == "imsi" {
				o.Identity = "999990000000001"
			} else {
				o.Text = "SYNTHETIC LAB DEMO"
			}
		}
		m.observe(o)
	}
	emit()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			emit()
		}
	}
}

func (m *Manager) persistLocked() error {
	if m.opts.DataDir == "" {
		return nil
	}
	b, err := json.Marshal(persisted{Jobs: m.jobs, Observations: m.observations})
	if err != nil {
		return err
	}
	if len(b) > maxStateBytes {
		return errors.New("state size limit exceeded")
	}
	f, err := os.CreateTemp(m.opts.DataDir, ".state-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(b)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(name, filepath.Join(m.opts.DataDir, "state.json"))
}
func (m *Manager) restore() error {
	p := filepath.Join(m.opts.DataDir, "state.json")
	info, err := os.Lstat(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > maxStateBytes {
		return errors.New("invalid or oversized state file")
	}
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()
	var state persisted
	d := json.NewDecoder(io.LimitReader(f, maxStateBytes+1))
	if err = d.Decode(&state); err != nil {
		return fmt.Errorf("invalid saved state: %w", err)
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return errors.New("trailing saved state data")
	}
	if len(state.Jobs) > maxJobs {
		state.Jobs = state.Jobs[len(state.Jobs)-maxJobs:]
	}
	if len(state.Observations) > maxObservations {
		state.Observations = state.Observations[len(state.Observations)-maxObservations:]
	}
	now := time.Now().UTC()
	// Historical experiments remain valid when a deployment lowers its new-job limit.
	historyValidator := &Manager{opts: Options{Mode: "demo", MaxDurationSeconds: 3600}}
	for i := range state.Jobs {
		j := &state.Jobs[i]
		_, idError := hex.DecodeString(j.ID)
		if len(j.ID) != 32 || idError != nil || j.Kind != j.Config.Kind || historyValidator.validate(j.Config) != nil {
			return errors.New("invalid saved job")
		}
		if j.Error != "" {
			switch j.Error {
			case "runtime process failed; verify dependencies, capture permissions and attached SDR", "state persistence failed", "service restarted while experiment was running", "previous runtime error (details redacted)":
			default:
				j.Error = "previous runtime error (details redacted)"
			}
		}
		if j.State == "running" {
			j.State = "failed"
			j.Error = "service restarted while experiment was running"
			j.EndedAt = &now
		}
		if j.State != "finished" && j.State != "cancelled" && j.State != "failed" {
			return errors.New("invalid saved job state")
		}
	}
	m.jobs = state.Jobs
	for _, o := range state.Observations {
		if o.Kind != "frequencies" && o.Kind != "imsi" && o.Kind != "sms" {
			continue
		}
		if len(o.JobID) > 32 || len(o.Band) > 16 || len(o.Timestamp) > 64 || len(o.CellID) > 16 || len(o.LAC) > 16 || len(o.MCC) > 3 || len(o.MNC) > 3 {
			continue
		}
		if o.Source != "demo" && o.Source != "shielded" {
			o.Source = "unknown"
		}
		m.observations = append(m.observations, sanitize(o))
	}
	return nil
}
