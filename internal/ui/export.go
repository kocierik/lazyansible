package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kocierik/lazyansible/internal/core"
	"github.com/kocierik/lazyansible/internal/history"
)

// exportRunMarkdown writes the last run's metadata and log output as a Markdown
// file in the current working directory. Returns the path written on success.
func exportRunMarkdown(workDir string, rec *history.Record, lines []core.LogLine) (string, error) {
	now := time.Now()
	fname := fmt.Sprintf("lazyansible-run-%s.md", now.Format("20060102-150405"))
	dest := filepath.Join(workDir, fname)

	var sb strings.Builder

	sb.WriteString("# lazyansible run report\n\n")
	sb.WriteString(fmt.Sprintf("**Generated**: %s\n\n", now.Format("2006-01-02 15:04:05")))

	// Metadata section.
	if rec != nil {
		sb.WriteString("## Run metadata\n\n")
		sb.WriteString(fmt.Sprintf("| Key | Value |\n|---|---|\n"))
		if rec.Kind == "adhoc" {
			sb.WriteString(fmt.Sprintf("| Type | ad-hoc |\n"))
			sb.WriteString(fmt.Sprintf("| Module | `%s` |\n", rec.Module))
			if rec.Args != "" {
				sb.WriteString(fmt.Sprintf("| Args | `%s` |\n", rec.Args))
			}
		} else {
			sb.WriteString(fmt.Sprintf("| Type | playbook |\n"))
			sb.WriteString(fmt.Sprintf("| Playbook | `%s` |\n", rec.PlaybookName))
			sb.WriteString(fmt.Sprintf("| Path | `%s` |\n", rec.PlaybookPath))
		}
		sb.WriteString(fmt.Sprintf("| Inventory | `%s` |\n", rec.Inventory))
		if rec.Limit != "" {
			sb.WriteString(fmt.Sprintf("| Limit | `%s` |\n", rec.Limit))
		}
		if rec.Tags != "" {
			sb.WriteString(fmt.Sprintf("| Tags | `%s` |\n", rec.Tags))
		}
		if rec.ExtraVars != "" {
			sb.WriteString(fmt.Sprintf("| Extra vars | `%s` |\n", rec.ExtraVars))
		}
		if rec.CheckMode {
			sb.WriteString("| Mode | check |\n")
		}
		if rec.DiffMode {
			sb.WriteString("| Diff | yes |\n")
		}
		sb.WriteString(fmt.Sprintf("| Start | %s |\n", rec.StartTime.Format("2006-01-02 15:04:05")))
		if !rec.EndTime.IsZero() {
			sb.WriteString(fmt.Sprintf("| Duration | %s |\n", rec.Duration()))
			sb.WriteString(fmt.Sprintf("| Exit code | %d |\n", rec.ExitCode))
		}
		sb.WriteString("\n")

		// Host status table.
		if len(rec.HostStats) > 0 {
			sb.WriteString("## Host status\n\n")
			sb.WriteString("| Host | Status |\n|---|---|\n")
			for host, status := range rec.HostStats {
				sb.WriteString(fmt.Sprintf("| %s | %s |\n", host, status))
			}
			sb.WriteString("\n")
		}
	}

	// Log output section.
	if len(lines) > 0 {
		sb.WriteString("## Log output\n\n```\n")
		for _, l := range lines {
			// Write raw text (no ANSI codes — these are the pre-style strings).
			sb.WriteString(l.Text)
			sb.WriteByte('\n')
		}
		sb.WriteString("```\n")
	}

	return dest, os.WriteFile(dest, []byte(sb.String()), 0o644)
}

// stripAnsi removes ANSI escape sequences from a string.
// Used when the caller only has already-styled text.
func stripAnsi(s string) string {
	var out strings.Builder
	inSeq := false
	for _, r := range s {
		if r == '\x1b' {
			inSeq = true
			continue
		}
		if inSeq {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inSeq = false
			}
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}
