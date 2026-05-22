package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const label = "com.jasonxu.codex-quick-model-switch"

func Install(binaryPath, envPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	agentsDir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		return err
	}
	plistPath := filepath.Join(agentsDir, label+".plist")
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>%s</string>
  <key>ProgramArguments</key>
  <array>
    <string>%s</string>
    <string>serve</string>
    <string>--env</string>
    <string>%s</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>%s</string>
  <key>StandardErrorPath</key><string>%s</string>
</dict>
</plist>
`, label, binaryPath, envPath, filepath.Join(os.TempDir(), label+".out.log"), filepath.Join(os.TempDir(), label+".err.log"))
	return os.WriteFile(plistPath, []byte(plist), 0o644)
}

func Start() error {
	return launchctl("bootstrap", "gui/"+uid(), plistPath())
}

func Stop() error {
	return launchctl("bootout", "gui/"+uid()+"/"+label)
}

func Uninstall() error {
	_ = Stop()
	return os.Remove(plistPath())
}

func Status() error {
	return launchctl("print", "gui/"+uid()+"/"+label)
}

func plistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", label+".plist")
}

func uid() string {
	return strings.TrimSpace(run("id", "-u"))
}

func launchctl(args ...string) error {
	cmd := exec.Command("launchctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func run(name string, args ...string) string {
	out, _ := exec.Command(name, args...).Output()
	return string(out)
}
