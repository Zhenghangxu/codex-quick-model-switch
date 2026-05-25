# Codex Quick Model Switch

`codex-quick-model-switch` is a small local router for Codex that lets you switch the model, reasoning effort, and service tier from Raycast using configured switches such as `/light`, `/medium`, `/high`, or `/extra-high`.

This project is only for Codex setups that use an API key through an OpenAI-compatible upstream proxy such as CliProxyAPI. It is not for Codex users who use Codex directly through OAuth without an API-key-backed provider.

It is designed for manual switching only. There is no automatic routing.

<img width="1511" height="952" alt="image" src="https://github.com/user-attachments/assets/57e26c2c-ee5e-4dfb-a441-8792cee3cb47" />


## Quick Start

These steps assume the repo is already cloned, but no Raycast preferences, router env file, LaunchAgent, or Codex provider config exists yet.

1. Build the router binary:

   ```bash
   cd /path/to/codex-quick-model-switch
   make build
   ```

2. Install and run the Raycast extension in development mode:

   ```bash
   cd /path/to/codex-quick-model-switch/raycast-codex-model-switch
   npm install
   npm run dev
   ```

3. Open Raycast, run `Toggle LLM Server`, and fill the extension preferences. Set `Upstream Base URL`, `Upstream API Key`, and `Router Binary` (`/path/to/codex-quick-model-switch/bin/codex-quick-model-switch`); keep the default switches and listen address unless you need to customize them.

4. Run `Toggle LLM Server` again. When the router is stopped, Raycast writes `~/.codex-quick-model-switch.env`, preserves or generates `QMS_ROUTER_API_KEY`, installs and starts the LaunchAgent, and runs `doctor`.

5. Copy the generated `QMS_ROUTER_API_KEY` from `~/.codex-quick-model-switch.env` into `~/.zshrc`:

   ```bash
   export QMS_ROUTER_API_KEY="paste-generated-router-key-here"
   ```

6. Add this provider to `~/.codex/config.toml`, make sure you remove the original `model` and `model_provider`, then restart codex:

   ```toml
   model = "codex-quick-model-switch"
   model_provider = "codex-quick-model-switch"

   [model_providers."codex-quick-model-switch"]
   name = "codex-quick-model-switch"
   base_url = "http://localhost:8321/v1"
   wire_api = "responses"
   env_key = "QMS_ROUTER_API_KEY"
   requires_openai_auth = true
   ```

   Do not add a nested `[model_providers."codex-quick-model-switch".auth]` command block for this provider.

7. Run `Switch Codex Model` from Raycast and choose a model.

Quick validation:

```bash
cd /path/to/codex-quick-model-switch
./bin/codex-quick-model-switch doctor
```

## Default Switches

| Switch | Model | Reasoning effort | Service tier |
| --- | --- | --- | --- |
| `/light` | `gpt-5.3-codex` | `medium` | none |
| `/medium` | `gpt-5.5` | `medium` | `fast` |
| `/high` | `gpt-5.5` | `high` | standard/default |
| `/extra-high` | `gpt-5.5` | `xhigh` | standard/default |

Internally, `none` removes `service_tier`, `fast` sends `service_tier: "fast"`, and `standard` sends the upstream-compatible default value currently used by this tool.

## More Docs

- [Reference](docs/reference.md): architecture, Raycast responsibilities, configuration, commands, cleanup, and manual service usage.
- [Troubleshooting](docs/troubleshooting.md): validation commands, live checks, and auth debugging without printing secrets.

## Development

Run tests:

```bash
go test ./...
```

Build:

```bash
make build
```
