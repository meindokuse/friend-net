# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Workspace Layout

Go workspace (`go.work`) monorepo with three modules:

| Module | Path | Purpose |
|--------|------|---------|
| `github.com/meindokuse/cloud-drive/auth-service-new` | `auth-service/` | JWT auth, OAuth (Google), sessions — Gin, PostgreSQL, Redis, Kafka producer |
| `github.com/meindokuse/cloud-drive/user-service-new` | `user-service/` | User profiles — chi, MongoDB, Kafka consumer |
| `github.com/meindokuse/cloud-drive/common` | `common/` | Shared event types (e.g. `AccountCreated`) consumed by both services |

Each service has its own `CLAUDE.md` with detailed per-service architecture.

## Commands

```bash
# Create the external Docker network (run once)
make network

# Start full stack: Traefik + Kafka infra + auth-service + user-service
make up

# Stop all services
make down

# Rebuild and restart Go services after code changes
make build

# Tail logs from all services
make logs

# Show running container status
make ps

# Run a single service locally (from inside the service directory)
go run ./cmd

# Run tests for a specific package
go test ./internal/application/service/user/create/...

# Lint (run from inside a service directory)
golangci-lint run ./...
```

Infrastructure (`infra/docker-compose.yaml`) runs Kafka (KRaft mode, no Zookeeper) and Kafka UI at `kafka.localhost`.

## Overall Architecture

```
Browser / API client
        │
        ▼
   Traefik (port 80)              ← traefikV2/ or traefikV3/
   ├── /auth/*  ──────────────► auth-service  (Gin, :8080)
   │               jwt-auth middleware on private routes
   └── /users/* ──────────────► user-service  (chi, :8081)
                   jwt-auth + rate-limit middleware on private routes

auth-service ──Kafka (accounts.events)──► user-service consumer
                  (AccountCreated event)
```

All containers share the `traefik-net` Docker network. Services are not exposed directly; Traefik routes by `Host(localhost) && PathPrefix(...)` labels.

### Cross-service event flow

1. `auth-service` writes an `outbox_events` row in the same Postgres transaction as account creation.
2. The flusher goroutine polls the outbox (every 5 s, batch 100) and publishes to Kafka topic `accounts.events`.
3. `user-service` consumes `accounts.events` (group `user-service`) and creates the user document idempotently (`user.id = account_id`).
4. The `common` module defines the shared `AccountCreated` struct used on both ends.

### Shared architectural patterns

Both services follow the same four-layer Clean Architecture:

- **Presentation** (`internal/app/`) — HTTP handlers only; no business logic
- **Application** (`internal/application/service/`) — use cases; defines its own repository interfaces locally per use-case
- **Domain** (`internal/domain/`) — pure Go structs, no DB/JSON tags, no external imports beyond `uuid`/`time`
- **Infrastructure** (`internal/infrastructure/`) — implements application interfaces; owns all DB/Kafka interaction

Both use the same initialization chain: `internal/init.go` wires dependencies, `internal/app.go` holds the `App` struct, `cmd/main.go` calls `internal.New(ctx).Run(ctx)`.

### Traefik JWT middleware

The `jwt-auth` middleware is defined in `traefikV2/dynamic/middlewares.yml` (or `traefikV3/`). It validates Bearer tokens before requests reach `user-service` private routes and `auth-service` private routes. The token is issued by `auth-service`; downstream services receive the forwarded `X-User-ID` header.

## Configuration

Both services load config from YAML (`CONFIG_PATH` env var, defaults to `config/config.yaml`) with env var overrides. Copy `.env.example` to `.env` before running locally.

Key required secrets for `auth-service`: `JWT_SECRET`, `JWT_REFRESH_SECRET`, `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`.
