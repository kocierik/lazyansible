package core

import "time"

// Panel identifies which TUI panel is focused.
type Panel int

const (
	PanelInventory Panel = iota
	PanelPlaybooks
	PanelStatus
	PanelLogs
)

// Host represents a single Ansible inventory host.
type Host struct {
	Name   string
	Groups []string
	Vars   map[string]string
}

// Group represents an Ansible inventory group.
type Group struct {
	Name     string
	Hosts    []string
	Children []string
	Vars     map[string]string
}

// Inventory holds the parsed Ansible inventory.
type Inventory struct {
	Hosts  map[string]*Host
	Groups map[string]*Group
	// OrderedGroups preserves display order.
	OrderedGroups []string
}

// Playbook represents a discovered Ansible playbook file.
type Playbook struct {
	Name string
	Path string
	// Hosts extracted from the playbook header, if parseable.
	Hosts []string
	// Tags collected from all plays and tasks.
	Tags []string
}

// RunOptions holds the parameters for a playbook run.
type RunOptions struct {
	Playbook     string
	Inventory    string
	Limit        string
	Tags         string
	CheckMode    bool
	DiffMode     bool
	ExtraVars    map[string]string
	// ExtraVarsRaw is passed verbatim as -e "..." (space-separated key=val pairs).
	ExtraVarsRaw string
	// VaultPasswordFile is passed as --vault-password-file if non-empty.
	VaultPasswordFile string
	// Env holds extra KEY=VALUE pairs appended to the process environment.
	Env []string
}

// AdHocOptions holds the parameters for an ansible ad-hoc command.
type AdHocOptions struct {
	Hosts     string
	Inventory string
	Module    string
	Args      string
	ExtraVars map[string]string
	Become    bool
}

// TaskStatus is the result state of an Ansible task on a host.
type TaskStatus int

const (
	TaskStatusUnknown TaskStatus = iota
	TaskStatusOK
	TaskStatusChanged
	TaskStatusFailed
	TaskStatusSkipped
	TaskStatusUnreachable
)

func (s TaskStatus) String() string {
	switch s {
	case TaskStatusOK:
		return "ok"
	case TaskStatusChanged:
		return "changed"
	case TaskStatusFailed:
		return "failed"
	case TaskStatusSkipped:
		return "skipped"
	case TaskStatusUnreachable:
		return "unreachable"
	default:
		return "unknown"
	}
}

// HostResult tracks per-host execution status.
type HostResult struct {
	Host       string
	Status     TaskStatus
	TaskName   string
	ChangedAt  time.Time
}

// RunResult summarises a completed playbook run.
type RunResult struct {
	Playbook  string
	StartTime time.Time
	EndTime   time.Time
	Hosts     map[string]*HostResult
	ExitCode  int
}

// LogLevel categorises a log line for colouring.
type LogLevel int

const (
	LogLevelInfo LogLevel = iota
	LogLevelOK
	LogLevelChanged
	LogLevelFailed
	LogLevelWarning
	LogLevelDebug
	// Diff visualisation levels (--diff output).
	LogLevelDiffAdd    // lines starting with +
	LogLevelDiffRemove // lines starting with -
	LogLevelDiffHunk   // lines starting with @@
	LogLevelDiffHeader // --- / +++ file headers
)

// LogLine is a single line of streamed ansible output.
type LogLine struct {
	Text      string
	Level     LogLevel
	Timestamp time.Time
}
