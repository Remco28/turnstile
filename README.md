# Turnstile

Small self-hosted token issuer + validator for Frank's app ecosystem.

Turnstile exists to make access management conversational and centralized instead of scattered across many tiny app-specific admin pages.

## Goals

- issue bearer tokens for named people
- grant tokens access to one or more projects/apps
- validate tokens before expensive or privileged actions run
- revoke/reissue quickly
- keep the runtime tiny: one Go binary + one SQLite file

## Non-goals

- OAuth / OIDC
- full SSO
- consumer-style password recovery
- admin website in v1

## Current MVP

- CLI for user/project/token management
- HTTP `/v1/validate` endpoint
- SQLite-backed storage
- token revocation + expiry
- scoped grant replacement (`replace-grants`)
- project access inspection (`who-has-access`)
- access audit logging (`access-log`)

## Quick start

```bash
go run ./cmd/turnstile create-user frank
go run ./cmd/turnstile create-project notesmith --description "AI writing app"
go run ./cmd/turnstile create-token --user frank --project notesmith --label "frank local"
go run ./cmd/turnstile who-has-access --project notesmith
go run ./cmd/turnstile access-log --limit 20
go run ./cmd/turnstile serve
```

Then validate:

```bash
curl -s http://127.0.0.1:7432/v1/validate \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: tsk_live_...' \
  -d '{"project":"notesmith"}'
```

## Environment

- `TURNSTILE_DB_PATH` — SQLite database path, default `./turnstile.db`
- `TURNSTILE_LISTEN_ADDR` — HTTP listen address, default `127.0.0.1:7432`

## Docs

- Architecture: `docs/ARCHITECTURE.md`
- Roadmap: `docs/ROADMAP.md`
- Manifest: `project-manifest.md`
