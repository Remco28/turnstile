# Turnstile Architecture Overview

## System Components

### Core Services
- **CLI / daemon binary** (`cmd/turnstile/main.go`) – one executable that handles admin commands and runs the HTTP validator.
- **Store layer** (`internal/store/store.go`) – SQLite-backed persistence for users, projects, tokens, grants, and access logs.
- **HTTP API** (`internal/httpapi/server.go`) – exposes `/healthz` and `/v1/validate`.
- **Token utilities** (`internal/token/token.go`) – creates opaque high-entropy bearer tokens.

### Supporting Services
- **SQLite** – single source of truth for operator state and audit trail.

## Process Architecture

```text
Operator / Hermes / CLI user
            |
            v
      turnstile CLI  -----> SQLite
            |
            +---- run serve -----> HTTP validator -----> SQLite
                                          ^
                                          |
                                NoteSmith / future apps
```

Turnstile is intentionally one process with two modes:
- **CLI mode** for issuing/revoking/listing access
- **server mode** for validating tokens online

## Data Flow Examples

### Issue a token

```text
Frank/Hermes -> create-user/create-project if needed -> create-token
            -> store token row + token_projects grants
            -> print raw token once as JSON
```

### Validate access for an app

```text
App -> POST /v1/validate with token + project
    -> load token
    -> reject if missing/revoked/expired
    -> check grant in token_projects
    -> write access_log row
    -> return {authorized:true|false}
```

### Revoke access

```text
Operator -> revoke-token
         -> set revoked_at
         -> future validations fail immediately
```

## Key Abstractions

- **User** – named human or operator identity like `frank`, `james`, `lisa`.
- **Project** – first-class app/service slug like `notesmith`, `bag-app`, `tinyfish`.
- **Token** – bearer credential owned by one user.
- **Grant** – token-to-project mapping.
- **Access log** – append-only validation history for both successes and failures.

## Authentication & Authorization

- **Presented credential**: bearer token via `X-API-Key`, `Authorization: Bearer`, or JSON body.
- **Authorization decision**: token must be active and mapped to the requested project.
- **Trust boundary**: server binds to localhost/Tailscale-facing address by config; apps keep only the raw token secret.
- **Deliberate v1 tradeoff**: tokens are stored plaintext so Frank/agents can retrieve and re-send them when needed.

## Configuration

- `TURNSTILE_DB_PATH` – DB path, default `./turnstile.db`
- `TURNSTILE_LISTEN_ADDR` – server listen address, default `127.0.0.1:7432`

## Runtime & Operations Notes

- Auto-initialize schema on first run.
- Use WAL mode so CLI and server access play nicely.
- Keep the HTTP surface tiny: `/healthz`, `/v1/validate` only.
- Log auth outcomes to DB for operator visibility.
- Prefer additive schema evolution.

## Related Docs

- `README.md`
- `docs/ROADMAP.md`
- `project-manifest.md`
