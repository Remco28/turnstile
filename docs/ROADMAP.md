# Turnstile Roadmap

## MVP

- [x] clarify product boundary: token issuer + validator, not full auth platform
- [x] define first-class projects/apps and scoped grants
- [ ] build Go CLI for users, projects, tokens
- [ ] build `/v1/validate` HTTP endpoint
- [ ] store users/projects/tokens/grants/audit in SQLite
- [ ] support revocation and expiry
- [ ] document local/dev/prod setup

## Next

- [ ] richer list/filter commands
- [ ] show last-used info and recent failures
- [ ] token rotation helpers
- [ ] safer reveal/fingerprint UX
- [ ] import/export / backup helpers

## Explicitly deferred

- [ ] admin website
- [ ] password login system
- [ ] OAuth / OIDC
- [ ] multi-tenant enterprise features
