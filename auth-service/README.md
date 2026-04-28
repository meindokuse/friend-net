# Auth Service - Полная документация

## Оглавление

1. [Обзор](#обзор)
2. [Архитектура](#архитектура)
3. [Быстрый старт](#быстрый-старт)
4. [Конфигурация](#конфигурация)
5. [HTTP API](#http-api)
6. [OAuth Flow](#oauth-flow)
7. [Базы данных](#базы-данных)
8. [Outbox Pattern](#outbox-pattern)
9. [Тестирование](#тестирование)
10. [Troubleshooting](#troubleshooting)

---

## Обзор

`auth-service` — микросервис аутентификации и авторизации на Go (Gin), построенный в стиле Clean Architecture.

### Основные возможности

- ✅ Регистрация и логин пользователей
- ✅ JWT токены (access + refresh) с rotation
- ✅ Управление сессиями устройств
- ✅ OAuth вход (Google, GitHub, VK)
- ✅ Introspection токенов для других сервисов
- ✅ Outbox Pattern для синхронизации с user-service
- ✅ Replay attack protection
- ✅ Device fingerprinting

### Технологический стек

- **Go 1.25+**
- **PostgreSQL** (pgx/v5) - основное хранилище
- **Redis** - сессии и blacklist
- **Kafka** (опционально) - события через Outbox Pattern
- **Docker** - контейнеризация

---

## Архитектура

### Слои проекта

```
auth-service/
├── cmd/server/              # Entry point
├── internal/
│   ├── controllers/http/    # HTTP handlers
│   ├── usecase/            # Бизнес-логика
│   │   ├── auth/           # Регистрация, логин, refresh
│   │   └── oauth/          # OAuth flow
│   ├── domain/             # Доменные модели
│   │   ├── account/        # Account, OAuthAccount
│   │   └── session/        # Session, RefreshPair
│   ├── adapters/           # Репозитории
│   │   ├── postgresql/     # PostgreSQL (pgx)
│   │   └── redis/          # Redis
│   ├── infra/              # OAuth провайдеры
│   └── pkg/                # Shared utilities
├── migrations/             # SQL миграции
└── pkg/                    # Shared packages
    ├── jwt/                # JWT manager
    ├── pass/               # Password hasher
    ├── logger/             # Structured logging
    └── postgresql/         # Connection pool
```

### Принцип взаимодействия

```
HTTP Request → Controller → UseCase → Repository → Database
                    ↓
                 Domain Models
```

---

## Быстрый старт

### Требования

- Docker + Docker Compose
- или локально: Go 1.25+, PostgreSQL, Redis

### Запуск в Docker

```bash
cd auth-service
docker compose up --build
```

Что стартует:
- `postgres` (5432)
- `redis` (6379)
- `migrate` (накатывает миграции)
- `auth-service` (8080)

### Проверка

```bash
curl http://localhost:8080/healthz
# {"status":"ok"}
```

### Локальный запуск

```bash
# 1. Запустить PostgreSQL и Redis
docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:latest
docker run -d -p 6379:6379 redis:latest

# 2. Применить миграции
make migrate-up

# 3. Запустить сервис
go run cmd/server/main.go
```

---

## Конфигурация

### Основной файл: `config.yaml`

```yaml
server:
  httpAddr: ":8080"

postgres:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  database: auth_db
  sslMode: disable
  maxConns: 25
  minConns: 5

redis:
  addr: localhost:6379
  db: 0
  password: ""

jwt:
  issuer: auth-service
  accessTTL: 15m
  refreshTTL: 7d
  gracePeriod: 5m

oauth:
  google:
    clientID: ${GOOGLE_CLIENT_ID}
    clientSecret: ${GOOGLE_CLIENT_SECRET}
    redirectURL: http://localhost:8080/auth/google/callback
```

### Переменные окружения

| ENV | Описание |
|-----|----------|
| `POSTGRES_PASSWORD` | Пароль PostgreSQL |
| `REDIS_PASSWORD` | Пароль Redis |
| `JWT_SECRET` | Секрет для подписи JWT |
| `GOOGLE_CLIENT_ID` | Google OAuth Client ID |
| `GOOGLE_CLIENT_SECRET` | Google OAuth Client Secret |

---

## HTTP API

### Auth Endpoints

| Метод | Путь | Описание | Auth |
|-------|------|----------|------|
| `POST` | `/auth/register` | Регистрация | Нет |
| `POST` | `/auth/login` | Логин | Нет |
| `POST` | `/auth/refresh` | Обновление токенов | Нет |
| `POST` | `/auth/logout` | Выход | Refresh token |
| `POST` | `/auth/logout-all` | Выход со всех устройств | Access token |
| `GET` | `/auth/sessions` | Список сессий | Access token |
| `DELETE` | `/auth/sessions/:id` | Удалить сессию | Access token |
| `POST` | `/auth/introspect` | Проверка токена | Нет |

### OAuth Endpoints

| Метод | Путь | Описание |
|-------|------|----------|
| `GET` | `/auth/google` | Redirect на Google |
| `GET` | `/auth/google/callback` | Callback от Google |
| `GET` | `/auth/link/google` | Привязка Google аккаунта |
| `GET` | `/auth/link/google/callback` | Callback привязки |

### Примеры запросов

#### Регистрация

```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Device-Fingerprint: device-123" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123!"
  }'
```

Ответ:
```json
{
  "account_id": "550e8400-e29b-41d4-a716-446655440000",
  "access_token": "eyJhbGc...",
  "refresh_token": "550e8400.abc123...",
  "token_type": "Bearer",
  "expires_in": 900,
  "expires_at": "2026-04-27T21:00:00Z",
  "refresh_expires_at": "2026-05-04T20:45:00Z"
}
```

#### Логин

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Device-Fingerprint: device-123" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123!"
  }'
```

#### Refresh

```bash
curl -X POST http://localhost:8080/auth/refresh \
  -H "Content-Type: application/json" \
  -H "X-Device-Fingerprint: device-123" \
  -d '{
    "refresh_token": "550e8400.abc123..."
  }'
```

#### Introspect

```bash
curl -X POST http://localhost:8080/auth/introspect \
  -H "Authorization: Bearer eyJhbGc..."
```

Ответ:
```json
{
  "active": true,
  "account_id": "550e8400-e29b-41d4-a716-446655440000",
  "session_id": "session-123",
  "expires_at": "2026-04-27T21:00:00Z"
}
```

---

## OAuth Flow

### Архитектура OAuth

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ 1. GET /auth/google
       ↓
┌─────────────┐
│ auth-service│
└──────┬──────┘
       │ 2. Redirect to Google
       ↓
┌─────────────┐
│   Google    │
└──────┬──────┘
       │ 3. User authorizes
       │ 4. Callback with code
       ↓
┌─────────────┐
│ auth-service│
│  - Exchange code for tokens
│  - Get user info
│  - Check oauth_accounts table
│  - Create/link account
│  - Create session
└──────┬──────┘
       │ 5. Return JWT tokens
       ↓
┌─────────────┐
│   Client    │
└─────────────┘
```

### Две таблицы

**ВАЖНО:** OAuth использует ДВЕ таблицы:

1. **`accounts`** - основные пользователи
   - `id` (UUID)
   - `email`
   - `password_hash` (пустой для OAuth)
   - `created_at`, `updated_at`

2. **`oauth_accounts`** - связи с OAuth провайдерами
   - `id` (UUID)
   - `account_id` (FK → accounts.id)
   - `provider` (google/github/vk)
   - `provider_id` (ID в Google)
   - `access_token`, `refresh_token`, `expiry`

### OAuth Login сценарии

#### Сценарий 1: OAuth аккаунт существует

```
1. Ищем в oauth_accounts по (provider, provider_id)
2. Находим → берем account_id
3. Обновляем токены в oauth_accounts
4. Создаем сессию
```

#### Сценарий 2: Account существует, OAuth нет

```
1. Ищем в oauth_accounts → не находим
2. Ищем в accounts по email → находим
3. Создаем запись в oauth_accounts (привязка)
4. Создаем сессию
```

#### Сценарий 3: Ничего нет (новый пользователь)

```
1. Ищем в oauth_accounts → не находим
2. Ищем в accounts → не находим
3. Создаем Account в accounts (password_hash = "")
4. Создаем OAuthAccount в oauth_accounts
5. Создаем Outbox Event для user-service
6. Создаем сессию
```

---

## Базы данных

### PostgreSQL таблицы

#### accounts

```sql
CREATE TABLE accounts (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMPTZ
);
```

#### oauth_accounts

```sql
CREATE TABLE oauth_accounts (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    email TEXT NOT NULL DEFAULT '',
    access_token TEXT NOT NULL DEFAULT '',
    refresh_token TEXT NOT NULL DEFAULT '',
    expiry TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_oauth_accounts_provider_provider_id UNIQUE (provider, provider_id)
);
```

#### outbox_events

```sql
CREATE TABLE outbox_events (
    id UUID PRIMARY KEY,
    aggregate_type TEXT NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ
);
```

### Redis ключи

| Ключ | Тип | Описание |
|------|-----|----------|
| `session:{sid}` | Hash | Данные сессии |
| `refresh:{sid}` | Hash | Refresh token rotation |
| `user_sessions:{uid}` | Set | Список сессий пользователя |
| `blacklist:{jti}` | String | Отозванные access токены |

---

## Outbox Pattern

### Зачем нужен Outbox?

Синхронизация данных между `auth-service` (PostgreSQL) и `user-service` (MongoDB) через события.

### Архитектура

```
┌─────────────────┐
│  auth-service   │
│  POST /register │
│       ↓         │
│  ┌──────────┐   │
│  │ UseCase  │   │
│  └────┬─────┘   │
│       ↓         │
│  ┌──────────────────────┐
│  │  PostgreSQL TX       │
│  │  ┌────────────────┐  │
│  │  │ INSERT account │  │
│  │  └────────────────┘  │
│  │  ┌────────────────┐  │
│  │  │ INSERT outbox  │  │
│  │  └────────────────┘  │
│  │  COMMIT              │
│  └──────────┬───────────┘
└─────────────┼───────────┘
              ↓
       ┌──────────────┐
       │  Debezium    │ ← Читает WAL
       │     CDC      │
       └──────┬───────┘
              ↓
       ┌──────────────┐
       │    Kafka     │
       │ accounts.    │
       │   events     │
       └──────┬───────┘
              ↓
┌─────────────┴───────────┐
│    user-service         │
│  ┌──────────────────┐   │
│  │ Kafka Consumer   │   │
│  └────────┬─────────┘   │
│           ↓             │
│  ┌──────────────────┐   │
│  │ CreateUser UC    │   │
│  └────────┬─────────┘   │
│           ↓             │
│     INSERT user         │
└─────────────────────────┘
```

### Payload структура

```json
{
  "account_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "alice@example.com",
  "username": "alice",
  "display_name": "Alice Smith",
  "created_at": "2026-04-27T20:30:00Z"
}
```

### Проверка Outbox

```sql
-- Проверить события
SELECT * FROM outbox_events ORDER BY created_at DESC LIMIT 10;

-- Проверить payload
SELECT 
    id,
    event_type,
    payload::text,
    processed_at
FROM outbox_events 
WHERE processed_at IS NULL;
```

---

## Тестирование

### Запуск тестов

```bash
# Все тесты
go test ./...

# Только auth usecase
go test ./internal/usecase/auth/... -v

# С покрытием
go test ./... -cover
```

### Текущий статус

✅ **Auth UseCase** - все тесты проходят
- `TestAuthFlowIntegration_RegisterLoginRefreshLogout`
- `TestAuthFlowIntegration_RefreshFingerprintMismatch`
- `TestValidateAccessToken_RejectsBlacklisted`

⚠️ **PostgreSQL Repository** - требуют запущенную БД
- Тесты используют реальный PostgreSQL
- Нужно запустить миграции перед тестами

### Компиляция

```bash
go build ./...
# Exit Code: 0 ✅
```

---

## Troubleshooting

### Ошибка: relation "accounts" does not exist

**Причина:** Миграции не применены

**Решение:**
```bash
cd auth-service
make migrate-up
```

### Ошибка: connection refused (PostgreSQL)

**Причина:** PostgreSQL не запущен

**Решение:**
```bash
docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:latest
```

### Ошибка: connection refused (Redis)

**Причина:** Redis не запущен

**Решение:**
```bash
docker run -d -p 6379:6379 redis:latest
```

### OAuth не работает

**Проверить:**
1. `GOOGLE_CLIENT_ID` и `GOOGLE_CLIENT_SECRET` установлены
2. Redirect URL совпадает с настройками в Google Console
3. Таблица `oauth_accounts` создана

### Outbox события не попадают в Kafka

**Проверить:**
1. Записи есть в `outbox_events`: `SELECT * FROM outbox_events;`
2. Debezium connector запущен: `curl http://localhost:8083/connectors`
3. Kafka доступна: `docker ps | grep kafka`

---

## Полезные команды

```bash
# Компиляция
go build ./...

# Тесты
go test ./...

# Форматирование
go fmt ./...

# Миграции
make migrate-up
make migrate-down

# Запуск
go run cmd/server/main.go

# Docker
docker-compose up -d
docker-compose down
docker-compose logs -f auth-service

# PostgreSQL
psql -U postgres -d auth_db -c "SELECT * FROM accounts;"
psql -U postgres -d auth_db -c "SELECT * FROM oauth_accounts;"
psql -U postgres -d auth_db -c "SELECT * FROM outbox_events;"

# Redis
redis-cli
> KEYS session:*
> HGETALL session:550e8400-e29b-41d4-a716-446655440000
```

---

## Статус проекта

### ✅ Готово

- [x] Регистрация и логин
- [x] JWT токены с rotation
- [x] Refresh token replay protection
- [x] OAuth flow (Google)
- [x] Две таблицы (accounts + oauth_accounts)
- [x] Outbox Pattern
- [x] PostgreSQL с pgx/v5
- [x] Redis для сессий
- [x] Компиляция без ошибок
- [x] Auth usecase тесты проходят

### 🔄 В процессе

- [ ] OAuth тесты
- [ ] Integration тесты с БД
- [ ] Debezium настройка

### 📝 TODO

- [ ] GitHub OAuth provider
- [ ] VK OAuth provider
- [ ] MFA (2FA)
- [ ] Rate limiting
- [ ] Metrics (Prometheus)
- [ ] Tracing (Jaeger)

---

## Контакты и поддержка

Для вопросов и предложений создавайте Issues в репозитории.

**Версия:** 1.0.0  
**Последнее обновление:** 27.04.2026
