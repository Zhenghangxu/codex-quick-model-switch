package patchjson

import (
	"bytes"
	"testing"

	"codex-quick-model-switch/internal/config"
)

func TestExplicitModelForwardsBodyUnchanged(t *testing.T) {
	body := []byte(`{"model":"gpt-5.5","input":[{"role":"user","content":"hi"}],"metadata":{"a":1}}`)
	out, patched, err := PatchRequest(body, "codex-quick-model-switch", config.Switch{Model: "gpt-5.3-codex", Effort: "medium"})
	if err != nil {
		t.Fatalf("PatchRequest returned error: %v", err)
	}
	if patched {
		t.Fatal("patched explicit model request")
	}
	if !bytes.Equal(out, body) {
		t.Fatalf("body changed:\n got %s\nwant %s", out, body)
	}
}

func TestVirtualResponsesRequestPatchesOnlyModelEffortAndTier(t *testing.T) {
	body := []byte(`{
  "model": "codex-quick-model-switch",
  "input": [{"role":"user","content":[{"type":"input_text","text":"keep me byte-stable"}]}],
  "reasoning": {"summary":"auto","effort":"low"},
  "tools": [{"type":"web_search_preview"}],
  "metadata": {"trace":"abc"},
  "service_tier": "slow"
}`)
	sw := config.Switch{Model: "gpt-5.5", Effort: "high", ServiceTier: config.ServiceTierFast}

	out, patched, err := PatchRequest(body, "codex-quick-model-switch", sw)
	if err != nil {
		t.Fatalf("PatchRequest returned error: %v", err)
	}
	if !patched {
		t.Fatal("expected virtual model request to be patched")
	}

	for _, want := range [][]byte{
		[]byte(`"model": "gpt-5.5"`),
		[]byte(`"reasoning": {"summary":"auto","effort":"high"}`),
		[]byte(`"service_tier": "fast"`),
		[]byte(`"input": [{"role":"user","content":[{"type":"input_text","text":"keep me byte-stable"}]}]`),
		[]byte(`"tools": [{"type":"web_search_preview"}]`),
		[]byte(`"metadata": {"trace":"abc"}`),
	} {
		if !bytes.Contains(out, want) {
			t.Fatalf("patched body missing %s:\n%s", want, out)
		}
	}
}

func TestVirtualChatRequestPatchesReasoningEffortAndRemovesTier(t *testing.T) {
	body := []byte(`{"model":"codex-quick-model-switch","messages":[{"role":"user","content":"hi"}],"reasoning_effort":"high","service_tier":"fast"}`)
	sw := config.Switch{Model: "gpt-5.3-codex", Effort: "medium", ServiceTier: config.ServiceTierNone}

	out, patched, err := PatchRequest(body, "codex-quick-model-switch", sw)
	if err != nil {
		t.Fatalf("PatchRequest returned error: %v", err)
	}
	if !patched {
		t.Fatal("expected patch")
	}
	if !bytes.Contains(out, []byte(`"model":"gpt-5.3-codex"`)) {
		t.Fatalf("model not patched: %s", out)
	}
	if !bytes.Contains(out, []byte(`"reasoning_effort":"medium"`)) {
		t.Fatalf("reasoning_effort not patched: %s", out)
	}
	if bytes.Contains(out, []byte(`service_tier`)) {
		t.Fatalf("service_tier was not removed: %s", out)
	}
}
