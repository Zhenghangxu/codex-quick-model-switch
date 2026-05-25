package install

import (
	"encoding/json"
	"fmt"
	"io"
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
	Output         io.Writer
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

	if err := backupIfExists(hooksPath, opts.BackupSuffix); err != nil {
		return err
	}
	if err := installHooks(hooksPath, opts); err != nil {
		return err
	}
	return printConfigInstructions(configPath, opts)
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

func printConfigInstructions(path string, opts Options) error {
	out := opts.Output
	if out == nil {
		out = os.Stdout
	}
	_, err := fmt.Fprintf(out, "Please update %s manually with the required Codex quick model switch settings:\n\n%s\n", path, requiredConfigSnippet(opts))
	return err
}

func requiredConfigSnippet(opts Options) string {
	return strings.TrimRight(upsertProvider(setFeatureHooks(setTopLevelString(setTopLevelString("", "model", opts.VirtualModel), "model_provider", opts.ProviderName)), opts), "\n") + "\n"
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
	entry := map[string]any{
		"hooks": []any{
			map[string]any{
				"type":    "command",
				"command": command,
			},
		},
	}

	existing, _ := hooks["UserPromptSubmit"].([]any)
	found := false
	for _, item := range existing {
		if hookGroupContainsCommand(item, command) {
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

func hookGroupContainsCommand(item any, command string) bool {
	group, ok := item.(map[string]any)
	if !ok {
		return false
	}
	if group["command"] == command {
		return true
	}
	handlers, ok := group["hooks"].([]any)
	if !ok {
		return false
	}
	for _, handler := range handlers {
		m, ok := handler.(map[string]any)
		if ok && m["command"] == command {
			return true
		}
	}
	return false
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
		if inFeatures && strings.HasPrefix(trimmed, "codex_hooks =") {
			lines[i] = ""
			continue
		}
		if inFeatures && strings.HasPrefix(trimmed, "hooks =") {
			lines[i] = "hooks = true"
		}
	}
	text = strings.Join(lines, "\n")
	lines = strings.Split(text, "\n")
	if inFeatures {
		for i, line := range lines {
			if strings.TrimSpace(line) == "[features]" {
				for _, candidate := range lines[i+1:] {
					trimmed := strings.TrimSpace(candidate)
					if strings.HasPrefix(trimmed, "[") {
						break
					}
					if strings.HasPrefix(trimmed, "hooks =") {
						return strings.Join(lines, "\n")
					}
				}
				lines = append(lines[:i+1], append([]string{"hooks = true"}, lines[i+1:]...)...)
				return strings.Join(lines, "\n")
			}
		}
	}
	return strings.TrimRight(text, "\n") + "\n\n[features]\nhooks = true\n"
}

func upsertProvider(text string, opts Options) string {
	header := fmt.Sprintf("[model_providers.%q]", opts.ProviderName)
	authHeader := fmt.Sprintf("[model_providers.%q.auth]", opts.ProviderName)
	block := fmt.Sprintf("%s\nname = '%s'\nbase_url = '%s'\nwire_api = 'responses'\nrequires_openai_auth = true\n\n%s\ncommand = '/usr/bin/awk'\nargs = %s\ntimeout_ms = 5000\nrefresh_interval_ms = 300000\n", header, escapeTOML(opts.ProviderName), escapeTOML(opts.ListenBaseURL), authHeader, tomlStringArray([]string{"-F=", `$1=="QMS_ROUTER_API_KEY"{print $2; exit}`, opts.HookEnvPath}))
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

func tomlStringArray(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("'%s'", escapeTOML(value)))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
