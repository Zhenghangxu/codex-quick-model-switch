package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorLoadsDefaultEnvFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("QMS_UPSTREAM_API_KEY", "")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			t.Fatalf("path = %q, want /healthz", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	envPath := filepath.Join(home, ".codex-quick-model-switch.env")
	env := strings.Join([]string{
		"QMS_UPSTREAM_API_KEY=upstream-key",
		"QMS_ROUTER_BASE_URL=" + server.URL,
		"",
	}, "\n")
	if err := os.WriteFile(envPath, []byte(env), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}

	var out bytes.Buffer
	if err := run([]string{"doctor"}, strings.NewReader(""), &out); err != nil {
		t.Fatalf("doctor returned error: %v", err)
	}
	if strings.TrimSpace(out.String()) != "router healthy" {
		t.Fatalf("doctor output = %q", out.String())
	}
}

func TestWriteDefaultEnvRequiresUpstreamAPIKey(t *testing.T) {
	t.Setenv("QMS_UPSTREAM_API_KEY", "")

	err := writeDefaultEnv(filepath.Join(t.TempDir(), "qms.env"))
	if err == nil {
		t.Fatal("expected missing upstream api key error")
	}
	if err.Error() != "QMS_UPSTREAM_API_KEY is required" {
		t.Fatalf("writeDefaultEnv error = %q", err.Error())
	}
}

func TestWriteDefaultEnvIncludesUpstreamAPIKey(t *testing.T) {
	t.Setenv("QMS_UPSTREAM_API_KEY", "upstream-key")
	envPath := filepath.Join(t.TempDir(), "qms.env")

	if err := writeDefaultEnv(envPath); err != nil {
		t.Fatalf("writeDefaultEnv returned error: %v", err)
	}
	text, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read env: %v", err)
	}
	if !strings.Contains(string(text), "QMS_UPSTREAM_API_KEY=upstream-key\n") {
		t.Fatalf("env file missing upstream key:\n%s", text)
	}
}
