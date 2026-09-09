package lab

import "math"

// Frequency is the latest retained observation of a channel in one scan job.
// Provenance is required: legacy unlinked observations remain in history only.
type Frequency struct {
	ScanJobID    string  `json:"scan_job_id"`
	Band         string  `json:"band"`
	FrequencyMHz float64 `json:"frequency_mhz"`
	ARFCN        int     `json:"arfcn"`
	CellID       string  `json:"cell_id"`
	LAC          string  `json:"lac"`
	MCC          string  `json:"mcc"`
	MNC          string  `json:"mnc"`
	PowerDBM     float64 `json:"power_dbm"`
	Source       string  `json:"source"`
	Timestamp    string  `json:"timestamp"`
	Selectable   bool    `json:"selectable"`
}

func (m *Manager) Frequencies() []Frequency {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.frequenciesLocked()
}

func (m *Manager) frequenciesLocked() []Frequency {
	jobs := make(map[string]Job, len(m.jobs))
	for _, job := range m.jobs {
		if job.Kind == "scan" {
			jobs[job.ID] = job
		}
	}
	type key struct {
		jobID   string
		channel int64
	}
	seen := make(map[key]bool)
	items := []Frequency{}
	// The bounded append-only observation order gives deterministic newest-first
	// results, preserving the latest metadata without sorting untrusted timestamps.
	for i := len(m.observations) - 1; i >= 0; i-- {
		o := m.observations[i]
		job, ok := jobs[o.JobID]
		if !ok || o.Kind != "frequencies" || o.Source != m.opts.Mode || o.Band != job.Config.Band {
			continue
		}
		validation := Config{Kind: "capture", Band: o.Band, FrequencyMHz: o.FrequencyMHz, Mode: "imsi", DurationSeconds: 1, ShieldedAck: true}
		if m.validate(validation) != nil {
			continue
		}
		k := key{o.JobID, int64(math.Round(o.FrequencyMHz * 5))}
		if seen[k] {
			continue
		}
		seen[k] = true
		items = append(items, Frequency{
			ScanJobID: o.JobID, Band: o.Band, FrequencyMHz: o.FrequencyMHz,
			ARFCN: o.ARFCN, CellID: o.CellID, LAC: o.LAC, MCC: o.MCC,
			MNC: o.MNC, PowerDBM: o.PowerDBM, Source: o.Source, Timestamp: o.Timestamp,
			Selectable: job.State == "finished" || job.State == "cancelled",
		})
	}
	return items
}
