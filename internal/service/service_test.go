package service

import (
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
