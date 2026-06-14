# backend — Go service + MySQL

Holds the Claude API key, assembles Coach Memory into prompts, calls Claude, runs the
deterministic safety layer, and persists all data. The Android client talks **only** to this
service; it never calls Anthropic directly.

## Layout

- `cmd/server/` — entrypoint and dependency wiring only; no business logic.
- `internal/platform/` — cross-cutting infra: `config`, `db`, `httpx`, `logging`.
- `internal/<domain>/` — one package per domain: `auth`, `memory`, `onboarding`, `readiness`,
  `injury`, `location`, `diet`, `coaching`, `safety`. (Domain → epic mapping in [`/CLAUDE.md`](../CLAUDE.md).)
- `migrations/` — ordered, versioned SQL. Never edit a shipped migration; add a new one.
- `api/` — OpenAPI contract; the source of truth for the client ↔ server interface.

## Conventions

Standard Go layout, consumer-defined interfaces, `context.Context` first arg on I/O, errors
wrapped with `%w`, `gofmt`/`go vet` clean, table-driven tests. The `safety` package and
`readiness` math are TDD with full coverage. Secrets come from env/config at runtime only
(see [`/.env.example`](../.env.example)).

## Local dev (once Wave 0 lands)

```
cp ../.env.example .env      # fill in real values; .env is git-ignored
go run ./cmd/server          # starts HTTP server on HTTP_ADDR
go test ./...                # run the suite
```
