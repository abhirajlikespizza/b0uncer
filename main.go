package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/b0uncer/b0uncer/internal/dashboard"
	"github.com/b0uncer/b0uncer/internal/engine"
	"github.com/b0uncer/b0uncer/internal/logger"
)

//go:embed web/index.html
var indexHTML []byte

type appConfig struct {
	RealBash string `json:"real_bash"`
}

func homeDir() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return os.Getenv("HOME")
}

func isDashboardRunning() bool {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:3456", 150*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func startDashboardDaemon() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	devNull, _ := os.Open(os.DevNull)
	cmd := exec.Command(exe, "--b0uncer-serve")
	if devNull != nil {
		cmd.Stdout = devNull
		cmd.Stderr = devNull
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	_ = cmd.Start()
}

func main() {
	// Dashboard daemon mode: b0uncer --b0uncer-serve
	if len(os.Args) > 1 && os.Args[1] == "--b0uncer-serve" {
		if err := logger.Init(); err != nil {
			os.Exit(1)
		}
		dashboard.Start(indexHTML)
		select {} // block forever
	}

	home := homeDir()

	// Load config
	realBash := "/bin/bash"
	if data, err := os.ReadFile(filepath.Join(home, ".b0uncer", "config.json")); err == nil {
		var cfg appConfig
		if json.Unmarshal(data, &cfg) == nil && cfg.RealBash != "" {
			realBash = cfg.RealBash
		}
	}

	// Load policies (never returns nil)
	policies, _ := engine.LoadPolicies(filepath.Join(home, ".b0uncer", "policies.json"))

	// Init logger
	_ = logger.Init()

	// Ensure dashboard daemon is running
	if !isDashboardRunning() {
		startDashboardDaemon()
	}

	fmt.Fprintln(os.Stderr, "b0uncer active")

	// Find -c flag
	args := os.Args[1:]
	cmdIdx := -1
	for i, arg := range args {
		if arg == "-c" && i+1 < len(args) {
			cmdIdx = i + 1
			break
		}
	}

	// No -c flag: pass through to real bash unchanged
	if cmdIdx == -1 {
		if err := syscall.Exec(realBash, append([]string{"bash"}, args...), os.Environ()); err != nil {
			fmt.Fprintf(os.Stderr, "b0uncer: exec failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	command := args[cmdIdx]

	// Evaluate
	start := time.Now()
	decision := engine.Evaluate(command, policies)
	durationMs := int(time.Since(start).Milliseconds())

	// Log (always log blocks/warns; log allows only if enabled)
	if decision.Action != "allow" || policies.LogAllowed {
		_ = logger.Log(command, decision.Action, decision.Reason, decision.RiskScore, durationMs)
	}

	// Block
	if decision.Action == "block" {
		fmt.Fprintln(os.Stderr, "B0uncer blocked: "+decision.Reason)
		os.Exit(1)
	}

	// Allow or warn: exec real bash
	if err := syscall.Exec(realBash, []string{"bash", "-c", command}, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "b0uncer: exec failed: %v\n", err)
		os.Exit(1)
	}
}
