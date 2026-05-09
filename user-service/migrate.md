# Clean Architecture Migration Guide for Go Services

## Концепция

Каждый сервис строится по принципу **Clean Architecture (DDD)** с 4 слоями и **Dependency Inversion**.

### Ключевые принципы

1. **Зависимости направлены внутрь** — внешние слои зависят от внутренних, но не наоборот
2. **Внутренние слои не знают о внешних** — domain не знает о БД, application не знает о proto/HTTP
3. **Интерфейсы определяются в application** — не в infrastructure
4. **Каждый use case — отдельная папка** — с собственными интерфейсами зависимостей

---

## Полная структура сервиса

```
services/{service-name}/
│
├── cmd/
│   └── main.go                              # Только internal.New(ctx).Run(ctx)
│
├── config/
│   ├── config.go                            # Структура конфига + синглтон Instance()
│   ├── config.yaml                          # YAML конфигурация
│   └── components.go                        # Константы: названия сервисов, топиков Kafka
│
├── api/
│   └── {service-name}/
│       └── {domain}/v1/
│           └── {domain}.proto               # gRPC контракты
│
├── internal/
│   │
│   ├── app.go                               # Структура App со всеми зависимостями
│   ├── init.go                              # Функции инициализации по порядку
│   │
│   │   ═══════════════════════════════════════════════════════════════
│   │   СЛОЙ 1: PRESENTATION (внешний)
│   │   ═══════════════════════════════════════════════════════════════
│   │
│   ├── app/                                 # gRPC handlers (transport layer)
│   │   └── {domain}/v1/
│   │       ├── service.go                   # struct Implementation { services *Registry }
│   │       ├── {method1}.go                 # func (i *Implementation) Method1(ctx, *Request) (*Response, error)
│   │       └── {method2}.go
│   │
│   │   Ответственность:
│   │   - Приём gRPC/HTTP запросов
│   │   - Конвертация proto → DTO
│   │   - Вызов application service
│   │   - Конвертация entity → proto
│   │
│   │   НЕ содержит:
│   │   - Бизнес-логики
│   │   - Знаний о БД, Kafka
│   │
│   │   ═══════════════════════════════════════════════════════════════
│   │   СЛОЙ 2: APPLICATION (use cases)
│   │   ═══════════════════════════════════════════════════════════════
│   │
│   ├── application/
│   │   └── service/{domain}/
│   │       ├── registry.go                  # type Registry struct { UseCase1, UseCase2, ... }
│   │       │                                # func NewRegistry(storage, gateway, messagebus) *Registry
│   │       │
│   │       └── {use_case}/                  # Каждый use case — отдельная папка
│   │           ├── service.go               # Интерфейсы + бизнес-логика
│   │           └── dto.go                   # Data Transfer Objects (опционально)
│   │
│   │   Ответственность:
│   │   - БИЗНЕС-ЛОГИКА (валидация, оркестрация)
│   │   - Определение интерфейсов зависимостей
│   │   - Координация работы storage/gateway/events
│   │
│   │   НЕ содержит:
│   │   - Знаний о proto, HTTP
│   │   - Знаний о PostgreSQL, Kafka
│   │   - Прямых импортов infrastructure
│   │
│   │   ═══════════════════════════════════════════════════════════════
│   │   СЛОЙ 3: DOMAIN (ядро)
│   │   ═══════════════════════════════════════════════════════════════
│   │
│   ├── domain/
│   │   ├── entity/                          # Сущности
│   │   │   ├── {entity1}.go                 # struct + конструктор New{} + методы
│   │   │   └── {entity2}.go
│   │   │
│   │   └── value_object/                    # Value objects (опционально)
│   │       └── {value_object}.go
│   │
│   │   Ответственность:
│   │   - Чистые Go структуры без тегов db/json/proto
│   │   - Бизнес-правила внутри сущностей
│   │   - Конструкторы и методы инкапсуляции
│   │
│   │   НЕ содержит:
│   │   - Любых внешних зависимостей (только uuid, decimal, time)
│   │   - Тегов сериализации БД/JSON/Proto
│   │
│   │   ═══════════════════════════════════════════════════════════════
│   │   СЛОЙ 4: INFRASTRUCTURE (реализация)
│   │   ═══════════════════════════════════════════════════════════════
│   │
│   ├── infrastructure/
│   │   │
│   │   ├── storage/                         # Репозитории (PostgreSQL, Redis)
│   │   │   ├── registry.go                  # type Registry struct { Task, Category }
│   │   │   │                                # func NewRegistry(pool) *Registry
│   │   │   │
│   │   │   └── {entity}/
│   │   │       ├── {entity}.go              # struct Storage, реализует интерфейсы application
│   │   │       └── dao/
│   │   │           └── {entity}.go          # DAO для маппинга БД ↔ Entity
│   │   │
│   │   ├── gateway/                         # gRPC клиенты к другим сервисам
│   │   │   ├── registry.go                  # type Registry struct { Profile, Task }
│   │   │   │                                # func NewRegistry(grpcConn) *Registry
│   │   │   │
│   │   │   └── {service}/
│   │   │       └── {service}.go             # struct Client, реализует интерфейсы application
│   │   │
│   │   └── messagebus/                      # Kafka producer/consumer
│   │       ├── registry.go                  # type Registry struct { Producers }
│   │       │
│   │       ├── producer/
│   │       │   └── producer.go              # Реализует event.Flusher
│   │       │
│   │       └── subscriber/                  # Только для сервисов-подписчиков (analytic)
│   │           └── subscriber.go
│   │
│   │   Ответственность:
│   │   - Реализация интерфейсов из application
│   │   - Работа с БД, Kafka, gRPC
│   │   - Конвертация DAO ↔ Entity
│   │
│   │   ═══════════════════════════════════════════════════════════════
│   │   EVENTS (опционально, для event-driven)
│   │   ═══════════════════════════════════════════════════════════════
│   │
│   ├── events/
│   │   └── {event_name}/
│   │       └── convertor.go                 # Конвертация entity → Event для Kafka
│   │
│   │   ═══════════════════════════════════════════════════════════════
│   │   PKG (внутренние утилиты)
│   │   ═══════════════════════════════════════════════════════════════
│   │
│   └── pkg/
│       ├── closer/                          # Graceful shutdown
│       ├── connector/                       # Подключения к внешним системам
│       │   ├── postgres/
│       │   └── kafka/
│       ├── event/                           # Event buffer + flusher
│       │   ├── buffer.go                    # Аккумулятор событий
│       │   ├── context.go                   # Buffer в context
│       │   ├── event.go                     # Типы Event, Events
│       │   └── flusher/platform/outbox.go   # Альтернативный flusher
│       ├── grpc/
│       │   └── intercept/                   # Interceptors
│       ├── msgbus/
│       │   ├── producer/                    # Kafka producer wrapper
│       │   └── subscriber/                  # Kafka consumer wrapper
│       ├── terror/                          # Кастомные ошибки
│       ├── transaction/                     # Транзакции
│       └── pipe/                            # Pipeline паттерн
│
├── tools/
│   ├── migrations/                          # SQL миграции
│   └── xo_templates/                        # Шаблоны для codegen
│
├── go.mod
├── go.sum
├── Makefile
├── buf.yaml                                 # Buf конфигурация
├── buf.gen.local.yaml
└── buf.gen.external.yaml
```

