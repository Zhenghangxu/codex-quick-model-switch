# Codex Quick Model Switch

`codex-quick-model-switch` is a small local router for Codex that lets you switch the model, reasoning effort, and service tier from Raycast using configured switches such as `/light`, `/medium`, `/high`, or `/extra-high`.

This project is only for Codex setups that use an API key through an OpenAI-compatible upstream proxy such as CliProxyAPI. It is not for Codex users who use Codex directly through OAuth without an API-key-backed provider.

It is designed for manual switching only. There is no automatic routing.

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

## Raycast Workflow

The recommended UI is the bundled Raycast extension in:

```text
/path/to/codex-quick-model-switch/raycast-codex-model-switch
```

Raycast is responsible for:

- collecting the upstream LLM endpoint, required API key, listen address, binary path, and switches in Raycast preferences
- writing `~/.codex-quick-model-switch.env` as a generated runtime file
- installing, starting, and stopping the macOS LaunchAgent
- listing configured model switches
- switching the active model without typing a prompt into Codex

Raycast preferences are the source of truth for editable settings. Treat `~/.codex-quick-model-switch.env` as private generated plumbing for the LaunchAgent and Codex auth lookup, not as a user-managed config file.

The Go router is still the Codex-facing proxy. Codex sends requests to `http://localhost:8321/v1`; Raycast only configures and controls the router.

## How It Works

The tool has two moving parts:

1. A local router listens on `127.0.0.1:8321` and proxies Codex model requests upstream.
2. The Raycast extension configures the router and switches the active model through local HTTP endpoints.

When you choose a switch in Raycast, the extension calls the local router, updates the active switch state, and refreshes the status view. Your next Codex request is routed using the active switch.

For normal proxying, the router keeps requests as stable as possible. If the request model is not the virtual model, the body is forwarded unchanged. If the request model is `codex-quick-model-switch`, only these fields are patched:

- `model`
- `reasoning.effort` for Responses API requests
- `reasoning_effort` for Chat Completions compatibility
- top-level `service_tier`

Everything else, including prompt input, tools, metadata, and unrelated fields, is preserved.

## Install With Raycast

Build the router binary:

```bash
cd /path/to/codex-quick-model-switch
make build
```

Install the Raycast extension in development mode:

```bash
cd /path/to/codex-quick-model-switch/raycast-codex-model-switch
npm install
npm run dev
```

Open Raycast with `Command+Space`, search for `Toggle LLM Server`, and open the extension preferences when Raycast prompts for them.

Set:

- `Upstream Base URL`: your OpenAI-compatible upstream endpoint, for example `http://localhost:8317/v1`
- `Upstream API Key`: required bearer token for that upstream endpoint
- `Router Binary`: `/path/to/codex-quick-model-switch/bin/codex-quick-model-switch`
- `Model Switches`: comma-separated `/shortcut=model:effort:service_tier` mappings
- `Router Listen Address`: `127.0.0.1:8321`

Run `Toggle LLM Server` from Raycast. When the LaunchAgent is stopped, it writes `~/.codex-quick-model-switch.env` from Raycast preferences, preserves or generates `QMS_ROUTER_API_KEY`, installs the LaunchAgent, starts it, and runs `doctor`. When the LaunchAgent is already running, it stops it.

Then run `Switch Codex Model` from Raycast and choose the active model. `Codex Model Switch Status` shows the LaunchAgent state, router health, active model, env path, and binary path.

## Configure Codex

Codex must use the local router as a user-level model provider. Put this in `~/.codex/config.toml`:

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

`env_key = "QMS_ROUTER_API_KEY"` tells Codex which environment variable contains the local router bearer token.

Important: do not add a nested `[model_providers."codex-quick-model-switch".auth]` command block for this provider. The local working configuration uses `env_key = "QMS_ROUTER_API_KEY"` together with `requires_openai_auth = true`.

Restart Codex after changing `config.toml`.

## Validate

Run the automated checks:

```bash
cd /path/to/codex-quick-model-switch
go test ./...
make build

cd /path/to/codex-quick-model-switch/raycast-codex-model-switch
npm test
npm run build
npm run lint
```

Run live checks:

```bash
cd /path/to/codex-quick-model-switch
./bin/codex-quick-model-switch service status
./bin/codex-quick-model-switch doctor

ROUTER_KEY="$(awk -F= '$1=="QMS_ROUTER_API_KEY"{print $2; exit}' ~/.codex-quick-model-switch.env)"
curl -fsS -H "Authorization: Bearer $ROUTER_KEY" http://127.0.0.1:8321/switches
curl -fsS -X POST -H "Authorization: Bearer $ROUTER_KEY" -H "Content-Type: application/json" -d '{"shortcut":"/medium"}' http://127.0.0.1:8321/switch
curl -fsS -H "Authorization: Bearer $ROUTER_KEY" http://127.0.0.1:8321/state
```

For UI validation, launch Raycast with `Command+Space`, run `Codex Model Switch Status`, then run `Switch Codex Model` and choose `/medium` or another configured model.

## Cleanup

Stop and remove the LaunchAgent:

```bash
cd /path/to/codex-quick-model-switch
./bin/codex-quick-model-switch service stop
./bin/codex-quick-model-switch service uninstall
```

Remove local router files if you no longer need them:

```bash
rm -f ~/.codex-quick-model-switch.env
rm -rf ~/Library/Application\ Support/codex-quick-model-switch
```

Remove the Raycast development extension from Raycast Preferences > Extensions.

To stop using the router from Codex, edit `~/.codex/config.toml` and set `model` / `model_provider` back to your normal provider, then remove the `[model_providers."codex-quick-model-switch"]` block.

## Start The Router

Install and start the macOS LaunchAgent:

