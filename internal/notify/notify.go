// Package notify sends desktop notifications at the end of Ansible runs.
// It tries notify-send (Linux/freedesktop), then osascript (macOS), then
// terminal-notifier (macOS alternative). Silently no-ops if none are available.
package notify

import (
	"fmt"
	"os/exec"
	"runtime"
)

// RunResult summarises the outcome of a completed run.
type RunResult struct {
	PlaybookName string
	ExitCode     int
	Duration     string // human-readable, e.g. "1m23s"
}

// Send fires a best-effort desktop notification. Errors are silently ignored
// so a missing notification daemon never breaks the TUI.
func Send(r RunResult) {
	title, body := buildMessage(r)
	switch runtime.GOOS {
	case "linux":
		sendLinux(title, body, r.ExitCode == 0)
	case "darwin":
		sendMacOS(title, body)
	}
}

func buildMessage(r RunResult) (title, body string) {
	if r.ExitCode == 0 {
		title = "✓ lazyansible — success"
	} else {
		title = fmt.Sprintf("✗ lazyansible — failed (exit %d)", r.ExitCode)
	}
	if r.Duration != "" {
		body = fmt.Sprintf("%s  ·  %s", r.PlaybookName, r.Duration)
	} else {
		body = r.PlaybookName
	}
	return title, body
}

func sendLinux(title, body string, ok bool) {
	if _, err := exec.LookPath("notify-send"); err != nil {
		return
	}
	urgency := "normal"
	if !ok {
		urgency = "critical"
	}
	_ = exec.Command("notify-send",
		"--urgency="+urgency,
		"--app-name=lazyansible",
		"--expire-time=6000",
		title,
		body,
	).Start()
}

func sendMacOS(title, body string) {
	// Try terminal-notifier first (richer), fall back to osascript.
	if _, err := exec.LookPath("terminal-notifier"); err == nil {
		_ = exec.Command("terminal-notifier",
			"-title", title,
			"-message", body,
			"-sender", "com.apple.Terminal",
		).Start()
		return
	}
	script := fmt.Sprintf(
		`display notification %q with title %q`,
		body, title,
	)
	_ = exec.Command("osascript", "-e", script).Start()
}
