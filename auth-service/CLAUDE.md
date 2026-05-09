# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Start local infrastructure
docker-compose up -d postgres redis

# Run service locally
go run ./cmd

# Build binary
CGO_ENABLED=0 GOOS=linux go build -o auth-service ./cmd

# Build Docker image
docker build -t auth-service:latest .

# Run full stack (service + postgres + redis) via Docker
docker-compose up auth-service-new

# Lint
golangci-lint run ./...

# Tests (none exist yet — see MVP TODO below)
go test ./...
```

Configuration is loaded from a YAML file (`CONFIG_PATH` env var) or from environment variables directly. Copy `.env.example` to `.env` and fill in secrets before running.

## Architecture

This is a **Go 1.23 authentication microservice** using Gin, PostgreSQL (pgx/v5), Redis (go-redis/v9), and optional Kafka (sarama). It follows **Clean Architecture** with four layers:

```
cmd/main.go                  → entrypoint, calls internal.New(ctx).Run(ctx)
internal/app.go              → App struct: wires all dependencies, starts HTTP + flusher
internal/
  app/                       → PRESENTATION  (HTTP handlers)
  application/service/       → USE CASES     (business logic)
  domain/entity/             → DOMAIN        (pure structs, no db/json tags)
  infrastructure/            → INFRASTRUCTURE (Postgres, Redis, Kafka, OAuth)
  pkg/                       → Utilities: jwt, pass, terror, closer, connector/*
config/                      → Config struct (cleanenv), config.go + components.go
migrations/                  → SQL migration files (mounted by docker-compose postgres)
```

Dependencies flow inward only: infrastructure implements interfaces defined by application services.

### Application layer — use case registries

`internal/application/service/auth/registry.go` aggregates 7 auth use cases:
- `login` — authenticates, enforces max-3-sessions limit (evicts oldest), issues token pair
- `register` — creates account + outbox event in a single Postgres transaction
- `refresh` — rotates token pair with reuse-attack detection (30s grace period)
- `logout` / `revoke_session` — revokes session, blacklists access JTI in Redis
- `introspect` — validates token, checks blacklist, returns claims
- `get_sessions` — lists active sessions for authenticated user

`internal/application/service/oauth/registry.go` aggregates 4 OAuth use cases: `login`, `link`, `unlink`, `get_linked`.

Each use case defines its own narrow interfaces (e.g., `AccountProvider`, `SessionManager`), which infrastructure implements.

### Infrastructure layer

`internal/infrastructure/storage/registry.go` — aggregates all repositories:
- `account/` — PostgreSQL CRUD; `CreateWithOutbox` wraps in a transaction
- `session/` — Redis: stores sessions and refresh token pairs (HMAC-SHA256 hashed)
- `oauth/` — PostgreSQL OAuth account links
- `outbox/` — PostgreSQL outbox table polled by the flusher

`internal/infrastructure/flusher/flusher.go` — background goroutine that polls `outbox_events` (where `processed_at IS NULL`), publishes to Kafka topic `accounts.events`, marks events processed. Controlled by `OutboxConfig` (flush interval 5s, batch 100).

### JWT / token design

- **Access token**: signed HS256 JWT; claims include `session_id`, `sub` (user UUID), `jti`
- **Refresh token**: opaque string `"<sessionID>.<random32>"` — `sessionID` extracted on parse; `random32` is HMAC-SHA256-hashed and stored in Redis alongside the previous hash
- **Fingerprint**: HMAC-SHA256 hash of a client-supplied fingerprint stored in the session; validated on each refresh
- Token rotation: on refresh, `previous = current, current = new`; if the presented hash matches neither, the session is revoked (reuse attack)
- Refresh token readable from httpOnly cookie, `X-Refresh-Token` header, or request body

### Error handling

`internal/pkg/terror/errors.go` defines typed errors (NotFound, Conflict, Unauthorized, BadRequest, Internal, Forbidden). HTTP handlers call the `terror.Is*()` helpers to map to HTTP status codes.

### Auth middleware

Defined inline in `internal/app.go` (around line 301): extracts Bearer token, calls `introspect` service, sets `account_id` and `session_id` into Gin context.

## HTTP Routes

| Method | Path | Auth required |
|--------|------|---------------|
| POST | `/auth/register` | No |
| POST | `/auth/login` | No |
| POST | `/auth/refresh` | No |
| POST | `/auth/logout` | Yes |
| POST | `/auth/logout-all` | Yes |
| GET | `/auth/sessions` | Yes |
| DELETE | `/auth/sessions/:session_id` | Yes |
| POST | `/auth/introspect` | No |
| GET | `/auth/google` | No |
| GET | `/auth/google/callback` | No |
| GET | `/auth/link/google` | Yes |
| GET | `/auth/link/google/callback` | Yes |
| GET | `/auth/linked` | Yes |
| DELETE | `/auth/linked/:provider` | Yes |
| GET | `/healthz` | No |

## Known MVP TODOs (from README)

- No tests yet — integration tests for auth flows and smoke tests against real Postgres + Redis are planned
- Rate limiting on `/auth/login` and `/auth/refresh`
- Auth middleware not applied to all OAuth-protected endpoints (expects `ctx.Get("account_id")`)
- Migrate from docker-compose init scripts to a proper migration runner (avoid mounting `.down.sql` at startup)
- `oauth_accounts.expiry` uses mixed unix/timestamp formats — needs clarification
- Audit logs for security events
