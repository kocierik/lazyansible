// Package config loads lazyansible settings from ~/.lazyansible/config.yml
// (or a path given by the LAZYANSIBLE_CONFIG env var).
// Missing file → silent no-op; all fields are optional.
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds user-defined defaults.
type Config struct {
	// Inventory is the default inventory file path (overridden by -i flag).
	Inventory string `yaml:"inventory"`
	// PlaybookDir is the default playbook search directory (overridden by -d flag).
	PlaybookDir string `yaml:"playbook_dir"`
	// NoMouse disables mouse capture on startup.
	NoMouse bool `yaml:"no_mouse"`
	// NotifyOnFinish sends a desktop notification when a run completes.
	NotifyOnFinish bool `yaml:"notify_on_finish"`
	// DefaultCheckMode starts the tool with --check pre-enabled.
	DefaultCheckMode bool `yaml:"default_check_mode"`
	// DefaultDiffMode starts the tool with --diff pre-enabled.
	DefaultDiffMode bool `yaml:"default_diff_mode"`
}

// DefaultPath returns the default config file location.
func DefaultPath() string {
	if p := os.Getenv("LAZYANSIBLE_CONFIG"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".lazyansible", "config.yml")
}

// Load reads and parses the config file at path.
// Returns an empty Config (all zero values) if the file does not exist.
func Load(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// WriteExample writes an annotated example config to path (creates parent dirs).
func WriteExample(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	example := `# lazyansible configuration file
# All fields are optional. CLI flags always take precedence.

# Default inventory file (same as -i flag).
# inventory: ./inventories/hosts.yml

# Default playbook search directory (same as -d flag).
# playbook_dir: ./playbooks

# Disable mouse capture so you can select text normally in the terminal.
# Use Shift+click as an alternative without this setting.
# no_mouse: false

# Send a desktop notification (notify-send / osascript) when a run finishes.
notify_on_finish: true

# Start with --check mode pre-enabled.
# default_check_mode: false

# Start with --diff mode pre-enabled.
# default_diff_mode: false
`
	return os.WriteFile(path, []byte(example), 0o600)
}
