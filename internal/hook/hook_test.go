package hook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"codex-quick-model-switch/internal/config"
)

func TestHandleNoMatchProducesNoOutput(t *testing.T) {
	cfg := config.Config{Switches: map[string]config.Switch{"/msl": {Shortcut: "/msl"}}}
	result, err := Handle([]byte(`{"prompt":"hello"}`), cfg, nil, nil)
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if result.Handled {
		t.Fatalf("Handled = true")
	}
	if len(result.Stdout) != 0 {
		t.Fatalf("Stdout = %s, want empty", result.Stdout)
	}
}

func TestHandleExactShortcutCallsRouterAndBlocksPrompt(t *testing.T) {
	var gotShortcut string
	router := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/switch" {
			t.Fatalf("path = %s, want /switch", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer router-key" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		var req SwitchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode switch request: %v", err)
		}
		gotShortcut = req.Shortcut
		w.WriteHeader(http.StatusNoContent)
	}))
	defer router.Close()

	var notified string
	cfg := config.Config{
		RouterBaseURL: router.URL,
		RouterAPIKey:  "router-key",
		Switches:      map[string]config.Switch{"/msh": {Shortcut: "/msh", Model: "gpt-5.5", Effort: "high"}},
	}
	result, err := Handle([]byte(`{"prompt":" /msh "}`), cfg, http.DefaultClient, func(title, body string) error {
		notified = title + " " + body
		return nil
	})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if gotShortcut != "/msh" {
		t.Fatalf("router shortcut = %q", gotShortcut)
	}
	if !result.Handled {
		t.Fatal("Handled = false")
	}
	if string(result.Stdout) != `{"decision":"block","reason":"Switched Codex model to gpt-5.5 (high)."}`+"\n" {
		t.Fatalf("Stdout = %s", result.Stdout)
	}
	if notified == "" {
		t.Fatal("expected quiet notification")
	}
}

func TestHandleRouterFailureStillBlocksShortcut(t *testing.T) {
	router := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer router.Close()

	cfg := config.Config{
		RouterBaseURL: router.URL,
		Switches:      map[string]config.Switch{"/msl": {Shortcut: "/msl", Model: "gpt-5.3-codex", Effort: "medium"}},
	}
	result, err := Handle([]byte(`{"prompt":"/msl"}`), cfg, http.DefaultClient, nil)
	if err == nil {
		t.Fatal("expected router failure")
	}
	if !result.Handled {
		t.Fatal("Handled = false")
	}
	if string(result.Stdout) != `{"decision":"block","reason":"Model switch failed; command was not sent to the model."}`+"\n" {
		t.Fatalf("Stdout = %s", result.Stdout)
	}
}
