# PBMP in Engo

Engo implements the required PBMP/1 discovery/status surface over a local Unix domain socket. Set `ENGO_PBMP_SOCKET` to enable it; the recommended path is `/tmp/engo.pbmp.sock`.

The socket is mode 0600 and no PBMP TCP listener is provided. Engo remains fully standalone when PBMP is disabled.

Implemented methods: `pbmp.info`, `capabilities.list`, `bot.info`, and `networks.list`.

This is intentionally the same implementation-neutral surface used by LuCa so BotWeb does not need Engo-specific code.
