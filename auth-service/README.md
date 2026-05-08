# Auth Service New

Сервис аутентификации на Go с JWT, refresh rotation, OAuth (Google) и Outbox-паттерном.

## MVP статус

**Текущий статус: `MVP готов`** (за исключением пунктов ниже).

Ниже список оставшихся задач, которые выведены за рамки текущего MVP.

### Важно (для стабильного MVP)

- Добавить минимальные интеграционные тесты на auth-flow: register/login/refresh/logout/introspect.
- Добавить smoke-тесты запуска с Postgres + Redis (и Kafka в optional-режиме).
- Разделить миграции от init-скриптов Docker: в `docker-compose` не стоит монтировать `*.down.sql` как init SQL.
- Уточнить модель `oauth_accounts.expiry` и использование времени (сейчас используются и unix/int64, и timestamp).
- Добавить middleware аутентификации для защищенных OAuth endpoint'ов (link/get linked/unlink), вместо прямого ожидания `ctx.Get("account_id")`.

### Желательно (после MVP)

- Добавить rate limit и базовую защиту от brute-force на `/auth/login` и `/auth/refresh`.
- Добавить аудит-логи по security-событиям (login fail, token reuse, session revoke all).
- Добавить OpenAPI/Swagger.

---

## Архитектура и структура

Проект следует Clean Architecture: зависимости направлены сверху вниз.

```
Presentation (HTTP/Gin) -> Application (use cases) -> Domain (entities)
                                      |
                                      v
                           Infrastructure (DB/Redis/Kafka/OAuth)
```

### 1) Presentation слой — `internal/app/`

**Что делает:**

- принимает HTTP-запросы;
- валидирует вход;
- собирает DTO для use case;
- сериализует HTTP-ответ.

**Папки:**

- `internal/app/auth/v1` — классические auth endpoint'ы;
- `internal/app/oauth/v1` — OAuth endpoint'ы.

### 2) Application слой — `internal/application/service/`

**Что делает:**

- реализует бизнес-сценарии;
- определяет интерфейсы зависимостей (репозитории/гейтвеи);
- оркестрирует несколько инфраструктурных компонентов.

**Группы use case:**

- `auth`: login/register/refresh/logout/introspect/get_sessions/revoke_session;
- `oauth`: login/link/unlink/get_linked.

### 3) Domain слой — `internal/domain/entity/`

**Что делает:**

- хранит предметные сущности;
- инкапсулирует доменные состояния и простые доменные операции.

**Ключевые сущности:**

- `Account`;
- `OAuthAccount`;
- `Session`;
- `RefreshPair`;
- `OutboxEvent`.

### 4) Infrastructure слой — `internal/infrastructure/`

**Что делает:**

- дает конкретные реализации интерфейсов из Application слоя;
- работает с внешними системами.

**Папки:**

- `storage/account` — PostgreSQL для аккаунтов;
- `storage/oauth` — PostgreSQL для OAuth-привязок;
- `storage/outbox` — PostgreSQL outbox-events;
- `storage/session` — Redis для сессий/refresh-pair/blacklist;
- `gateway/oauth` — Google OAuth клиент;
- `messagebus` — Kafka producer;
- `flusher` — polling-воркер для Outbox.

---

## Как сервис работает внутри

### Login flow (`/auth/login`)

1. Handler валидирует email/password и собирает fingerprint.
2. Use case `auth/login`:
  - ищет аккаунт по email;
  - сверяет пароль;
  - ограничивает число активных сессий;
  - создает сессию в Redis;
  - генерирует access + refresh;
  - сохраняет hash refresh pair.
3. Handler отдает токены + ставит refresh cookie.

### Refresh flow (`/auth/refresh`)

1. Refresh берется из header/cookie/body.
2. Use case `auth/refresh`:
  - парсит refresh token (sessionID + randomPart);
  - проверяет сессию и fingerprint;
  - сверяет hash с current/prev;
  - ротирует refresh pair;
  - при reuse-атаке ревокает сессию.
3. Возвращает новую пару токенов.

### Logout / Introspect

- `logout` ревокает текущую сессию, blacklist'ит jti access-токена.
- `logout-all` ревокает все сессии account.
- `introspect` валидирует access token и проверяет blacklist.

### OAuth login (`/auth/google`, `/auth/google/callback`)

1. Генерируется `state`, сохраняется в httpOnly cookie и выполняется redirect на Google.
2. Callback обменивает code на provider token.
3. Получает профиль пользователя.
4. Либо находит существующую OAuth привязку, либо создает/линкует аккаунт.
5. Создает локальную сессию и выдает JWT/refresh.

### Outbox flow

1. При создании account событие `account.created` пишется в `outbox_events` в одной транзакции с `accounts`.
2. Flusher периодически читает непроцессед события.
3. Публикует в Kafka.
4. Помечает `processed_at`.

---

## API (фактически подключенные роуты)

### Health

- `GET /healthz`

### Auth

- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/refresh`
- `POST /auth/logout`
- `POST /auth/logout-all`
- `GET /auth/sessions`
- `DELETE /auth/sessions/:session_id`
- `POST /auth/introspect`

### OAuth (подключено в роутере)

- `GET /auth/google`
- `GET /auth/google/callback`
- `GET /auth/link/google`
- `GET /auth/link/google/callback`
- `GET /auth/linked`
- `DELETE /auth/linked/:provider`

---

## Конфигурация

Источник конфигурации:

- если есть `CONFIG_PATH` (по умолчанию `config/config.yaml`) и файл существует — читается YAML;
- иначе читаются env-переменные.

Минимально обязательные параметры:

- `JWT_SECRET` (>= 32 символов);
- `JWT_REFRESH_SECRET` (>= 32 символов);
- `GOOGLE_CLIENT_ID`;
- `GOOGLE_CLIENT_SECRET`.

Полный пример env: `.env.example`.

---

## Локальный запуск

1. Поднять инфраструктуру:
  - `docker-compose up -d postgres redis`
2. Применить `up` миграции через мигратор (golang-migrate/goose).
3. Запустить сервис:
  - `go run ./cmd`

Проверка:

- `GET http://localhost:8080/healthz`

---

## Технологии

- `gin` — HTTP;
- `pgx` — PostgreSQL;
- `go-redis` — Redis;
- `golang-jwt/jwt` — access token;
- `bcrypt` (`x/crypto`) — хеширование паролей;
- `oauth2` — Google OAuth;
- `sarama` — Kafka.

