# auth-service

`auth-service` — сервис аутентификации и сессий на Go (Gin), построенный в стиле `usecase + Clean Architecture`.

Основные задачи сервиса:
- регистрация и логин пользователей;
- выдача и обновление токенов (`access` + `refresh`);
- управление сессиями устройств;
- OAuth-вход через Google;
- introspection access-токена для интеграции с другими сервисами.

---

## 1. Архитектура и слои

### 1.1 Слои проекта

| Слой | Путь | Назначение |
|---|---|---|
| Entry point | `cmd/server` | Поднимает HTTP-сервер, DI, middleware, маршруты |
| Controllers | `internal/controllers/http` | HTTP-обвязка: парсинг/валидация входа, маппинг ошибок/ответов |
| Usecase | `internal/usecase/auth`, `internal/usecase/oauth` | Бизнес-правила auth и OAuth |
| Domain | `internal/domain/user`, `internal/domain/session` | Доменные сущности и контракты данных |
| Adapters | `internal/adapters/postgresql`, `internal/adapters/redis` | Реализации репозиториев/хранилищ |
| Infra | `internal/infra` | Интеграции со внешними провайдерами (Google OAuth) |
| Shared pkg | `pkg/*` | JWT, парольный хешер, логгер, клиенты Postgres/Redis |

### 1.2 Принцип взаимодействия (кратко)

1. HTTP-запрос приходит в controller.
2. Controller собирает input и вызывает usecase.
3. Usecase работает через интерфейсы `DB/Redis/OAuthRepository`.
4. Адаптеры ходят в Postgres/Redis/Google API.
5. Usecase возвращает доменный результат, controller формирует HTTP-ответ.

---

## 2. Быстрый старт

### 2.1 Требования

- Docker + Docker Compose
- или локально: Go `1.25+`, Postgres, Redis

### 2.2 Запуск в Docker

```bash
cd auth-service
docker compose up --build
```

Что стартует:
- `postgres` (`5432`)
- `redis` (`6379`)
- `migrate` (накатывает SQL миграции)
- `auth-service` (`8080`)

Проверка:
```bash
curl http://localhost:8080/healthz
```

---

## 3. Конфигурация

Основной файл: `config.yaml`.
Часть секретов может переопределяться через env.

### 3.1 Ключевые параметры

| Блок | Параметр | Что делает |
|---|---|---|
| `server` | `httpAddr` | Адрес HTTP-сервера |
| `controller` | `cookieDomain`, `cookieSecure`, `refreshCookieName` | Настройки refresh-cookie |
| `postgres` | host/port/user/database | Подключение к Postgres |
| `redis` | `addr`, `db`, timeouts | Подключение к Redis |
| `jwt` | `issuer`, `accessTTL`, `refreshTTL`, `gracePeriod` | Политика токенов |
| `pass` | `cost` | BCrypt cost |
| `oauth.google` | `clientID`, `clientSecret`, `redirectURL`, `scopes` | Google OAuth |

### 3.2 ENV override

| ENV | Что переопределяет |
|---|---|
| `POSTGRES_PASSWORD` | пароль Postgres |
| `REDIS_PASSWORD` | пароль Redis |
| `JWT_SECRET` | JWT signing secret |
| `GOOGLE_CLIENT_ID` | Google OAuth client id |
| `GOOGLE_CLIENT_SECRET` | Google OAuth client secret |

Пример файла: `.env.example`.

---

## 4. HTTP API

Базовый префикс: `/auth`.

### 4.1 Auth endpoints

| Метод | Путь | Назначение | Требует access token |
|---|---|---|---|
| `POST` | `/auth/register` | Регистрация + авто-логин | Нет |
| `POST` | `/auth/login` | Логин | Нет |
| `POST` | `/auth/refresh` | Обновление access/refresh | Нет |
| `POST` | `/auth/logout` | Логаут текущей сессии | Желательно (или refresh) |
| `POST` | `/auth/logout-all` | Логаут всех устройств | Да |
| `GET` | `/auth/sessions` | Список активных сессий | Да |
| `DELETE` | `/auth/sessions/:session_id` | Ревок конкретной сессии | Да |
| `POST` | `/auth/introspect` | Проверка access token | Нет |

### 4.2 OAuth endpoints

| Метод | Путь | Назначение |
|---|---|---|
| `GET` | `/auth/google` | Redirect на Google |
| `GET` | `/auth/google/callback` | Callback логина Google |
| `GET` | `/auth/link/google` | Redirect для link-аккаунта |
| `GET` | `/auth/link/google/callback` | Callback link-аккаунта |

### 4.3 Заголовки и cookie

