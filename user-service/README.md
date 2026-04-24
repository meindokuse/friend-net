# User Service

Микросервис управления пользователями для социальной сети. Реализует Domain-Driven Design с чистой архитектурой.

## Архитектура

```
user-service/
├── internal/
│   ├── domain/           # Доменная модель (entities, value objects, events)
│   │   ├── user/         # User aggregate
│   │   └── shared/       # Общие VO (Username, Email, Phone)
│   ├── usecase/          # Бизнес-логика (use cases)
│   │   └── user/         # User use cases + repository interface
│   └── adapters/         # Внешние адаптеры
│       └── mongo/        # MongoDB реализация репозитория
└── cmd/                  # Entry points (TODO)
```

## Технологии

- **Go 1.25**
- **MongoDB** — основное хранилище
- **DDD** — Domain-Driven Design
- **Clean Architecture** — зависимости направлены внутрь

## Особенности реализации

### Domain Model

- **User** — корневой агрегат с инкапсулированным состоянием
- **Value Objects**: Username, Email, Phone с валидацией
- **Optimistic Locking** через поле `version`
- **Soft Delete** — пользователи не удаляются физически
- **Domain Events**: UserCreated, UserProfileUpdated, UserDeleted

### Repository Pattern

Интерфейс `UserRepository` определён в usecase слое (Dependency Inversion):

```go
type UserRepository interface {
    Create(ctx context.Context, u *User) error
    Update(ctx context.Context, u *User) error
    GetByID(ctx context.Context, id uuid.UUID) (*User, error)
    GetByUsername(ctx context.Context, username Username) (*User, error)
    GetByEmail(ctx context.Context, email Email) (*User, error)
    GetByPhone(ctx context.Context, phone Phone) (*User, error)
    GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*User, error)
    Search(ctx context.Context, query string, limit, offset int) ([]*User, error)
    UpdateLastSeen(ctx context.Context, id uuid.UUID) error
}
```

### MongoDB Schema

```javascript
{
  _id: UUID,
  username: String (unique, lowercase),
  email: String (unique, sparse),
  phone: String (unique, sparse, E.164 format),
  profile: {
    display_name: String,
    bio: String?,
    avatar_url: String?
  },
  settings: {
    privacy: {
      who_can_message: "everyone" | "friends" | "nobody",
      who_can_see_last_seen: "everyone" | "friends" | "nobody",
      who_can_see_profile: "everyone" | "friends" | "nobody"
    },
    language: String,
    timezone: String
  },
  verification: {
    email_verified: Boolean,
    phone_verified: Boolean
  },
  is_active: Boolean,
  created_at: ISODate,
  updated_at: ISODate,
  last_seen_at: ISODate?,
  deleted_at: ISODate?,
  version: Int
}
```

### Индексы

- `username` — unique
- `email` — unique, sparse
- `phone` — unique, sparse
- `deleted_at` — для фильтрации удалённых
- Text index на `username` + `profile.display_name` для поиска

## Локальная разработка

### Запуск MongoDB

```bash
docker-compose up -d
```

### Установка зависимостей

```bash
go mod download
```

### Запуск тестов

```bash
# Unit + Integration тесты (требуется MongoDB на localhost:27017)
go test ./internal/adapters/mongo/... -v

# Только unit тесты domain
go test ./internal/domain/... -v
```

### Переменные окружения

```env
MONGO_URI=mongodb://localhost:27017
MONGO_DATABASE=user_service
MONGO_TIMEOUT=10s
```

## Use Cases (TODO)

- [x] Domain model
- [x] Repository interface
- [x] MongoDB adapter
- [ ] CreateUser use case
- [ ] UpdateProfile use case
- [ ] UpdateSettings use case
- [ ] ChangeEmail use case
- [ ] ChangePhone use case
- [ ] SoftDelete use case
- [ ] GetUser use case
- [ ] SearchUsers use case
- [ ] BatchGetUsers use case
- [ ] UpdateLastSeen use case

## API Endpoints (TODO)

```
POST   /users              — создать пользователя
GET    /users/:id          — получить профиль
PATCH  /users/me           — обновить свой профиль
GET    /users/search?q=    — поиск пользователей
POST   /users/:id/block    — заблокировать
GET    /users/me/blocked   — список заблокированных
```

## События (Kafka)

- `user.created` — новый пользователь зарегистрирован
- `user.profile.updated` — профиль обновлён
- `user.deleted` — пользователь удалён (soft delete)

## Мониторинг (TODO)

- Health check: `/health`
- Readiness: `/ready`
- Metrics: `/metrics` (Prometheus)

## Production Checklist

- [ ] Graceful shutdown
- [ ] Structured logging (JSON)
- [ ] Distributed tracing (OpenTelemetry)
- [ ] Circuit breaker для MongoDB
- [ ] Rate limiting
- [ ] Request ID propagation
- [ ] MongoDB connection pooling
- [ ] Backup strategy
- [ ] Monitoring & alerting
