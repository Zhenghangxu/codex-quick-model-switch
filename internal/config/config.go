package config

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultListenAddr      = "127.0.0.1:8321"
	DefaultUpstreamBaseURL = "http://localhost:8317/v1"
	DefaultVirtualModel    = "codex-quick-model-switch"
	DefaultSwitches        = "/light=gpt-5.3-codex:medium:none,/medium=gpt-5.5:medium:fast,/high=gpt-5.5:high:standard,/extra-high=gpt-5.5:xhigh:standard"

	ServiceTierNone     = "none"
	ServiceTierFast     = "fast"
	ServiceTierStandard = "standard"
)

type Switch struct {
	Shortcut    string `json:"shortcut"`
	Model       string `json:"model"`
	Effort      string `json:"effort"`
	ServiceTier string `json:"service_tier"`
}

type Config struct {
	ListenAddr      string
	RouterBaseURL   string
	UpstreamBaseURL string
	RouterAPIKey    string
	UpstreamAPIKey  string
	VirtualModel    string
	StatePath       string
	Switches        map[string]Switch
	SwitchOrder     []Switch
}

func Load(envPath string) (Config, error) {
	env := map[string]string{}
	if envPath != "" {
		values, err := readEnvFile(envPath)
		if err != nil {
			return Config{}, err
		}
		env = values
	}

	listenAddr := get("QMS_LISTEN_ADDR", DefaultListenAddr, env)
	switchOrder, switches, err := ParseSwitchList(get("QMS_SWITCHES", DefaultSwitches, env))
	if err != nil {
		return Config{}, err
	}
	upstreamAPIKey := get("QMS_UPSTREAM_API_KEY", "", env)
	if upstreamAPIKey == "" {
		return Config{}, fmt.Errorf("QMS_UPSTREAM_API_KEY is required")
	}

	cfg := Config{
		ListenAddr:      listenAddr,
		RouterBaseURL:   get("QMS_ROUTER_BASE_URL", routerBaseURL(listenAddr), env),
		UpstreamBaseURL: strings.TrimRight(get("QMS_UPSTREAM_BASE_URL", DefaultUpstreamBaseURL, env), "/"),
		RouterAPIKey:    get("QMS_ROUTER_API_KEY", "", env),
		UpstreamAPIKey:  upstreamAPIKey,
		VirtualModel:    get("QMS_VIRTUAL_MODEL", DefaultVirtualModel, env),
		StatePath:       expandHome(get("QMS_STATE_PATH", "~/Library/Application Support/codex-quick-model-switch/state.json", env)),
		Switches:        switches,
		SwitchOrder:     switchOrder,
	}
	return cfg, nil
}

func ParseSwitches(raw string) (map[string]Switch, error) {
	_, switches, err := ParseSwitchList(raw)
	return switches, err
}

func ParseSwitchList(raw string) ([]Switch, map[string]Switch, error) {
	result := map[string]Switch{}
	ordered := []Switch{}
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		left, right, ok := strings.Cut(entry, "=")
		if !ok {
			return nil, nil, fmt.Errorf("switch %q must use shortcut=model:effort:tier", entry)
		}
		shortcut := strings.TrimSpace(left)
		if !strings.HasPrefix(shortcut, "/") || len(shortcut) < 2 || strings.ContainsAny(shortcut, " \t\r\n") {
			return nil, nil, fmt.Errorf("invalid shortcut %q", shortcut)
		}

		parts := strings.Split(right, ":")
		if len(parts) != 3 {
			return nil, nil, fmt.Errorf("switch %q must define model:effort:tier", shortcut)
		}
		sw := Switch{
			Shortcut:    shortcut,
			Model:       strings.TrimSpace(parts[0]),
			Effort:      strings.TrimSpace(parts[1]),
			ServiceTier: strings.TrimSpace(parts[2]),
		}
		if sw.Model == "" {
			return nil, nil, fmt.Errorf("switch %q has empty model", shortcut)
		}
		if !validEffort(sw.Effort) {
			return nil, nil, fmt.Errorf("switch %q has invalid effort %q", shortcut, sw.Effort)
		}
		if !validTier(sw.ServiceTier) {
			return nil, nil, fmt.Errorf("switch %q has invalid service tier %q", shortcut, sw.ServiceTier)
		}
		result[shortcut] = sw
		ordered = append(ordered, sw)
	}
	if len(result) == 0 {
		return nil, nil, fmt.Errorf("no switches configured")
	}
	return ordered, result, nil
}

func readEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(expandHome(path))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	values := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
	}
	return values, scanner.Err()
}

func get(key, fallback string, env map[string]string) string {
	if v, ok := env[key]; ok && v != "" {
		return v
	}
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func routerBaseURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err == nil && (host == "" || host == "0.0.0.0" || host == "::") {
		host = "127.0.0.1"
	}
	if err == nil {
		return (&url.URL{Scheme: "http", Host: net.JoinHostPort(host, port)}).String()
	}
	return "http://" + addr
}

func expandHome(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func validEffort(effort string) bool {
	switch effort {
	case "none", "minimal", "low", "medium", "high", "xhigh":
		return true
	default:
		return false
	}
}

func validTier(tier string) bool {
	switch tier {
	case ServiceTierNone, ServiceTierFast, ServiceTierStandard:
		return true
	default:
		return false
	}
}
