package patchjson

import (
	"fmt"

	"codex-quick-model-switch/internal/config"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func PatchRequest(body []byte, virtualModel string, active config.Switch) ([]byte, bool, error) {
	model := gjson.GetBytes(body, "model")
	if !model.Exists() || model.String() != virtualModel {
		return body, false, nil
	}
	if !gjson.ValidBytes(body) {
		return nil, false, fmt.Errorf("invalid json request body")
	}
	if active.Model == "" {
		return nil, false, fmt.Errorf("no active switch")
	}

	out := string(body)
	var err error
	opts := &sjson.Options{Optimistic: true}
	out, err = sjson.SetOptions(out, "model", active.Model, opts)
	if err != nil {
		return nil, false, err
	}

	if gjson.Get(out, "reasoning").Exists() {
		out, err = sjson.SetOptions(out, "reasoning.effort", active.Effort, opts)
	} else {
		out, err = sjson.SetOptions(out, "reasoning_effort", active.Effort, opts)
	}
	if err != nil {
		return nil, false, err
	}

	switch active.ServiceTier {
	case config.ServiceTierNone:
		out, err = sjson.Delete(out, "service_tier")
	case config.ServiceTierFast:
		out, err = sjson.SetOptions(out, "service_tier", "fast", opts)
	case config.ServiceTierStandard:
		out, err = sjson.SetOptions(out, "service_tier", "auto", opts)
	default:
		out, err = sjson.Delete(out, "service_tier")
	}
	if err != nil {
		return nil, false, err
	}

	return []byte(out), true, nil
}
