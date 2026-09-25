# Engo

Engo is an IRC bot implemented in Go with [Tengo](https://github.com/d5/tengo) as its embedded scripting language.

M0 establishes the project foundation: a buildable Go executable, minimal IRC registration/PING-PONG handling, Tengo execution, tests, CI and an Alpine-based OCI image.

## Status

**M0 — foundation**

Implemented:

- Go module and `cmd/engo` executable
- minimal IRC client with TLS support
- `NICK` / `USER` registration
- IRC `PING` / `PONG`
- embedded Tengo runtime
- example Tengo script
- unit test for script execution
- GitHub Actions CI
- Alpine OCI build
- MIT license

The richer bot/event API is intentionally deferred to later milestones.

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
| `ENGO_SCRIPT` | `scripts/examples/hello.tengo` | Tengo script run at startup |

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
