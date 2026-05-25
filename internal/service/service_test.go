package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStartReportsInstallPrerequisiteWhenPlistIsMissing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	err := Start()
	if err == nil {
		t.Fatal("Start returned nil error")
	}
	if !strings.Contains(err.Error(), "service is not installed") {
		t.Fatalf("Start error = %q, want install prerequisite", err)
	}
	if !strings.Contains(err.Error(), "codex-quick-model-switch service install") {
		t.Fatalf("Start error = %q, want service install command", err)
	}
}

func TestServiceUsesAbsoluteSystemToolPaths(t *testing.T) {
	if launchctlPath != "/bin/launchctl" {
		t.Fatalf("launchctlPath = %q, want /bin/launchctl", launchctlPath)
	}
	if idPath != "/usr/bin/id" {
		t.Fatalf("idPath = %q, want /usr/bin/id", idPath)
	}
}

func TestStartIsNoopWhenLaunchAgentIsAlreadyRunning(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	agentsDir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatalf("mkdir agents: %v", err)
	}
	if err := os.WriteFile(plistPath(), []byte("plist"), 0o644); err != nil {
		t.Fatalf("write plist: %v", err)
	}

	originalLaunchctl := launchctl
	t.Cleanup(func() { launchctl = originalLaunchctl })

	var calls [][]string
	launchctl = func(args ...string) error {
		calls = append(calls, append([]string(nil), args...))
		if args[0] == "print" {
			return nil
		}
		t.Fatalf("unexpected launchctl call: %v", args)
		return nil
	}

	if err := Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if len(calls) != 1 || calls[0][0] != "print" {
		t.Fatalf("launchctl calls = %#v, want status print only", calls)
	}
}
