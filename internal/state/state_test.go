package state

import (
	"testing"

	"codex-quick-model-switch/internal/config"
)

func TestStoreSwitchPersistsActiveSwitch(t *testing.T) {
	path := t.TempDir() + "/state.json"
	store := NewStore(path)
	sw := config.Switch{Shortcut: "/high", Model: "gpt-5.5", Effort: "high", ServiceTier: config.ServiceTierStandard}

	if err := store.Save(ActiveState{Active: sw}); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	reloaded := NewStore(path)
	got, err := reloaded.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got.Active != sw {
		t.Fatalf("Active = %#v, want %#v", got.Active, sw)
	}
}

func TestStoreReturnsZeroStateWhenMissing(t *testing.T) {
	store := NewStore(t.TempDir() + "/missing/state.json")

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got.Active.Model != "" {
		t.Fatalf("Active model = %q, want empty", got.Active.Model)
	}
}
