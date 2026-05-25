# Troubleshooting

## Validate

Run automated checks:

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

## Doctor Fails

- make sure the service is started
- check `QMS_LISTEN_ADDR`
- try running `serve --env ~/.codex-quick-model-switch.env` manually

## Codex Provider Unauthorized

- keep `env_key = "QMS_ROUTER_API_KEY"` in `~/.codex/config.toml`
- do not add a nested `[model_providers."codex-quick-model-switch".auth]` command block
- make sure the router key visible to Codex is the same value as `~/.codex-quick-model-switch.env`
- restart Codex after changing environment or config

Validate without printing secrets:

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

## Switch Command Fails

- check that the router is healthy with `doctor`
- verify that the switch exists in `QMS_SWITCHES`
- check the service status with `service status`
