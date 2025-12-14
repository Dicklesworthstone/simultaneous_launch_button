# Simultaneous Launch Button (slb)

A cross-platform CLI that implements a “two-person rule” for running potentially destructive commands from AI coding agents.

When an agent wants to run something risky (e.g. `rm -rf`, `git push --force`, `kubectl delete`, `DROP TABLE`), `slb` is designed to require peer review and explicit approval before execution.

Status: under active development. The current implementation is early scaffolding; the authoritative design is in `PLAN_TO_MAKE_SLB.md`.

## Why this exists

Coding agents can get tunnel vision, hallucinate, or misunderstand context. A second reviewer (ideally with a different model/tooling) catches mistakes before they become irreversible.

`slb` is built for multi-agent workflows where many agent terminals run in parallel and a single bad command could destroy work, data, or infrastructure.

## Core ideas (v2 plan)

- **Client-side execution**: the daemon (when used) is a notary/verifier; commands execute in the requesting process so the correct shell env is inherited.
- **Command hash binding**: approvals bind to an exact `CommandSpec` (raw + argv + cwd + shell) via sha256.
- **Risk tiers**: `CRITICAL` (2+ approvals), `DANGEROUS` (1 approval), `CAUTION` (auto after delay) with configurable patterns.
- **SQLite source of truth**: project state lives in `.slb/state.db`; JSON files are materialized snapshots for watching/interop.
- **Integrations**: MCP Agent Mail for coordination (notify reviewers, thread audits), plus hooks/rules for agent tools.

## Repository layout (current)

This repo is organized in a Go “internal/…” layout with placeholders for the planned architecture:

- `cmd/slb/`: CLI entrypoint
- `internal/cli/`: Cobra root + quick reference card (work in progress)
- `internal/core/`: domain types (early)
- `internal/db/`: SQLite connection wrapper (early)
- `internal/config/`: config structs + loader (early)
- `internal/daemon/`: daemon/notary (stub)
- `internal/tui/`: Bubble Tea TUI (stub)
- `internal/utils/`: hashing + structured logging

## Quickstart (dev)

```bash
# Run from source
go run ./cmd/slb

# Build a local binary
make build
./build/slb version
```

## Install (binary release)

```bash
curl -fsSL https://raw.githubusercontent.com/Dicklesworthstone/slb/main/scripts/install.sh | bash
```

Optional:

```bash
# Install somewhere else (default: /usr/local/bin)
INSTALL_DIR="$HOME/.local/bin" curl -fsSL https://raw.githubusercontent.com/Dicklesworthstone/slb/main/scripts/install.sh | bash
```

## Shell completions

```bash
# zsh (~/.zshrc)
eval "$(slb completion zsh)"

# bash (~/.bashrc)
eval "$(slb completion bash)"

# fish (~/.config/fish/config.fish)
slb completion fish | source
```

## Planning & task tracking

- Design doc: `PLAN_TO_MAKE_SLB.md`
- Agent rules: `AGENTS.md`
- Task queue (Beads): `bd ready --json`
- Prioritization helper: `bv --robot-priority`

## Safety note

`slb` is meant to add friction and peer review for dangerous actions, not replace real access controls. Use least-privilege credentials and environment safeguards as your first line of defense.
