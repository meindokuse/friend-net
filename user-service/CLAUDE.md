# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Workspace Layout

This is a Go workspace (`go.work`) mono-repo at `D:\cloud-drive\` with three modules:

- `auth-service/` — authentication service (JWT, OAuth, PostgreSQL, Redis, Kafka)
- `user-service/` — user management service (this module)
- `common/` — shared event types consumed by both services

Module name: `github.com/meindokuse/cloud-drive/user-service-new`

## Build & Run

```bash
# Run locally (from user-service/)
go run ./cmd/main.go

# Build binary
go build -o user-service-new ./cmd/main.go

# Run with Docker Compose (mongo + service)
docker-compose up --build

# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/application/service/user/create_user/...
```

## Configuration

Config is loaded from `config/config.yaml` by default. Override with `CONFIG_PATH` env var, or set individual env vars (env vars take precedence). Copy `.env.example` to `.env` for local development.

Key env vars:
- `HTTP_ADDR` (default `:8081`)
- `MONGO_URI`, `MONGO_DATABASE`, `MONGO_TIMEOUT`
- `KAFKA_BROKERS`, `KAFKA_TOPIC`, `KAFKA_GROUP_ID`, `KAFKA_ENABLED`
- `APP_ENV`, `LOG_LEVEL`

## Architecture: Clean Architecture with 4 Layers

Dependencies point strictly inward. Inner layers never import outer layers.

```
Presentation (internal/app/)
    → Application (internal/application/service/)
        → Domain (internal/domain/)
            ← Infrastructure implements application interfaces
```

**Layer rules:**
- **Domain** (`internal/domain/`): pure Go structs, zero external dependencies. Only `uuid`, `time` allowed. No db/json/proto tags.
- **Application** (`internal/application/service/`): business logic and use cases. Defines its own repository interfaces locally per use-case. No HTTP, Mongo, or Kafka imports.
- **Presentation** (`internal/app/`): HTTP handlers (chi router). Converts HTTP ↔ DTOs, calls application services, no business logic.
- **Infrastructure** (`internal/infrastructure/`): implements application interfaces. Contains MongoDB storage (with DAO pattern), Kafka consumer.

## Key Patterns

**Use case structure** — each use case lives in its own package with local interface definitions:
```
internal/application/service/user/{use_case}/service.go
  type Repository interface { ... }   // defined here, not shared
  type Service struct { repo Repository }
  func NewService(repo Repository) *Service
  func (s *Service) Execute(ctx, Input) (Output, error)
```

**Registry pattern** — each layer aggregates components into a Registry:
- `storage.Registry` → `{User, Idempotency}`
- `userservice.Registry` → `{CreateUser, GetUser, UpdateProfile, ...}`
- `messagebus.Registry` → `{Consumer}`

**Initialization chain** (`internal/init.go`):
```
initMongo → initStorages → initServices → initMessageBus → initHTTPServer
```

**Error mapping** (`internal/app/user/v1/helpers.go`): `writeUsecaseError()` maps domain errors to HTTP status codes using `errors.Is()`. Domain errors are defined in `internal/domain/entity/user.go` and `internal/pkg/apperr/`.

**Optimistic locking**: User entity has a `version` field. `Update` increments it; conflicts return `ErrVersionConflict` (HTTP 409).

**Soft deletes**: Users are not removed from MongoDB; `deletedAt` is set instead. `ErrAlreadyDeleted` returns HTTP 410.

## Domain Invariants

- `Email` or `Phone` required at creation
- `DisplayName` required, max 64 chars; `Bio` max 500 chars
- `Username`: 3–32 chars, alphanumeric + underscore, normalized to lowercase
- `Email`: RFC5321, max 254 chars, normalized to lowercase
- `Phone`: E.164 format (`+[1-9]\d{1,14}`)
- Privacy levels: `everyone | friends | nobody`

## Kafka Consumer

Consumes `accounts.events` topic (group `user-service`). On `AccountCreated` event, creates a user idempotently (`user.id = account_id`). Uses 16 virtual-partition workers (hash-routed by AccountID for ordering). Idempotency is tracked in MongoDB `processed_events` collection (TTL 7 days).

## HTTP API

All `/users/me*` routes require `X-User-ID` header (UUID). Authentication middleware is set in `internal/app/user/v1/service.go`.

| Method | Path | Handler file |
|--------|------|-------------|
| POST | `/users` | `create_user.go` |
| GET | `/users/me` | `get_user.go` |
| GET | `/users/{id}` | `get_user.go` |
| GET | `/users/username/{username}` | `get_user.go` |
| POST | `/users/batch` | `get_user.go` |
| PATCH | `/users/me/profile` | `update_profile.go` |
| PATCH | `/users/me/settings` | `update_settings.go` |
| PATCH | `/users/me/email` | `change_email.go` |
| PATCH | `/users/me/phone` | `change_phone.go` |
| DELETE | `/users/me` | `delete.go` |
| POST | `/users/me/last-seen` | `last_seen.go` |
| GET | `/users/search` | `search.go` |
| GET | `/users/me/list` | `list.go` |

## Pending Work (from migrate.md)

- `application/service/user/registry.go` — not yet assembled
- `infrastructure/storage/registry.go` — not yet extracted
- Unit/integration tests — not yet written
- External HTTP API backward-compatibility verification pending
