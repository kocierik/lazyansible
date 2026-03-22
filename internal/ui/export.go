package ui

import (
	"fmt"
	"os"
	"path/filepath"
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

	var sb []byte
	app := func(s string) { sb = append(sb, s...) }

	app("# lazyansible run report\n\n")
	app(fmt.Sprintf("**Generated**: %s\n\n", now.Format("2006-01-02 15:04:05")))

	// Metadata section.
	if rec != nil {
		app("## Run metadata\n\n")
		app("| Key | Value |\n|---|---|\n")
		if rec.Kind == "adhoc" {
			app("| Type | ad-hoc |\n")
			app(fmt.Sprintf("| Module | `%s` |\n", rec.Module))
			if rec.Args != "" {
				app(fmt.Sprintf("| Args | `%s` |\n", rec.Args))
			}
		} else {
			app("| Type | playbook |\n")
			app(fmt.Sprintf("| Playbook | `%s` |\n", rec.PlaybookName))
			app(fmt.Sprintf("| Path | `%s` |\n", rec.PlaybookPath))
		}
		app(fmt.Sprintf("| Inventory | `%s` |\n", rec.Inventory))
		if rec.Limit != "" {
			app(fmt.Sprintf("| Limit | `%s` |\n", rec.Limit))
		}
		if rec.Tags != "" {
			app(fmt.Sprintf("| Tags | `%s` |\n", rec.Tags))
		}
		if rec.ExtraVars != "" {
			app(fmt.Sprintf("| Extra vars | `%s` |\n", rec.ExtraVars))
		}
		if rec.CheckMode {
			app("| Mode | check |\n")
		}
		if rec.DiffMode {
			app("| Diff | yes |\n")
		}
		app(fmt.Sprintf("| Start | %s |\n", rec.StartTime.Format("2006-01-02 15:04:05")))
		if !rec.EndTime.IsZero() {
			app(fmt.Sprintf("| Duration | %s |\n", rec.Duration()))
			app(fmt.Sprintf("| Exit code | %d |\n", rec.ExitCode))
		}
		app("\n")

		// Host status table.
		if len(rec.HostStats) > 0 {
			app("## Host status\n\n")
			app("| Host | Status |\n|---|---|\n")
			for host, status := range rec.HostStats {
				app(fmt.Sprintf("| %s | %s |\n", host, status))
			}
			app("\n")
		}
	}

	// Log output section.
	if len(lines) > 0 {
		app("## Log output\n\n```\n")
		for _, l := range lines {
			app(l.Text)
			app("\n")
		}
		app("```\n")
	}

	return dest, os.WriteFile(dest, sb, 0o644)
}
