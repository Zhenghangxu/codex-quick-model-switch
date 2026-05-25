package router

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"codex-quick-model-switch/internal/config"
	"codex-quick-model-switch/internal/state"
)

func TestSwitchAndStateEndpoints(t *testing.T) {
	cfg := config.Config{
		Switches: map[string]config.Switch{
			"/light": {Shortcut: "/light", Model: "gpt-5.3-codex", Effort: "medium", ServiceTier: config.ServiceTierNone},
		},
	}
	handler := New(cfg, state.NewStore(t.TempDir()+"/state.json"), nil)

	req := httptest.NewRequest(http.MethodPost, "/switch", strings.NewReader(`{"shortcut":"/light"}`))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("POST /switch status = %d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/state", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /state status = %d body=%s", rr.Code, rr.Body.String())
	}
	var got state.ActiveState
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	if got.Active.Model != "gpt-5.3-codex" {
		t.Fatalf("active model = %q", got.Active.Model)
	}
}

func TestSwitchesEndpointReturnsOrderedSwitchesAndActiveState(t *testing.T) {
	cfg := config.Config{
		RouterAPIKey: "router-key",
		Switches: map[string]config.Switch{
			"/deep": {Shortcut: "/deep", Model: "gpt-5.5", Effort: "xhigh", ServiceTier: config.ServiceTierStandard},
			"/mini": {Shortcut: "/mini", Model: "gpt-5.4-mini", Effort: "low", ServiceTier: config.ServiceTierFast},
		},
		SwitchOrder: []config.Switch{
			{Shortcut: "/mini", Model: "gpt-5.4-mini", Effort: "low", ServiceTier: config.ServiceTierFast},
			{Shortcut: "/deep", Model: "gpt-5.5", Effort: "xhigh", ServiceTier: config.ServiceTierStandard},
		},
	}
	store := state.NewStore(t.TempDir() + "/state.json")
	if err := store.Save(state.ActiveState{Active: cfg.Switches["/deep"]}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	handler := New(cfg, store, nil)

	req := httptest.NewRequest(http.MethodGet, "/switches", nil)
	req.Header.Set("Authorization", "Bearer router-key")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GET /switches status = %d body=%s", rr.Code, rr.Body.String())
	}
	var got struct {
		Switches []config.Switch `json:"switches"`
		Active   config.Switch   `json:"active"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode switches: %v", err)
	}
	if len(got.Switches) != 2 {
		t.Fatalf("switch count = %d", len(got.Switches))
	}
	if got.Switches[0].Shortcut != "/mini" || got.Switches[1].Shortcut != "/deep" {
		t.Fatalf("switch order = %#v", got.Switches)
	}
	if got.Active.Shortcut != "/deep" {
		t.Fatalf("active shortcut = %q", got.Active.Shortcut)
	}
}

func TestProxyPatchesVirtualModelAndForwardsExplicitModelUnchanged(t *testing.T) {
	var bodies []string
	var authHeaders []string
	var upstreamQueries []string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(body))
		authHeaders = append(authHeaders, r.Header.Get("Authorization"))
		upstreamQueries = append(upstreamQueries, r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	cfg := config.Config{
		UpstreamBaseURL: upstream.URL + "/v1",
		RouterAPIKey:    "router-key",
		UpstreamAPIKey:  "upstream-key",
		VirtualModel:    "codex-quick-model-switch",
		Switches: map[string]config.Switch{
			"/medium": {Shortcut: "/medium", Model: "gpt-5.5", Effort: "medium", ServiceTier: config.ServiceTierFast},
		},
	}
	store := state.NewStore(t.TempDir() + "/state.json")
	if err := store.Save(state.ActiveState{Active: cfg.Switches["/medium"]}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	handler := New(cfg, store, http.DefaultClient)

	virtualBody := `{"model":"codex-quick-model-switch","input":"keep","reasoning":{"effort":"low","summary":"auto"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/responses?beta=true", strings.NewReader(virtualBody))
	req.Header.Set("Authorization", "Bearer router-key")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("virtual proxy status = %d body=%s", rr.Code, rr.Body.String())
	}

	explicitBody := `{"model":"gpt-5.3-codex","input":"keep"}`
	req = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(explicitBody))
	req.Header.Set("Authorization", "Bearer router-key")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("explicit proxy status = %d body=%s", rr.Code, rr.Body.String())
	}

	if len(bodies) != 2 {
		t.Fatalf("upstream saw %d requests", len(bodies))
	}
	if !strings.Contains(bodies[0], `"model":"gpt-5.5"`) || !strings.Contains(bodies[0], `"service_tier":"fast"`) {
		t.Fatalf("virtual request was not patched correctly: %s", bodies[0])
	}
	if bodies[1] != explicitBody {
		t.Fatalf("explicit request changed:\n got %s\nwant %s", bodies[1], explicitBody)
	}
	if authHeaders[0] != "Bearer upstream-key" || authHeaders[1] != "Bearer upstream-key" {
		t.Fatalf("upstream auth headers = %#v", authHeaders)
	}
	if upstreamQueries[0] != "beta=true" {
		t.Fatalf("upstream query = %q", upstreamQueries[0])
	}
}

func TestProxyRejectsInvalidRouterKey(t *testing.T) {
	cfg := config.Config{
		RouterAPIKey:    "router-key",
		UpstreamBaseURL: "http://127.0.0.1:9/v1",
		VirtualModel:    "codex-quick-model-switch",
	}
	handler := New(cfg, state.NewStore(t.TempDir()+"/state.json"), http.DefaultClient)

	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"gpt-5.5"}`))
	req.Header.Set("Authorization", "Bearer wrong")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
}
