# User Service New

`user-service-new` — новая версия user-сервиса в той же архитектурной модели, что и `auth-service-new`: явные слои, registry/wiring, отделение transport от use-case, четкие контракты между слоями.

---

## Цель адаптации

Привести `user-service` к единому стандарту сервисов проекта:

- одинаковый подход к слоям (`app` / `application` / `domain` / `infrastructure`);
- одинаковый подход к `config`, `cmd/main.go`, `internal/app.go`;
- одинаковый подход к graceful shutdown и lifecycle внешних коннекторов;
- одинаковый подход к описанию архитектуры и точек расширения.

---

## MVP статус `user-service-new`

**Текущий статус: `MVP готов`**.

В `user-service-new` перенесена прикладная логика `user-service` в новую архитектурную структуру по слоям (presentation/application/domain/infrastructure) без изменения целевого поведения API и основных инвариантов.

---

## Целевая структура проекта

```text
user-service-new/
├── cmd/
│   └── main.go                              # Точка входа: internal.New(ctx).Run(ctx)
│
├── config/
│   ├── config.go                            # Структуры конфига + singleton loader
│   ├── config.yaml                          # Локальный дефолтный конфиг
│   └── components.go                        # Константы (имена топиков, сервиса, etc.)
│
├── internal/
│   ├── app.go                               # DI/wiring всех компонентов + lifecycle
│   │
│   ├── app/                                 # Layer 1: Presentation
│   │   ├── user/v1/
│   │   │   ├── service.go                   # HTTP implementation + зависимости на use cases
│   │   │   ├── create.go                    # POST /users
│   │   │   ├── get.go                       # GET /users/{id}, /users/username/{username}
│   │   │   ├── me.go                        # /users/me* endpoint'ы
│   │   │   ├── search.go                    # /users/search, /users/me/list
│   │   │   └── response.go                  # единая сериализация/ошибки
│   │   │
│   │   └── event/v1/
│   │       └── account_created.go           # обработка account.created (consumer adapter)
│   │
│   ├── application/                         # Layer 2: Use cases
│   │   └── service/
│   │       ├── user/
│   │       │   ├── registry.go              # реестр use-case сервисов user
│   │       │   ├── create/
│   │       │   │   └── service.go
│   │       │   ├── get/
│   │       │   │   └── service.go
│   │       │   ├── update_profile/
│   │       │   │   └── service.go
│   │       │   ├── update_settings/
│   │       │   │   └── service.go
│   │       │   ├── change_email/
│   │       │   │   └── service.go
│   │       │   ├── change_phone/
│   │       │   │   └── service.go
│   │       │   ├── delete/
│   │       │   │   └── service.go
│   │       │   ├── update_last_seen/
│   │       │   │   └── service.go
│   │       │   ├── search/
│   │       │   │   └── service.go
│   │       │   ├── list/
│   │       │   │   └── service.go
│   │       │   └── batch_get/
│   │       │       └── service.go
│   │       │
│   │       └── event/
│   │           └── account_created/
│   │               └── service.go           # use-case обработки event из auth-service
│   │
│   ├── domain/                              # Layer 3: Domain
│   │   ├── entity/
│   │   │   └── user.go                      # User aggregate
│   │   ├── valueobject/
│   │   │   ├── email.go
│   │   │   ├── phone.go
│   │   │   └── username.go
│   │   └── errors/
│   │       └── errors.go
│   │
│   ├── infrastructure/                      # Layer 4: Infrastructure
│   │   ├── storage/
│   │   │   ├── registry.go
│   │   │   └── user/
│   │   │       └── storage.go               # MongoDB implementation репозитория
│   │   │
│   │   ├── messagebus/
│   │   │   ├── registry.go
│   │   │   └── consumer.go                  # Kafka reader wrapper + lifecycle
│   │   │
│   │   └── processor/
│   │       └── account_created.go           # adapter для вызова application/event service
│   │
│   └── pkg/
│       ├── closer/
│       │   └── closer.go
│       ├── connector/
│       │   ├── mongo/
│       │   │   └── connector.go
│       │   └── kafka/
│       │       └── connector.go
│       └── terror/
│           └── errors.go
│
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
├── .env.example
└── README.md
```

