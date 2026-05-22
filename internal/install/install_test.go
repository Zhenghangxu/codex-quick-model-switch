package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallBacksUpConfigEnablesHooksAndPreservesHookEntries(t *testing.T) {
	home := t.TempDir()
	codexHome := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexHome, 0o755); err != nil {
		t.Fatalf("mkdir codex home: %v", err)
	}
	if err := os.WriteFile(filepath.Join(codexHome, "config.toml"), []byte("model = \"old\"\n[features]\nhooks = false\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	existingHooks := []byte(`{"hooks":{"Stop":[{"command":"echo keep"}]}}`)
	if err := os.WriteFile(filepath.Join(codexHome, "hooks.json"), existingHooks, 0o644); err != nil {
		t.Fatalf("write hooks: %v", err)
	}

	err := Install(Options{
		HomeDir:        home,
		BinaryPath:     "/usr/local/bin/codex-quick-model-switch",
		ListenBaseURL:  "http://127.0.0.1:8321/v1",
		VirtualModel:   "codex-quick-model-switch",
		BackupSuffix:   ".bak-test",
		HookEnvPath:    filepath.Join(home, ".qms.env"),
		ProviderName:   "codex-quick-model-switch",
		ProviderAPIKey: "local-router",
	})
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(codexHome, "config.toml.bak-test")); err != nil {
		t.Fatalf("config backup missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(codexHome, "hooks.json.bak-test")); err != nil {
		t.Fatalf("hooks backup missing: %v", err)
	}

	configBytes, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	configText := string(configBytes)
	for _, want := range []string{
		`model = 'codex-quick-model-switch'`,
		`model_provider = 'codex-quick-model-switch'`,
		`hooks = true`,
		`base_url = 'http://127.0.0.1:8321/v1'`,
		`wire_api = 'responses'`,
		`requires_openai_auth = true`,
	} {
		if !contains(configText, want) {
			t.Fatalf("config missing %q:\n%s", want, configText)
		}
	}

	hooksBytes, err := os.ReadFile(filepath.Join(codexHome, "hooks.json"))
	if err != nil {
		t.Fatalf("read hooks: %v", err)
	}
	var hooks map[string]any
	if err := json.Unmarshal(hooksBytes, &hooks); err != nil {
		t.Fatalf("hooks json invalid: %v", err)
	}
	if !contains(string(hooksBytes), "echo keep") {
		t.Fatalf("existing hook not preserved:\n%s", hooksBytes)
	}
	if !contains(string(hooksBytes), "UserPromptSubmit") || !contains(string(hooksBytes), "codex-quick-model-switch hook") {
		t.Fatalf("switch hook not installed:\n%s", hooksBytes)
	}
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
