package hook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"codex-quick-model-switch/internal/config"
)

type SwitchRequest struct {
	Shortcut string `json:"shortcut"`
}

type Result struct {
	Handled bool
	Stdout  []byte
}

type Notifier func(title, body string) error

func Handle(input []byte, cfg config.Config, client *http.Client, notify Notifier) (Result, error) {
	var event struct {
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal(input, &event); err != nil {
		return Result{}, err
	}

	shortcut := strings.TrimSpace(event.Prompt)
	sw, ok := cfg.Switches[shortcut]
	if !ok {
		return Result{}, nil
	}
	if client == nil {
		client = http.DefaultClient
	}

	body, _ := json.Marshal(SwitchRequest{Shortcut: shortcut})
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(cfg.RouterBaseURL, "/")+"/switch", bytes.NewReader(body))
	if err != nil {
		return Result{Handled: true, Stdout: blockJSON("Model switch failed; command was not sent to the model.")}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.RouterAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.RouterAPIKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return Result{Handled: true, Stdout: blockJSON("Model switch failed; command was not sent to the model.")}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{Handled: true, Stdout: blockJSON("Model switch failed; command was not sent to the model.")}, fmt.Errorf("router returned %s", resp.Status)
	}

	message := fmt.Sprintf("Switched Codex model to %s (%s).", sw.Model, sw.Effort)
	if notify != nil {
		_ = notify("Codex model switched", message)
	}
	return Result{Handled: true, Stdout: blockJSON(message)}, nil
}

func MacOSNotifier(title, body string) error {
	return exec.Command("osascript", "-e", fmt.Sprintf(`display notification %q with title %q`, body, title)).Run()
}

func blockJSON(reason string) []byte {
	data, _ := json.Marshal(map[string]string{
		"decision": "block",
		"reason":   reason,
	})
	return append(data, '\n')
}
