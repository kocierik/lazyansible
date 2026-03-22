// Package history persists playbook run records to ~/.lazyansible/history/.
package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Record holds metadata about a single playbook or ad-hoc run.
type Record struct {
	ID           string            `json:"id"`
	Kind         string            `json:"kind"` // "playbook" | "adhoc"
	PlaybookName string            `json:"playbook_name"`
	PlaybookPath string            `json:"playbook_path"`
	Inventory    string            `json:"inventory"`
	Limit        string            `json:"limit,omitempty"`
	Tags         string            `json:"tags,omitempty"`
	ExtraVars    string            `json:"extra_vars,omitempty"`
	CheckMode    bool              `json:"check_mode,omitempty"`
	DiffMode     bool              `json:"diff_mode,omitempty"`
	Module       string            `json:"module,omitempty"`   // ad-hoc
	Args         string            `json:"args,omitempty"`      // ad-hoc
	StartTime    time.Time         `json:"start_time"`
	EndTime      time.Time         `json:"end_time"`
	ExitCode     int               `json:"exit_code"`
	HostStats    map[string]string `json:"host_stats,omitempty"` // host → status
}

// Duration returns the run duration as a human-readable string.
func (r *Record) Duration() string {
	d := r.EndTime.Sub(r.StartTime).Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}

// Result returns a short result string for display.
func (r *Record) Result() string {
	if r.ExitCode == 0 {
		return "ok"
	}
	return fmt.Sprintf("exit %d", r.ExitCode)
}

// ─── Storage ──────────────────────────────────────────────────────────────────

// dir returns (and creates if needed) the history directory.
func dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(home, ".lazyansible", "history")
	if err := os.MkdirAll(d, 0o755); err != nil {
		return "", err
	}
	return d, nil
}

// Save writes a record to disk.
func Save(r *Record) error {
	d, err := dir()
	if err != nil {
		return err
	}
	filename := fmt.Sprintf("%s-%s.json",
		r.StartTime.Format("20060102-150405"),
		sanitize(r.PlaybookName),
	)
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(d, filename), data, 0o644)
}

// Load returns all history records sorted by StartTime descending (newest first).
func Load() ([]*Record, error) {
	d, err := dir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(d)
	if err != nil {
		return nil, err
	}

	var records []*Record
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(d, e.Name()))
		if err != nil {
			continue
		}
		var r Record
		if err := json.Unmarshal(data, &r); err != nil {
			continue
		}
		records = append(records, &r)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].StartTime.After(records[j].StartTime)
	})
	return records, nil
}

// Limit returns the last n records.
func Limit(records []*Record, n int) []*Record {
	if len(records) <= n {
		return records
	}
	return records[:n]
}

func sanitize(s string) string {
	out := make([]byte, 0, len(s))
	for _, b := range []byte(s) {
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '-' || b == '_' {
			out = append(out, b)
		} else {
			out = append(out, '-')
		}
	}
	if len(out) > 32 {
		out = out[:32]
	}
	return string(out)
}