| Имя | Где используется | Обязательно |
|---|---|---|
| `Authorization: Bearer <access>` | `sessions`, `logout-all`, `revoke`, introspect | Для защищённых ручек |
| `X-Refresh-Token` | `refresh`, `logout` | Альтернатива cookie/body |
| `X-Device-Fingerprint` | `login`, `register`, `refresh`, OAuth callback | Рекомендуется |
| `refresh_token` cookie | `refresh`, `logout` | Альтернатива header/body |
| `X-Trace-ID` | Логгирование/трейс | Опционально (генерируется автоматически) |

### 4.4 Формат токен-ответа

Успешные `register/login/refresh` возвращают:

| Поле | Тип | Описание |
|---|---|---|
| `access_token` | string | JWT access token |
| `refresh_token` | string | opaque refresh token |
| `token_type` | string | Обычно `Bearer` |
| `expires_in` | number | TTL access в секундах |
| `expires_at` | datetime | Время истечения access |
| `refresh_expires_at` | datetime | Время истечения refresh |
| `user_id` | string | ID пользователя |

---

## 5. Данные и хранилища

### 5.1 Postgres таблицы

| Таблица | Назначение |
|---|---|
| `users` | Локальные пользователи |
| `oauth_accounts` | Привязки OAuth-провайдеров к пользователям |

#### `users`

| Поле | Тип | Комментарий |
|---|---|---|
| `id` | UUID PK | ID пользователя |
| `email` | TEXT UNIQUE | Логин |
| `password_hash` | TEXT | BCrypt hash |
| `mfa_enabled` | BOOLEAN | Reserved |
| `mfa_secret` | TEXT | Reserved |
| `created_at` | TIMESTAMPTZ | Дата создания |

#### `oauth_accounts`

| Поле | Тип | Комментарий |
|---|---|---|
| `id` | UUID PK | ID записи |
| `user_id` | UUID FK -> users.id | Владелец |
| `provider` | TEXT | `google` и т.д. |
| `provider_id` | TEXT | ID пользователя у провайдера |
| `email` | TEXT | email от провайдера |
| `access_token` | TEXT | токен провайдера |
| `refresh_token` | TEXT | refresh провайдера |
| `expiry` | TIMESTAMPTZ | expiry provider token |
| `created_at`, `updated_at` | TIMESTAMPTZ | Аудит |

### 5.2 Redis ключи

| Ключ | Тип | Что хранится |
|---|---|---|
| `session:{sid}` | Hash | Статус сессии, user_id, fingerprint hash, IP, UA, даты |
| `refresh:{sid}` | Hash | `current`/`prev` hash refresh-token (rotation) |
| `user_sessions:{uid}` | Set | Список session id пользователя |
| `blacklist:{jti}` | String + TTL | Отозванные access token jti |

---

## 6. End-to-End сценарии (полный флоу)

### 6.1 Сценарий A: Register -> Login session -> Token response

1. Клиент вызывает `POST /auth/register` с `email/password`.
2. Controller валидирует JSON и вызывает `auth.Register(...)`.
3. Usecase `Register`:
- хеширует пароль (`pkg/pass`);
- создаёт пользователя в Postgres через `UserRepository.Save`.
4. После успешной регистрации controller делает авто-логин (`LoginUser`).
5. `LoginUser`:
- читает пользователя по email;
- проверяет пароль;
- считает активные сессии в Redis;
- при переполнении лимита выталкивает самую старую сессию.
6. `createSessionAndTokens`:
- создаёт `sessionID`;
- сохраняет `session:{sid}` в Redis;
- генерирует access JWT + refresh token;
- сохраняет hash refresh в `refresh:{sid}`.
7. Controller ставит `refresh_token` cookie и отдаёт JSON с access/refresh метаданными.

### 6.2 Сценарий B: Refresh token rotation + replay protection

1. Клиент отправляет `POST /auth/refresh` (cookie/header/body).
2. Controller собирает refresh token и fingerprint.
3. Usecase `Refresh`:
- парсит refresh (`session_id.random`);
- грузит сессию из Redis;
- проверяет `session.IsActive()`;
- сравнивает fingerprint hash;
- получает `RefreshPair` из Redis.
4. Сравнивается hash random части:
- `MatchCurrent` -> обычная ротация;
- `MatchPrev` -> grace-rotation;
- `MatchNone` -> считается reuse-атакой, сессия revocation.
5. На успехе обновляются `access_token`, `refresh_token`, cookie и TTL.

### 6.3 Сценарий C: Introspect из другого сервиса

1. Внешний сервис получает токен клиента.
2. Вызывает `POST /auth/introspect` (Bearer или JSON body).
3. `ValidateAccessToken`:
- проверяет подпись/exp JWT;
- проверяет blacklist по `jti` в Redis.
4. Ответ:
- `{"active": false}` если токен невалиден;
- `{"active": true, "user_id": "...", "session_id": "...", "expires_at": ...}` если валиден.

Это ключевой E2E-флоу интеграции auth-service с API Gateway/другими микросервисами.

### 6.4 Сценарий D: Logout / Logout-all / Session management

