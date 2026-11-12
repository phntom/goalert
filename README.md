# goalert

Real-time Israel Home Front Command (Pikud HaOref) alert bot for Mattermost.

## What it does

goalert polls the official Home Front Command alert feeds and posts prioritized alerts to Mattermost:

- Posts alerts to Mattermost with per-city @-mentions and hashtags.
- Renders each alert per language: Hebrew, English, Russian and Arabic (locales in `internal/config/locale.*.yaml`).
- Marks posts as urgent (important for lockdown and hazardous-materials alerts) and requests acknowledgement for alerts with a shelter time of 60 seconds or more.
- Splits alerts covering more than 20 cities into multiple posts.
- Patches existing posts as alerts evolve, instead of duplicating them.
- Forwards important IDF announcements from the IDF Telegram channel.
- Exposes Prometheus metrics and health endpoints.

## Sources

- Home Front Command alerts JSON from `oref.org.il` (`internal/sources/oref.go`)
- ynet alert feed (`internal/sources/ynet.go`)
- Official Pikud HaOref and IDF Telegram channels via TDLib (`internal/sources/telegram.go`)

Each source can be disabled at startup with `DISABLE_OREF`, `DISABLE_YNET` or `DISABLE_TELEGRAM`.

## Configuration

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `CHAT_DOMAIN` | yes | — | Mattermost base URL |
| `AUTH_TOKEN` | yes | — | Mattermost bot access token |
| `APP_ID` | for Telegram | — | Telegram API id (read by gotd) |
| `APP_HASH` | for Telegram | — | Telegram API hash (read by gotd) |
| `LOG_LEVEL` | no | `info` | `debug`, `info`, `warn` or `error` |
| `CONFIG_TEAM` | no | `phantom` | Mattermost team holding the config channel |
| `CONFIG_CHANNEL` | no | `config` | Mattermost channel storing the Telegram session |
| `DISABLE_YNET` | no | unset | `1` disables the ynet source |
| `DISABLE_OREF` | no | unset | `1` disables the oref source |
| `DISABLE_TELEGRAM` | no | unset | `1` disables the Telegram source |

The process exits with status 8 if `CHAT_DOMAIN` or `AUTH_TOKEN` is empty, and with status 7 if the Telegram client cannot start.

## Mattermost setup

1. Create a bot account with a personal access token and set `AUTH_TOKEN`.
2. Invite the bot to every channel it should post to; it posts to all channels it is a member of (except `off-topic` and `town-square`).
3. Create a channel named `config` in the `CONFIG_TEAM` team and invite the bot; goalert stores its Telegram session there as posts.
4. Each channel's language is picked from its display name: Hebrew, Arabic or Cyrillic script selects that language, anything else is English (`ChannelToLanguage` in `internal/bot/render.go`).

## Running

From source:

```
make run        # needs the environment variables above
make build      # writes ./bin/alert
```

Docker:

```
make docker-build   # builds phntom/goalert:dev
docker run --rm \
  -e CHAT_DOMAIN=https://mattermost.example.com \
  -e AUTH_TOKEN=<token> \
  -e APP_ID=<id> \
  -e APP_HASH=<hash> \
  phntom/goalert:dev
```

The image is distroless: no shell, no package manager, runs as a non-root user.

Helm: the chart in `charts/goalert` deploys the bot with liveness and readiness probes wired to the health endpoints. Main values keys: `config.*` (Mattermost and Telegram settings; set `config.existingSecret` to use your own secret instead of the generated one), `image`, `ingress`, `livenessProbe`, `readinessProbe`.

## Monitoring

Port 3000 serves:

- `/metrics` — Prometheus metrics (fetch success/failure counters, response times, post and patch counters, city and region histograms)
- `/healthz` — liveness: 200 while the process runs
- `/readyz` — readiness: 200 once the Mattermost login succeeded, 503 before

Logs are one JSON object per line on stderr.

## Development

```
make check   # format check + lint + vet + test + build
make test    # go test -race ./...
```

The test suite runs with the race detector. CI (`.github/workflows/ci.yml`) runs lint, vet, tests and `helm lint` on every push and pull request.

`internal/district/update-districts.py` refreshes the embedded district tables from the Home Front Command site; it needs `curl` and rewrites the four `internal/district/districts.*.json` files.

## License

Apache License 2.0 — see [LICENSE](LICENSE).
