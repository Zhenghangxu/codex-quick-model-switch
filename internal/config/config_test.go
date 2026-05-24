package config

import (
	"os"
	"testing"
)

func TestParseDefaultSwitches(t *testing.T) {
	got, err := ParseSwitches(DefaultSwitches)
	if err != nil {
		t.Fatalf("ParseSwitches returned error: %v", err)
	}

	cases := map[string]Switch{
		"/msl":  {Shortcut: "/msl", Model: "gpt-5.3-codex", Effort: "medium", ServiceTier: ServiceTierNone},
		"/msm":  {Shortcut: "/msm", Model: "gpt-5.5", Effort: "medium", ServiceTier: ServiceTierFast},
		"/msh":  {Shortcut: "/msh", Model: "gpt-5.5", Effort: "high", ServiceTier: ServiceTierStandard},
		"/msxh": {Shortcut: "/msxh", Model: "gpt-5.5", Effort: "xhigh", ServiceTier: ServiceTierStandard},
	}

	if len(got) != len(cases) {
		t.Fatalf("got %d switches, want %d", len(got), len(cases))
	}
	for shortcut, want := range cases {
		if got[shortcut] != want {
			t.Fatalf("%s = %#v, want %#v", shortcut, got[shortcut], want)
		}
	}
}

func TestParseSwitchesRejectsInvalidShortcut(t *testing.T) {
	_, err := ParseSwitches("msl=gpt-5.3-codex:medium:none")
	if err == nil {
		t.Fatal("expected invalid shortcut error")
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("HOME", "/Users/example")
	for _, key := range []string{
		"QMS_LISTEN_ADDR",
		"QMS_ROUTER_BASE_URL",
		"QMS_UPSTREAM_BASE_URL",
		"QMS_ROUTER_API_KEY",
		"QMS_UPSTREAM_API_KEY",
		"QMS_VIRTUAL_MODEL",
		"QMS_STATE_PATH",
		"QMS_SWITCHES",
	} {
		t.Setenv(key, "")
	}
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.ListenAddr != DefaultListenAddr {
		t.Fatalf("ListenAddr = %q, want %q", cfg.ListenAddr, DefaultListenAddr)
	}
	if cfg.UpstreamBaseURL != DefaultUpstreamBaseURL {
		t.Fatalf("UpstreamBaseURL = %q, want %q", cfg.UpstreamBaseURL, DefaultUpstreamBaseURL)
	}
	if cfg.VirtualModel != DefaultVirtualModel {
		t.Fatalf("VirtualModel = %q, want %q", cfg.VirtualModel, DefaultVirtualModel)
	}
	if cfg.RouterAPIKey != "" {
		t.Fatalf("RouterAPIKey = %q, want empty default", cfg.RouterAPIKey)
	}
	if cfg.StatePath != "/Users/example/Library/Application Support/codex-quick-model-switch/state.json" {
		t.Fatalf("StatePath = %q", cfg.StatePath)
	}
	if cfg.Switches["/msl"].Model != "gpt-5.3-codex" {
		t.Fatalf("/msl model = %q", cfg.Switches["/msl"].Model)
	}
}

func TestLoadEnvFileIncludesRouterAndUpstreamKeys(t *testing.T) {
	envPath := t.TempDir() + "/qms.env"
	if err := os.WriteFile(envPath, []byte("QMS_ROUTER_API_KEY=router-key\nQMS_UPSTREAM_API_KEY=upstream-key\n"), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}

	cfg, err := Load(envPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.RouterAPIKey != "router-key" {
		t.Fatalf("RouterAPIKey = %q", cfg.RouterAPIKey)
	}
	if cfg.UpstreamAPIKey != "upstream-key" {
		t.Fatalf("UpstreamAPIKey = %q", cfg.UpstreamAPIKey)
	}
}
