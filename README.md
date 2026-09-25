# Engo

Engo is an IRC bot implemented in Go with [Tengo](https://github.com/d5/tengo) as its embedded scripting language.

M0–M4 established the project foundation, robust IRC lifecycle, Tengo event/command API, transactional script lifecycle, persistence, timers and constrained HTTP access. M5 adds IRCv3 capability negotiation plus capability- and account-based authorization. PBMP integration adds an optional local management interface shared with LuCa and BotWeb.

## Status

**M5 — IRCv3 and permissions (implementation in progress; CI qualification pending)**

Implemented:

- Go module and `cmd/engo` executable
- IRC client with TLS support and TLS 1.2 minimum
- `NICK` / `USER` registration and IRCv3 CAP negotiation
- optional SASL PLAIN authentication
- IRC `PING` / `PONG`
- exponential reconnect/backoff
- graceful SIGINT/SIGTERM shutdown
- configuration validation
- embedded Tengo runtime
- example Tengo script
- unit test for script execution
- GitHub Actions CI
- Alpine OCI build
- MIT license
- transactional multi-script hot reload with allocation limits
- per-script persistent KV storage and timers
- constrained HTTPS client with hostname allowlist and SSRF defenses
- per-script capabilities (including explicit HTTP grants)
- IRCv3 `account-tag`, `account-notify`, `extended-join` and `server-time` negotiation
- account identity provenance tracking and fail-closed command permissions
- optional standalone architecture; the future Engo Web control plane is not required to run Engo

CI includes normal tests, race-detector tests, vet, Go build and Alpine container build. A current green GitHub Actions run has not yet been independently verified.

## Run the Tengo proof of concept

```sh
go run ./cmd/engo
```

Without `ENGO_SERVER`, Engo executes the configured Tengo script and exits.

## Connect to IRC

```sh
ENGO_SERVER=irc.example.net:6697 \
ENGO_NICK=engo \
go run ./cmd/engo
```

Configuration variables:

| Variable | Default | Description |
| --- | --- | --- |
| `ENGO_SERVER` | empty | IRC server as `host:port`; empty means no IRC connection |
| `ENGO_NICK` | `engo` | IRC nickname |
| `ENGO_USER` | `engo` | IRC username |
| `ENGO_REALNAME` | `Engo IRC bot` | IRC real name |
| `ENGO_TLS` | `1` | TLS enabled unless set to `0` |
| `ENGO_SCRIPT` | `scripts/examples/hello.tengo` | Single Tengo script; used when `ENGO_SCRIPTS_DIR` is unset |
| `ENGO_SCRIPTS_DIR` | empty | Directory of `.tengo` scripts loaded transactionally |
| `ENGO_SCRIPT_MAX_ALLOCS` | `100000` | Maximum Tengo VM allocations per registration/event execution |
| `ENGO_SASL_USERNAME` | empty | SASL PLAIN authentication identity; requires password |
| `ENGO_SASL_PASSWORD` | empty | SASL PLAIN password; requires username |
| `ENGO_RECONNECT_MIN` | `2s` | Initial reconnect delay |
| `ENGO_RECONNECT_MAX` | `2m` | Maximum reconnect delay |
| `ENGO_PBMP_SOCKET` | empty | Optional local PBMP/1 Unix socket; e.g. `/tmp/engo.pbmp.sock` |\n| `ENGO_CHANNELS` | empty | Comma-separated configured channels exposed through PBMP `channels.list` |

## OCI

```sh
docker build -t engo:dev .
docker run --rm engo:dev
```

Alpine Linux is the default OCI base where practical.

## Project direction

Go owns protocol correctness, networking, TLS, lifecycle, security boundaries and resource management. Tengo scripts will define commands and event-driven bot behaviour through a controlled capability-based API.

See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for the architecture and milestone roadmap.

## License

MIT. See [`LICENSE`](LICENSE).
