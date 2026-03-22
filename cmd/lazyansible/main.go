package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	appcfg "github.com/kocierik/lazyansible/internal/config"
	"github.com/kocierik/lazyansible/internal/ui"
)

func main() {
	var (
		inventoryFlag  = flag.String("i", "", "Path to inventory file (auto-detected if omitted)")
		playbookDir    = flag.String("d", "", "Directory to search for playbooks (defaults to current dir)")
		noMouseFlag    = flag.Bool("no-mouse", false, "Disable mouse capture (allows native text selection)")
		initConfigFlag = flag.Bool("init-config", false, "Write an example config to ~/.lazyansible/config.yml and exit")
		versionFlag    = flag.Bool("version", false, "Print version and exit")
	)
	flag.Parse()

	if *versionFlag {
		fmt.Println("lazyansible v0.7.0")
		os.Exit(0)
	}

	// Load user config (silent no-op if file missing).
	cfgPath := appcfg.DefaultPath()
	userCfg, err := appcfg.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not read config %s: %v\n", cfgPath, err)
	}

	if *initConfigFlag {
		if err := appcfg.WriteExample(cfgPath); err != nil {
			fmt.Fprintf(os.Stderr, "error writing config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Config written to %s\n", cfgPath)
		os.Exit(0)
	}

	workDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get working directory: %v\n", err)
		os.Exit(1)
	}

	// CLI flags take precedence over config file values.
	invPath := *inventoryFlag
	if invPath == "" {
		invPath = userCfg.Inventory
	}
	if invPath != "" && !filepath.IsAbs(invPath) {
		invPath = filepath.Join(workDir, invPath)
	}

	pbDir := *playbookDir
	if pbDir == "" {
		pbDir = userCfg.PlaybookDir
	}
	if pbDir != "" && !filepath.IsAbs(pbDir) {
		pbDir = filepath.Join(workDir, pbDir)
	}

	noMouse := *noMouseFlag || userCfg.NoMouse

	cfg := ui.Config{
		InventoryPath: invPath,
		PlaybookDir:   pbDir,
		WorkDir:       workDir,
	}

	app := ui.New(cfg)
	app.SetNotifyOnFinish(userCfg.NotifyOnFinish)

	opts := []tea.ProgramOption{tea.WithAltScreen()}
	if !noMouse {
		opts = append(opts, tea.WithMouseCellMotion())
	}

	p := tea.NewProgram(app, opts...)
	app.SetProgram(p)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "lazyansible error: %v\n", err)
		os.Exit(1)
	}
}
