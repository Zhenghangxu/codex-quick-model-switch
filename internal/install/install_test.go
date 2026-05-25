package install

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallPrintsConfigChangeWithoutTouchingConfigAndPreservesHookEntries(t *testing.T) {
	home := t.TempDir()
	codexHome := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexHome, 0o755); err != nil {
		t.Fatalf("mkdir codex home: %v", err)
	}
	originalConfig := []byte("model = \"old\"\n[features]\nhooks = false\ncodex_hooks = false\n")
	if err := os.WriteFile(filepath.Join(codexHome, "config.toml"), originalConfig, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	existingHooks := []byte(`{"hooks":{"Stop":[{"command":"echo keep"}]}}`)
	if err := os.WriteFile(filepath.Join(codexHome, "hooks.json"), existingHooks, 0o644); err != nil {
		t.Fatalf("write hooks: %v", err)
	}

	var out bytes.Buffer
	err := Install(Options{
		HomeDir:        home,
		BinaryPath:     "/usr/local/bin/codex-quick-model-switch",
		ListenBaseURL:  "http://127.0.0.1:8321/v1",
		VirtualModel:   "codex-quick-model-switch",
		BackupSuffix:   ".bak-test",
		HookEnvPath:    filepath.Join(home, ".qms.env"),
		ProviderName:   "codex-quick-model-switch",
		ProviderAPIKey: "local-router",
		Output:         &out,
	})
	if err != nil {
		t.Fatalf("Install returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(codexHome, "config.toml.bak-test")); !os.IsNotExist(err) {
		t.Fatalf("config backup should not be created, stat error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(codexHome, "hooks.json.bak-test")); err != nil {
		t.Fatalf("hooks backup missing: %v", err)
	}

	configBytes, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !bytes.Equal(configBytes, originalConfig) {
		t.Fatalf("config.toml was modified:\n%s", configBytes)
	}

	output := out.String()
	for _, want := range []string{
		"Please update",
		filepath.Join(codexHome, "config.toml"),
		`model = 'codex-quick-model-switch'`,
		`model_provider = 'codex-quick-model-switch'`,
		`hooks = true`,
		`base_url = 'http://127.0.0.1:8321/v1'`,
		`wire_api = 'responses'`,
		`env_key = 'local-router'`,
		`requires_openai_auth = true`,
	} {
		if !contains(output, want) {
			t.Fatalf("installer output missing %q:\n%s", want, output)
		}
	}
	for _, unwanted := range []string{
		`[model_providers."codex-quick-model-switch".auth]`,
		`command = '/usr/bin/awk'`,
		filepath.Join(home, ".qms.env"),
	} {
		if contains(output, unwanted) {
			t.Fatalf("installer output should not contain %q:\n%s", unwanted, output)
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

	rootHooks := hooks["hooks"].(map[string]any)
	userPromptSubmit := rootHooks["UserPromptSubmit"].([]any)
	group := userPromptSubmit[0].(map[string]any)
	if _, ok := group["command"]; ok {
		t.Fatalf("UserPromptSubmit entry used obsolete flat command shape:\n%s", hooksBytes)
	}
	handlers := group["hooks"].([]any)
	handler := handlers[0].(map[string]any)
	if handler["type"] != "command" || !contains(handler["command"].(string), "codex-quick-model-switch hook") {
		t.Fatalf("UserPromptSubmit command hook not installed with current schema:\n%s", hooksBytes)
	}
}

func TestUpsertProviderUsesEnvKeyAndRemovesCommandBackedAuth(t *testing.T) {
	input := `model = 'old'

[model_providers."codex-quick-model-switch"]
name = 'old'
base_url = 'http://old/v1'
wire_api = 'responses'
requires_openai_auth = true

[model_providers."codex-quick-model-switch".auth]
command = '/usr/bin/awk'
args = ['-F=', '$1=="QMS_ROUTER_API_KEY"{print $2; exit}', '/Users/example/.codex-quick-model-switch.env']
timeout_ms = 5000
refresh_interval_ms = 300000

[model_providers.other]
name = 'other'
`

	output := upsertProvider(input, Options{
		ProviderName:   "codex-quick-model-switch",
		ListenBaseURL:  "http://127.0.0.1:8321/v1",
		ProviderAPIKey: "QMS_ROUTER_API_KEY",
	})

	for _, want := range []string{
		`[model_providers."codex-quick-model-switch"]`,
		`base_url = 'http://127.0.0.1:8321/v1'`,
		`env_key = 'QMS_ROUTER_API_KEY'`,
		`requires_openai_auth = true`,
		`[model_providers.other]`,
	} {
		if !contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
	for _, unwanted := range []string{
		`[model_providers."codex-quick-model-switch".auth]`,
		`command = '/usr/bin/awk'`,
		`timeout_ms = 5000`,
	} {
		if contains(output, unwanted) {
			t.Fatalf("output should not contain %q:\n%s", unwanted, output)
		}
	}
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