```bash
./bin/codex-quick-model-switch service install
./bin/codex-quick-model-switch service start
```

Check status:

```bash
./bin/codex-quick-model-switch service status
```

Check router health:

```bash
./bin/codex-quick-model-switch doctor
```

You can also run the router manually:

```bash
./bin/codex-quick-model-switch serve --env ~/.codex-quick-model-switch.env
```

## Configure Codex Authentication

The router protects local endpoints with `QMS_ROUTER_API_KEY`.

Raycast owns the editable router settings. The env file below is generated whenever `Toggle LLM Server` starts the service, and should not be manually edited.

Raycast and the service CLI can create this key in:

```text
~/.codex-quick-model-switch.env
```

Codex also needs the same value when it sends requests to the local provider. Use `env_key = "QMS_ROUTER_API_KEY"` in `~/.codex/config.toml`:

```toml
[model_providers."codex-quick-model-switch"]
name = "codex-quick-model-switch"
base_url = "http://localhost:8321/v1"
wire_api = "responses"
env_key = "QMS_ROUTER_API_KEY"
requires_openai_auth = true
```

This is the expected local configuration shape for this project.

Avoid adding a nested command-backed auth section:

```toml
[model_providers."codex-quick-model-switch".auth]
command = "/usr/bin/awk"
```

That shape has caused local Codex provider auth problems for this project.

Restart Codex after changing config so it reloads `config.toml`.

## Use It

Once the router is running and Codex has reloaded config:

1. Run `Switch Codex Model` from Raycast.
2. Choose one of the configured switches, such as `/medium`.
3. Send your next normal prompt in Codex. It will use the selected model settings.

For example, choosing `/light` selects `gpt-5.3-codex` with medium reasoning and no service tier. Choosing `/high` selects `gpt-5.5` with high reasoning and the standard/default service tier.

## Configuration

Configuration is read from environment variables and from an optional `.env` file passed with `--env`.

Default env file:

```text
~/.codex-quick-model-switch.env
```

Supported variables:

| Variable | Default | Purpose |
| --- | --- | --- |
| `QMS_LISTEN_ADDR` | `127.0.0.1:8321` | Local router listen address |
| `QMS_ROUTER_BASE_URL` | derived from listen address | Base URL used by doctor and local router clients |
| `QMS_ROUTER_API_KEY` | generated by Raycast or service CLI; preserved when already present | Bearer token required by router endpoints |
| `QMS_UPSTREAM_BASE_URL` | `http://localhost:8317/v1` | Upstream OpenAI-compatible proxy base URL |
| `QMS_UPSTREAM_API_KEY` | required | Bearer token for the upstream proxy |
| `QMS_VIRTUAL_MODEL` | `codex-quick-model-switch` | Virtual model name Codex sends to this router |
| `QMS_SWITCHES` | default switch list | Comma-separated switch mapping |
| `QMS_STATE_PATH` | `~/Library/Application Support/codex-quick-model-switch/state.json` | Active switch state file |

`QMS_SWITCHES` format:

```text
/switch=model:reasoning_effort:service_tier
```

Example:

```text
QMS_SWITCHES=/mini=gpt-5.4-mini:low:fast,/deep=gpt-5.5:xhigh:standard
```

Allowed reasoning efforts:

```text
minimal, low, medium, high, xhigh
```

Allowed service tiers:

```text
none, fast, standard
```

## Commands

```bash
./bin/codex-quick-model-switch serve --env ~/.codex-quick-model-switch.env
./bin/codex-quick-model-switch gen-key
./bin/codex-quick-model-switch doctor
./bin/codex-quick-model-switch service install
./bin/codex-quick-model-switch service start
./bin/codex-quick-model-switch service stop
./bin/codex-quick-model-switch service status
./bin/codex-quick-model-switch service uninstall
```

## Troubleshooting

If `doctor` fails:

- make sure the service is started
- check `QMS_LISTEN_ADDR`
- try running `serve --env ~/.codex-quick-model-switch.env` manually

If Codex says the provider is unauthorized:

- keep `env_key = "QMS_ROUTER_API_KEY"` in `~/.codex/config.toml`
- do not add a nested `[model_providers."codex-quick-model-switch".auth]` command block
- make sure the router key visible to Codex is the same value as `~/.codex-quick-model-switch.env`
- restart Codex after changing environment or config

To validate without printing secrets:

```bash
env_file_value="$(awk -F= '$1=="QMS_ROUTER_API_KEY"{print $2; exit}' ~/.codex-quick-model-switch.env)"
zsh_value="$(zsh -lic 'printf %s "$QMS_ROUTER_API_KEY"' 2>/dev/null)"

printf 'env file: present=%s len=%s sha256=%s\n' "$([ -n "$env_file_value" ] && printf yes || printf no)" "${#env_file_value}" "$(printf %s "$env_file_value" | shasum -a 256 | awk '{print $1}')"
printf 'zshrc:    present=%s len=%s sha256=%s\n' "$([ -n "$zsh_value" ] && printf yes || printf no)" "${#zsh_value}" "$(printf %s "$zsh_value" | shasum -a 256 | awk '{print $1}')"
[ "$env_file_value" = "$zsh_value" ] && echo 'match=yes' || echo 'match=no'
```

Then prove which key the router accepts:

```bash
curl -i -H "Authorization: Bearer $zsh_value" http://127.0.0.1:8321/state
curl -i -H "Authorization: Bearer $env_file_value" http://127.0.0.1:8321/state
```

If a switch command fails:

- check that the router is healthy with `doctor`
- verify that the switch exists in `QMS_SWITCHES`
- check the service status with `service status`

## Development

Run tests:

```bash
go test ./...
```

Build:

```bash
make build
```
