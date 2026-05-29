# Local Router

A planned Go tool for giving temporary local web artifacts stable, friendly browser URLs such as `http://dev.localhost`, while the actual artifact server remains ephemeral and keeps stdout/lifecycle attached to the invoking CLI.

This repository is currently in specification mode. See [`docs/spec.md`](docs/spec.md).

## Core idea

- Run one small persistent loopback-only router/reverse proxy.
- Let ad hoc artifact servers register and unregister routes dynamically.
- Use `.localhost` hostnames by default to avoid Windows hosts-file edits.
- Provide a live dashboard at `http://art.localhost`.

## Status

No implementation yet; requirements and architecture are being captured first.
