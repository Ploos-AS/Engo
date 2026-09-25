# Engo architecture

Engo is an IRC bot implemented in Go with Tengo as its embedded scripting language.

## Design principles

- Go owns network I/O, TLS, IRC protocol handling, lifecycle, configuration, logging, permissions and resource limits.
- Tengo owns bot behaviour: commands, event handlers, timers and user-extensible logic.
- The scripting API must be capability-oriented. Scripts should receive only the functionality they need.
- IRC protocol state must remain valid even when a script fails.
- Runtime reloads must not require disconnecting from IRC once hot reload is implemented.

## M0 layout

- `cmd/engo`: executable entry point.
- `internal/config`: environment configuration.
- `internal/irc`: minimal IRC transport and registration.
- `internal/script`: Tengo runtime integration.
- `scripts/examples`: executable Tengo examples.

## Planned scripting API

The public Tengo API will evolve around controlled objects such as `bot` and event/context values. Planned capabilities include:

- `bot.command(name, handler)`
- `bot.on(event, handler)`
- `bot.say(target, text)`
- `bot.notice(target, text)`
- `bot.action(target, text)`
- timers/scheduler
- scoped HTTP client
- scoped persistent key/value state
- structured logging

Raw IRC writes, filesystem access and network access outside declared capabilities should not be exposed by default.

## Milestones

- M0: repository foundation, Go executable, minimal IRC wire client, Tengo proof of concept, tests and CI.
- M1: robust IRC connection lifecycle, reconnect/backoff, TLS/SASL and configuration validation. **Implemented.**
- M2: Tengo event and command API. **Implemented (initial API).**
- M3: script lifecycle, isolation and hot reload.
- M4: persistence, timers and capability-scoped HTTP.
- M5: IRCv3 and permissions/ACL model.
- M6: production OCI/release pipeline and operational documentation.


## M2 Tengo API

M2 uses a deliberately small capability surface. Scripts declare handlers with `bot("on", event, id)` or `bot("command", command, id)`. During isolated event execution, `bot("active", id)` selects the active handler and the `event` map exposes `name`, `nick`, `target`, `text`, `command` and `args`.

Output capabilities are `bot("say", target, text)`, `bot("notice", target, text)` and `bot("action", target, text)`. Raw IRC writes remain unavailable to scripts.