---

## Слои и правила

### Layer 1: Presentation (`internal/app/`)

- отвечает за HTTP/Kafka transport;
- валидирует input и переводит его в DTO application-слоя;
- не содержит бизнес-логики;
- не знает деталей Mongo/Kafka.

### Layer 2: Application (`internal/application/service/`)

- содержит бизнес-сценарии;
- определяет интерфейсы зависимостей (репозитории/гейтвеи);
- не зависит от HTTP, Mongo-driver, kafka-go.

### Layer 3: Domain (`internal/domain/`)

- чистые сущности и value object;
- инварианты домена;
- без transport/storage зависимостей.

### Layer 4: Infrastructure (`internal/infrastructure/`)

- реализации интерфейсов из application;
- работа с MongoDB/Kafka;
- сериализация/десериализация технических форматов.

---

## Потоки работы

### HTTP flow

1. `app/user/v1` принимает запрос.
2. Формирует DTO use-case.
3. Вызывает соответствующий `application/service/user/*`.
4. Отдает response через единый `response.go`.

### Event flow (`account.created`)

1. Kafka consumer получает событие из `accounts.events`.
2. `app/event/v1`/processor маппит payload в DTO.
3. Вызывает `application/service/event/account_created`.
4. Use-case создает User идемпотентно (`user.id = account_id`).

---

## API (целевая совместимость с текущим `user-service`)

- `POST /users`
- `GET /users/{id}`
- `GET /users/username/{username}`
- `POST /users/batch`
- `GET /users/search`
- `GET /users/me`
- `PATCH /users/me/profile`
- `PATCH /users/me/settings`
- `PATCH /users/me/email`
- `PATCH /users/me/phone`
- `DELETE /users/me`
- `POST /users/me/last-seen`
- `GET /users/me/list`

---

## Migration checklist (`user-service` -> `user-service-new`)

### 1) Bootstrapping и wiring

- [x] Перенести точку входа на `cmd/main.go` -> `internal.New(ctx).Run(ctx)`.
- [x] Вынести все init-части в `internal/app.go` (mongo, kafka, http, shutdown).
- [x] Подключить `pkg/closer` для единообразного graceful shutdown.

### 2) Слой Presentation

- [x] Разделить handlers по фичам в `internal/app/user/v1` (без business logic).
- [x] Вынести единый error/response mapper в `response.go`.
- [x] Добавить middleware-слой для auth-контекста `/users/me*`.

### 3) Слой Application

- [x] Разбить текущий `internal/usecase/user` на пакетные use-case сервисы.
- [x] В каждом use-case определить интерфейсы зависимостей локально.
- [ ] Собрать `registry.go` для user use-cases.

### 4) Слой Domain

- [x] Перенести сущности и VO в `internal/domain/entity` и `internal/domain/valueobject`.
- [x] Унифицировать ошибки домена в `internal/domain/errors`.

### 5) Infrastructure

- [x] Перенести Mongo repository в `internal/infrastructure/storage/user/storage.go`.
- [ ] Выделить `storage/registry.go`.
- [x] Перенести Kafka consumer wiring в `infrastructure/messagebus`.
- [x] Вынести обработку `account.created` в отдельный processor adapter.

### 6) Контракты и совместимость

- [ ] Сохранить внешний HTTP API без breaking changes.
- [ ] Сохранить идемпотентность account-created consumer.
- [ ] Сохранить текущие бизнес-инварианты (soft delete, versioning, pagination).

### 7) Качество и проверка

- [ ] Обновить unit/integration тесты под новую структуру.
- [x] Добавить smoke запуск (mongo + kafka optional).
- [x] Проверить `go test ./...` и локальный e2e happy-path.

---

## Принципы миграции (важно)

- Миграция выполняется **без изменения бизнес-логики**.
- Меняется только организационная архитектура, wiring и контракты между слоями.
- Поведение API и event-processing должно остаться эквивалентным текущему `user-service`.