---

## Направление зависимостей

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│     PRESENTATION (internal/app/)                                            │
│         │                                                                   │
│         │ импортирует                                                       │
│         ▼                                                                   │
│     APPLICATION (internal/application/service/)                             │
│         │                                                                   │
│         │ импортирует                                                       │
│         ▼                                                                   │
│     DOMAIN (internal/domain/entity/)                                        │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘

                              ▲
                              │
                              │ реализует интерфейсы
                              │
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│     INFRASTRUCTURE (internal/infrastructure/)                               │
│                                                                             │
│     - storage (реализует interfaces из application)                         │
│     - gateway (реализует interfaces из application)                         │
│     - messagebus (реализует interfaces из application)                      │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Правила для каждого слоя

### DOMAIN

| Разрешено | Запрещено |
|-----------|-----------|
| Стандартная библиотека | Импорт `database/sql`, `pgx` |
| `github.com/gofrs/uuid` | Импорт proto пакеты |
| `github.com/shopspring/decimal` | Теги `db:"..."`, `json:"..."` |
| `time.Time` | Теги `protobuf:"..."` |
| Конструкторы `NewEntity()` | Внешние зависимости |
| Методы изменения состояния | |

### APPLICATION

| Разрешено | Запрещено |
|-----------|-----------|
| Импорт `internal/domain/entity` | Импорт `internal/infrastructure/*` |
| Определение интерфейсов | Импорт proto пакеты |
| Использование интерфейсов | Прямое использование `*pgxpool.Pool` |
| `context.Context` | Знание о PostgreSQL, Kafka |
| Бизнес-логика | Знание о HTTP, gRPC |
| DTO структуры | |

