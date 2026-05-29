# Initial Specification Plan

## Context

The user provided an AI chat about a Go tool for local ephemeral web artifacts with stable `.localhost` URLs. The request is to set up a project and extract requirements before implementation.

## Initial TODO Checkpoints

- [ ] Review extracted specification with the user.
- [ ] Decide project/tool name and command shape.
- [ ] Decide WSL-side-only v1 vs early Windows-side router support.
- [ ] Decide port 80 fallback behavior.
- [ ] After approval, create implementation plan before coding.

## Current recommendation

Start with a WSL-side Go CLI/router implementation and keep the architecture portable. Use `.localhost` dynamic names by default, a dashboard at `art.localhost`, and an in-memory registration API with heartbeat cleanup.
