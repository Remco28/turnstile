# Turnstile Project Manifest

## 1. Core Identity

- **Architecture:** `docs/ARCHITECTURE.md`
- **Roadmap / goals:** `docs/ROADMAP.md`
- **Primary product summary:** `README.md`

## 2. Dynamic State

- **Activity log:** `comms/log.md`
- **Message board:** `comms/board.md`
- **Task specs:** `comms/tasks/`

## 3. Code & Config Entrypoints

- **Main binary entrypoint:** `cmd/turnstile/main.go`
- **Storage layer:** `internal/store/store.go`
- **HTTP API:** `internal/httpapi/server.go`
- **Token generation / parsing:** `internal/token/token.go`
- **Configuration:** `internal/config/config.go`
- **Dependencies:** `go.mod`
- **Database file:** `turnstile.db` (default local dev path)

## 4. Working Rules

- Keep Turnstile narrow: token issuance + validation, not a full identity platform.
- Prefer stdlib and tiny dependencies.
- JSON output for machine-facing CLI commands.
- Tokens are scoped to projects/apps; avoid global skeleton keys by default.
- Log authorization decisions so revocation/debugging stays easy.
