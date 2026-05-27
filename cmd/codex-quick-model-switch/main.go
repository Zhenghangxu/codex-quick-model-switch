package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"codex-quick-model-switch/internal/config"
	"codex-quick-model-switch/internal/hook"
	"codex-quick-model-switch/internal/install"
	"codex-quick-model-switch/internal/router"
	"codex-quick-model-switch/internal/service"
	"codex-quick-model-switch/internal/state"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) == 0 {
		return usage()
	}
	switch args[0] {
	case "serve":
		fs := flag.NewFlagSet("serve", flag.ContinueOnError)
		envPath := fs.String("env", "", "path to .env")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		cfg, err := config.Load(*envPath)
		if err != nil {
			return err
		}
		return http.ListenAndServe(cfg.ListenAddr, router.New(cfg, state.NewStore(cfg.StatePath), http.DefaultClient))
	case "hook":
		fs := flag.NewFlagSet("hook", flag.ContinueOnError)
		envPath := fs.String("env", "", "path to .env")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		cfg, err := config.Load(*envPath)
		if err != nil {
			return err
		}
		input, err := io.ReadAll(stdin)
		if err != nil {
			return err
		}
		result, err := hook.Handle(input, cfg, http.DefaultClient, hook.MacOSNotifier)
		if len(result.Stdout) > 0 {
			_, _ = stdout.Write(result.Stdout)
		}
		return err
	case "gen-key":
		key, err := generateKey()
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(stdout, key)
		return err
	case "install":
		binaryPath, err := os.Executable()
		if err != nil {
			return err
		}
		envPath := defaultEnvPath()
		if _, err := os.Stat(envPath); os.IsNotExist(err) {
			if err := writeDefaultEnv(envPath); err != nil {
				return err
			}
		}
		cfg, err := config.Load(envPath)
		if err != nil {
			return err
		}
		return install.Install(install.Options{
			BinaryPath:     binaryPath,
			ListenBaseURL:  cfg.RouterBaseURL + "/v1",
			VirtualModel:   cfg.VirtualModel,
			HookEnvPath:    envPath,
			ProviderName:   "codex-quick-model-switch",
			ProviderAPIKey: "QMS_ROUTER_API_KEY",
			Output:         stdout,
		})
	case "doctor":
		cfg, err := config.Load(defaultEnvPath())
		if err != nil {
			return err
		}
		if err := validateCodexConfig(defaultCodexConfigPath(), cfg.VirtualModel, "codex-quick-model-switch"); err != nil {
			return err
		}
		resp, err := http.Get(cfg.RouterBaseURL + "/healthz")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			return fmt.Errorf("router health returned %s", resp.Status)
		}
		_, err = fmt.Fprintln(stdout, "router healthy")
		return err
	case "service":
		return runService(args[1:])
	default:
		return usage()
	}
}

func validateCodexConfig(path, virtualModel, providerName string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("Codex config %s is missing; add model = %q and model_provider = %q", path, virtualModel, providerName)
	}
	if err != nil {
		return err
	}
	text := string(data)
	if got := topLevelTOMLString(text, "model"); got != virtualModel {
		return fmt.Errorf("Codex config %s must set model = %q; got %q", path, virtualModel, got)
	}
	if got := topLevelTOMLString(text, "model_provider"); got != providerName {
		return fmt.Errorf("Codex config %s must set model_provider = %q; got %q", path, providerName, got)
	}
	return nil
}

func topLevelTOMLString(text, key string) string {
	prefix := key + " "
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			return ""
		}
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}
		left, right, ok := strings.Cut(trimmed, "=")
		if !ok || strings.TrimSpace(left) != key {
			continue
		}
		return trimTOMLString(right)
	}
	return ""
}

func trimTOMLString(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		quote := value[0]
		if (quote == '\'' || quote == '"') && value[len(value)-1] == quote {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func runService(args []string) error {
	if len(args) == 0 {
		return usage()
	}
	switch args[0] {
	case "install":
		binaryPath, err := os.Executable()
		if err != nil {
			return err
		}
		envPath := defaultEnvPath()
		if _, err := os.Stat(envPath); os.IsNotExist(err) {
			if err := writeDefaultEnv(envPath); err != nil {
				return err
			}
		}
		return service.Install(binaryPath, envPath)
	case "start":
		return service.Start()
	case "stop":
		return service.Stop()
	case "status":
		return service.Status()
	case "uninstall":
		return service.Uninstall()
	default:
		return usage()
	}
}

func writeDefaultEnv(path string) error {
	key, err := generateKey()
	if err != nil {
		return err
	}
	upstreamAPIKey := os.Getenv("QMS_UPSTREAM_API_KEY")
	if upstreamAPIKey == "" {
		return fmt.Errorf("QMS_UPSTREAM_API_KEY is required")
	}
	text := fmt.Sprintf("QMS_LISTEN_ADDR=%s\nQMS_ROUTER_API_KEY=%s\nQMS_UPSTREAM_BASE_URL=%s\nQMS_UPSTREAM_API_KEY=%s\nQMS_VIRTUAL_MODEL=%s\nQMS_SWITCHES=%s\n", config.DefaultListenAddr, key, config.DefaultUpstreamBaseURL, upstreamAPIKey, config.DefaultVirtualModel, config.DefaultSwitches)
	return os.WriteFile(path, []byte(text), 0o600)
}

func defaultEnvPath() string {
	return filepath.Join(mustHome(), ".codex-quick-model-switch.env")
}

func defaultCodexConfigPath() string {
	return filepath.Join(mustHome(), ".codex", "config.toml")
}

func generateKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(key), nil
}

func mustHome() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return home
}

func usage() error {
	return fmt.Errorf("usage: codex-quick-model-switch serve|hook|gen-key|service|install|doctor")
}
