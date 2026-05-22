package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Options struct {
	HomeDir        string
	BinaryPath     string
	ListenBaseURL  string
	VirtualModel   string
	BackupSuffix   string
	HookEnvPath    string
	ProviderName   string
	ProviderAPIKey string
}

func Install(opts Options) error {
	if opts.HomeDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		opts.HomeDir = home
	}
	if opts.ProviderName == "" {
		opts.ProviderName = "codex-quick-model-switch"
	}
	if opts.ProviderAPIKey == "" {
		opts.ProviderAPIKey = "QMS_ROUTER_API_KEY"
	}
	if opts.BackupSuffix == "" {
		opts.BackupSuffix = ".bak-" + time.Now().Format("20060102-150405")
	}

	codexHome := filepath.Join(opts.HomeDir, ".codex")
	if err := os.MkdirAll(codexHome, 0o755); err != nil {
		return err
	}
	configPath := filepath.Join(codexHome, "config.toml")
	hooksPath := filepath.Join(codexHome, "hooks.json")

	if err := backupIfExists(configPath, opts.BackupSuffix); err != nil {
		return err
	}
	if err := backupIfExists(hooksPath, opts.BackupSuffix); err != nil {
		return err
	}
	if err := installConfig(configPath, opts); err != nil {
		return err
	}
	return installHooks(hooksPath, opts)
}

func backupIfExists(path, suffix string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return os.WriteFile(path+suffix, data, 0o600)
}

func installConfig(path string, opts Options) error {
	text := ""
	if data, err := os.ReadFile(path); err == nil {
		text = string(data)
	} else if !os.IsNotExist(err) {
		return err
	}

	text = setTopLevelString(text, "model", opts.VirtualModel)
	text = setTopLevelString(text, "model_provider", opts.ProviderName)
	text = setFeatureHooks(text)
	text = upsertProvider(text, opts)
	return os.WriteFile(path, []byte(strings.TrimRight(text, "\n")+"\n"), 0o600)
}

func installHooks(path string, opts Options) error {
	root := map[string]any{}
	if data, err := os.ReadFile(path); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		if err := json.Unmarshal(data, &root); err != nil {
			return err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	hooks, _ := root["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
		root["hooks"] = hooks
	}
	command := strings.TrimSpace(opts.BinaryPath + " hook --env " + opts.HookEnvPath)
	entry := map[string]any{"command": command}

	existing, _ := hooks["UserPromptSubmit"].([]any)
	found := false
	for _, item := range existing {
		if m, ok := item.(map[string]any); ok && m["command"] == command {
			found = true
			break
		}
	}
	if !found {
		existing = append(existing, entry)
	}
	hooks["UserPromptSubmit"] = existing

	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func setTopLevelString(text, key, value string) string {
	lines := strings.Split(text, "\n")
	prefix := key + " ="
	replacement := fmt.Sprintf("%s = '%s'", key, escapeTOML(value))
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			break
		}
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			lines[i] = replacement
			return strings.Join(lines, "\n")
		}
	}
	return replacement + "\n" + text
}

func setFeatureHooks(text string) string {
	lines := strings.Split(text, "\n")
	inFeatures := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[features]" {
			inFeatures = true
			continue
		}
		if inFeatures && strings.HasPrefix(trimmed, "[") {
			break
		}
		if inFeatures && strings.HasPrefix(trimmed, "hooks =") {
			lines[i] = "hooks = true"
			return strings.Join(lines, "\n")
		}
	}
	if inFeatures {
		for i, line := range lines {
			if strings.TrimSpace(line) == "[features]" {
				lines = append(lines[:i+1], append([]string{"hooks = true"}, lines[i+1:]...)...)
				return strings.Join(lines, "\n")
			}
		}
	}
	return strings.TrimRight(text, "\n") + "\n\n[features]\nhooks = true\n"
}

func upsertProvider(text string, opts Options) string {
	header := fmt.Sprintf("[model_providers.%q]", opts.ProviderName)
	block := fmt.Sprintf("%s\nname = '%s'\nbase_url = '%s'\nwire_api = 'responses'\nenv_key = '%s'\nrequires_openai_auth = true\n", header, escapeTOML(opts.ProviderName), escapeTOML(opts.ListenBaseURL), escapeTOML(opts.ProviderAPIKey))
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != header {
			continue
		}
		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if strings.HasPrefix(strings.TrimSpace(lines[j]), "[") {
				end = j
				break
			}
		}
		return strings.Join(append(append(lines[:i], strings.Split(strings.TrimRight(block, "\n"), "\n")...), lines[end:]...), "\n")
	}
	return strings.TrimRight(text, "\n") + "\n\n" + block
}

func escapeTOML(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}
