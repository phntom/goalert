# goalert

Pikud HaOref (Israel) early-warning rocket-alert bot for Mattermost. It posts
the fastest available alert into chat: first the full trigger text with
`@mentions`/`#hashtags` (so members get an instant push notification), then
~200 ms later it patches the post down to a clean, localized card. Channels are
served in Hebrew, English, Russian and Arabic, chosen by the channel name.

## Sources

- **ynet** — primary fast path. Defaults to the AWS-direct origin
  `source-alerts.ynet.co.il`, which bypasses Akamai's cache so new alerts are
  not delayed by the CDN TTL. Polled ~4×/second.
- **oref history** — the authoritative Pikud HaOref feed; source of truth for
  end-of-alert (all-clear) and a dedup backstop.
- **Tzeva Adom WebSocket** — push-based, lowest latency, and the origin of
  pre-alerts (an early "expect alerts shortly" warning).

## Run

```sh
go run ./cmd/goalert
```

### Environment

| var | required | default | purpose |
|-----|----------|---------|---------|
| `CHAT_DOMAIN` | yes | — | Mattermost server URL |
| `AUTH_TOKEN` | yes | — | Mattermost bot token |
| `YNET_URL` | no | `source-alerts.ynet.co.il/...` | override the ynet endpoint |
| `OREF_HISTORY_URL` | no | oref `GetAlarmsHistory.aspx` | override the oref feed |
| `TZEVAADOM_WS_URL` | no | `wss://ws.tzevaadom.co.il/socket` | override the WS endpoint |
| `METRICS_ADDR` | no | `:3000` | Prometheus `/metrics` listen address |
| `DISABLE_YNET` / `DISABLE_OREF` / `DISABLE_TZEVAADOM` | no | — | set to `1` to disable a source |

The bot posts to every channel it is a member of (except `town-square` and
`off-topic`).

## Area data

City/area data is embedded in `internal/area/data.gen.json` and reconciles oref
`GetCitiesMix`, the oref per-language city lists, and Tzeva Adom `cities.json`.
Regenerate it from the live sources with:

```sh
go run ./internal/area/cmd/genareas
```

## Test

```sh
go test ./...
```
