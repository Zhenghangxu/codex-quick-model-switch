# Codex Quick Model Switch Plan

## Summary
Build a small Go tool in `/path/to/codex-quick-model-switch` that uses the official Codex `UserPromptSubmit` hook contract from [OpenAI Codex hooks](https://developers.openai.com/codex/hooks). The hook detects configured switch shortcuts such as `/msl`, `/msm`, `/msh`, and `/msxh`, calls a local router, sends a quiet macOS notification, and returns `{"decision":"block"}` so the switch command never reaches the model.

The router will use a minimum-impact request patching strategy: no prompt extraction, no classifier, no full request rebuild. It will surgically modify only the model name, thinking effort, and service tier for virtual-model requests, preserving all prompt, tools, metadata, cache keys, and request ordering as much as possible for maximum prompt-cache stability.

## Key Changes
- Create a Go module with one binary, `codex-quick-model-switch`, exposing:
  - `serve --env .env`
  - `hook --env .env`
  - `gen-key`
  - `service install/start/stop/status/uninstall`
  - `install`
  - `doctor`
- Router endpoints:
  - `GET /healthz`
  - `GET /state`
  - `POST /switch`
  - `/v1/responses` and `/v1/chat/completions` proxy endpoints.
- Request patching:
  - If `model` is not the virtual model, forward the request byte-for-byte unchanged.
  - If `model` is `codex-quick-model-switch`, patch only:
    - `model`
    - `reasoning.effort` for Responses API requests
    - `reasoning_effort` for Chat Completions compatibility
    - top-level `service_tier` when the active switch config sets or clears a service tier
  - Preserve existing `reasoning.summary` and every unrelated field.
  - Treat service tier as part of the switch config:
    - `none` removes or omits `service_tier`.
    - `fast` writes `service_tier: "fast"`.
    - `standard` means normal standard processing; emit the upstream-compatible standard/default tier value required by the selected provider instead of blindly writing an unsupported literal.
  - Avoid `encoding/json` re-marshal for proxied request bodies; use a small tested JSON patch layer, likely backed by `gjson/sjson`, with byte-preservation tests.
- Env defaults:
  - `QMS_LISTEN_ADDR=127.0.0.1:8321`
  - `QMS_UPSTREAM_BASE_URL=http://localhost:8317/v1`
  - `QMS_VIRTUAL_MODEL=codex-quick-model-switch`
  - `QMS_SWITCHES=/msl=gpt-5.3-codex:medium:none,/msm=gpt-5.5:medium:fast,/msh=gpt-5.5:high:standard,/msxh=gpt-5.5:xhigh:standard`
  - `QMS_STATE_PATH=~/Library/Application Support/codex-quick-model-switch/state.json`
- Global install:
  - Back up `~/.codex/config.toml` and `~/.codex/hooks.json`.
  - Enable canonical `features.hooks = true`.
  - Preserve existing hook entries.
  - Point default Codex model/provider at the local router.

## Test Plan
- Unit test hook behavior: no match, exact `/msl`, `/msm`, `/msh`, `/msxh`, router failure, invalid shortcut, and quiet notification.
- Unit test state switching and persistence.
- Unit test switch config parsing for all four default shortcuts, including `/msl` as `gpt-5.3-codex` with medium reasoning and no service tier.
- Unit test JSON patching with byte-level assertions that `input`, `tools`, `metadata`, and unrelated fields are unchanged while `service_tier` is set, normalized, or removed according to switch config.
- Integration test proxy forwarding: virtual model patches only model/effort/service tier; explicit model forwards unchanged.
- Installer tests with temp config/hook files: backups created, hooks enabled, existing hooks preserved, provider updated.
- Manual verification:
  - `go test ./...`
  - `make build`
  - `curl http://127.0.0.1:8321/healthz`
  - `doctor`
  - Send `/msl`, `/msm`, `/msh`, and `/msxh` in Codex and verify the next normal prompt routes to the expected model, thinking effort, and service-tier behavior.

## Assumptions
- v1 uses global switch state.
- v1 ships these default shortcuts:
  - `/msl -> gpt-5.3-codex:medium:none`
  - `/msm -> gpt-5.5:medium:fast`
  - `/msh -> gpt-5.5:high:standard`
  - `/msxh -> gpt-5.5:xhigh:standard`
- No Groq, classifier, task extraction, prompt inspection, or automatic switching.
- The router’s cache-friendly rule is strict: only model, thinking effort, and service tier may change during normal proxying.