#### Logout текущей сессии
1. Клиент вызывает `POST /auth/logout` с access (желательно) и/или refresh.
2. Usecase извлекает `session_id` из refresh или access claims.
3. Помечает сессию revoked в Redis и удаляет refresh-pair.
4. Access token blacklist по `jti` (если access передан).
5. Controller очищает `refresh_token` cookie.

#### Logout всех устройств
1. `POST /auth/logout-all` с Bearer access.
2. Через introspection получаем `user_id`.
3. Redis-адаптер ревокает все session id из `user_sessions:{uid}`.

#### Список устройств
1. `GET /auth/sessions` с Bearer access.
2. Возвращается список активных сессий пользователя с флагом `current`.

### 6.5 Сценарий E: OAuth Google login

1. Клиент открывает `GET /auth/google`.
2. Controller генерирует `state` и редиректит на Google OAuth URL.
3. Google делает callback в `/auth/google/callback?code=...&state=...`.
4. Usecase OAuth:
- обменивает `code` на OAuth token;
- тянет профиль пользователя;
- ищет `oauth_accounts(provider, provider_id)`;
- если нет, пытается матчить локального пользователя по email;
- если и его нет, создаёт нового пользователя + OAuth account.
5. Создаётся локальная сессия и локальные access/refresh токены.
6. Controller ставит refresh-cookie и возвращает auth payload.

---

## 7. Контракт интеграции для других сервисов

Рекомендуемая схема:

1. Клиент идёт в API Gateway.
2. Gateway берёт Bearer token.
3. Gateway вызывает `POST /auth/introspect`.
4. При `active=true` прокидывает в downstream:
- `X-User-ID`
- `X-Session-ID`
- при необходимости `X-Token-Exp`.
5. Downstream сервисы не парсят JWT самостоятельно (single source of truth = auth-service).

Плюсы:
- единая политика revoke/blacklist;
- меньше дублирования криптологии;
- проще ротация секретов.

---

## 8. Proto контракт

`proto/auth.proto` описывает gRPC-контракт:
- `ValidateToken`
- `GetUserIDByToken`

На текущий момент в рантайме поднят HTTP API (gRPC-сервер не поднят в `main.go`), но proto уже готов как контракт для следующего этапа.

---

## 9. Логи и observability

- Middleware `RequestContextLogger`:
  - проставляет `X-Trace-ID` (или генерирует);
  - логирует start/finish каждого запроса;
  - пишет метод, путь, статус, duration.
- Все usecase и adapters логируют основные шаги и ошибки через `slog`.

---

## 10. Security notes

| Механизм | Как реализован |
|---|---|
| Пароли | BCrypt (`pkg/pass`) |
| Access token | JWT HS256 + `jti` |
| Refresh token | Opaque (`session_id.random`) |
| Refresh storage | Хранится только hash random-part в Redis |
| Replay defense | `current/prev` + `gracePeriod` + revoke при reuse |
| Device binding | Fingerprint hash в сессии |
| Logout hardening | Access blacklist по `jti` |

---

## 11. Известные ограничения и что учесть

1. Для `link/google` сейчас ожидается `user_id` в `gin.Context`, но отдельный auth middleware-инжектор user-id в роуты link пока не подключён.
2. `state` для OAuth сейчас генерируется, но полноценная серверная валидация state (хранение в Redis + TTL + single-use) отмечена как TODO.
3. Есть `proto`-контракт, но gRPC-сервер ещё не включён в runtime (HTTP уже рабочий).

---

## 12. Быстрые примеры запросов

### Register
```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Device-Fingerprint: dev-machine-1" \
  -d '{"email":"test@example.com","password":"StrongPass123"}'
```

### Login
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Device-Fingerprint: dev-machine-1" \
  -d '{"email":"test@example.com","password":"StrongPass123"}'
```

### Refresh (через body)
```bash
curl -X POST http://localhost:8080/auth/refresh \
  -H "Content-Type: application/json" \
  -H "X-Device-Fingerprint: dev-machine-1" \
  -d '{"refresh_token":"<your_refresh_token>"}'
```

### Introspect
```bash
curl -X POST http://localhost:8080/auth/introspect \
  -H "Authorization: Bearer <access_token>"
```

### Sessions
```bash
curl http://localhost:8080/auth/sessions \
  -H "Authorization: Bearer <access_token>"
```

### Logout
```bash
curl -X POST http://localhost:8080/auth/logout \
  -H "Authorization: Bearer <access_token>" \
  -H "X-Refresh-Token: <refresh_token>"
```

---

## 13. Структура директорий

```text
auth-service/
  cmd/server
  internal/
    adapters/
      postgresql/
      redis/
    config/
    controllers/http/
    domain/
      session/
      user/
    dto/
    infra/
    usecase/
      auth/
      oauth/
  migrations/
  pkg/
    jwt/
    logger/
    pass/
    postgresql/
    redis/
  proto/
```