### PRESENTATION

| Разрешено | Запрещено |
|-----------|-----------|
| Импорт `internal/application/service` | Бизнес-логика |
| Импорт proto пакеты | Прямой вызов storage |
| Импорт `internal/domain/entity` (для конвертации) | Прямое подключение к БД |
| Конвертация proto ↔ DTO | |
| Вызов application service | |

### INFRASTRUCTURE

| Разрешено | Запрещено |
|-----------|-----------|
| Импорт `internal/domain/entity` | Определение бизнес-интерфейсов |
| Импорт `github.com/jackc/pgx/v5` | |
| Импорт `github.com/IBM/sarama` | |
| Реализация интерфейсов application | |
| DAO для маппинга | |

---

## Интерфейсы: где определять

```
┌─────────────────────────────────────────────────────────────────┐
│ APPLICATION LAYER                                               │
│                                                                 │
│ internal/application/service/task/create_task/service.go       │
│                                                                 │
│ type Creator interface {                                        │
│     CreateTask(ctx, *entity.Task) error                        │
│ }                                                               │
│                                                                 │
│ type Category interface {                                       │
│     GetCategory(ctx, id string) (entity.Category, error)       │
│ }                                                               │
│                                                                 │
│ type PermissionChecker interface {                              │
│     CheckPermission(ctx, userID int64) (bool, error)           │
│ }                                                               │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ реализуют
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ INFRASTRUCTURE LAYER                                            │
│                                                                 │
│ storage/task/task.go        → implements Creator               │
│ storage/category/category.go → implements Category             │
│ gateway/profile/profile.go  → implements PermissionChecker     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Правило:** Интерфейс определяется там, где он используется (application), а не там, где реализуется (infrastructure).

---

## Registry паттерн

Каждый слой имеет Registry для агрегации компонентов:

```
storage.Registry     → { Task *Storage, Category *Storage }
gateway.Registry     → { Profile *Client }
messagebus.Registry  → { Producers }
service.Registry     → { CreateTask *Service, GetTask *Service, ... }
```

Инициализация происходит в `internal/init.go`:

```
1. initPostgres     → pool
2. initGrpcConn     → grpcConn map[string]grpc.ClientConnInterface
3. initMessageBus   → messagebus.Registry
4. initGateways     → gateway.Registry (нужен grpcConn)
5. initStorages     → storage.Registry (нужен pool)
6. initServices     → service.Registry (нужен storage, gateway, messagebus)
7. initMainServer   → gRPC server
8. initControllers  → регистрация gRPC handlers
```

---

## Event-driven архитектура

### Producer (в task-service)

```
Application Service
       │
       │ buf, ctx := event.WithContext(ctx, flusher)
       │ event.Add(ctx, task_created.New(task))
       │ buf.Flush(ctx)
       │
       ▼
MessageProducer.Flush(ctx, events)
       │
       ▼
Kafka.SyncProducer.SendMessages()
       │
       ▼
     Kafka
```

### Consumer (в analytic-service)

```
     Kafka
       │
       ▼
sarama.ConsumerGroup.Consume()
       │
       ▼
MessageHandler.Handle(ctx, session, message)
       │
       ▼
Application Service (accept_task.Create)
       │
       ▼
Storage.Create()
```

---

## Типичный use case файл

```
internal/application/service/{domain}/{use_case}/
│
├── service.go
│   ├── type SomeDependency interface { ... }    # Интерфейсы зависимостей
│   ├── type Service struct { ... }               # Поля — интерфейсы
│   ├── func NewService(deps ...) *Service        # Конструктор с DI
│   └── func (s *Service) Execute(ctx, dto)       # Бизнес-логика
│
└── dto.go (опционально)
    └── type SomeDTO struct { ... }               # Входные данные
```

---

## Типичный storage файл

```
internal/infrastructure/storage/{entity}/
│
├── {entity}.go
│   ├── type Storage struct { pool *pgxpool.Pool }
│   ├── func NewStorage(pool) *Storage
│   └── func (s *Storage) SomeMethod(ctx, ...)    # Реализация интерфейса
│
└── dao/
    └── {entity}.go
        ├── type Task struct { ... }              # Поля с тегами db
        └── func (d *Task) ConvertTo() *entity    # DAO → Entity
```

---

