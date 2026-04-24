# Интеграция MongoDB Repository

## Быстрый старт

### 1. Запуск MongoDB

```bash
cd user-service
docker-compose up -d
```

Это запустит:
- MongoDB на `localhost:27017`
- Mongo Express UI на `http://localhost:8081`

### 2. Использование в коде

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/google/uuid"
    
    "github.com/meindokuse/cloud-drive/user-service/internal/adapters/mongo"
    domainuser "github.com/meindokuse/cloud-drive/user-service/internal/domain/user"
    "github.com/meindokuse/cloud-drive/user-service/internal/domain/shared/vo"
)

func main() {
    ctx := context.Background()

    // Подключение к MongoDB
    cfg := mongo.Config{
        URI:      "mongodb://localhost:27017",
        Database: "user_service",
        Timeout:  10 * time.Second,
    }

    db, err := mongo.Connect(ctx, cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer mongo.Disconnect(ctx, db)

    // Создание репозитория (автоматически создаст индексы)
    repo, err := mongo.NewUserRepository(db)
    if err != nil {
        log.Fatal(err)
    }

    // Создание пользователя
    username := vo.MustNewUsername("johndoe")
    email := vo.MustNewEmail("john@example.com")
    
    user, err := domainuser.NewUser(
        uuid.New(),
        username,
        &email,
        nil,
        "John Doe",
    )
    if err != nil {
        log.Fatal(err)
    }

    // Сохранение в БД
    if err := repo.Create(ctx, user); err != nil {
        log.Fatal(err)
    }

    // Получение из БД
    found, err := repo.GetByUsername(ctx, username)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("User found: %s (%s)", found.Username(), found.Profile().DisplayName)

    // Обновление профиля
    bio := "Software Engineer"
    if err := found.UpdateProfile("John Doe Jr.", &bio, nil); err != nil {
        log.Fatal(err)
    }

    if err := repo.Update(ctx, found); err != nil {
        log.Fatal(err)
    }

    // Поиск пользователей
    results, err := repo.Search(ctx, "john", 10, 0)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Found %d users", len(results))
}
```

## Обработка ошибок

Репозиторий возвращает доменные ошибки:

```go
err := repo.Create(ctx, user)
switch {
case errors.Is(err, domainuser.ErrUsernameAlreadyTaken):
    // Username уже занят
case errors.Is(err, domainuser.ErrEmailAlreadyTaken):
    // Email уже занят
case errors.Is(err, domainuser.ErrPhoneAlreadyTaken):
    // Телефон уже занят
default:
    // Другая ошибка
}

err = repo.Update(ctx, user)
switch {
case errors.Is(err, domainuser.ErrVersionConflict):
    // Optimistic lock conflict - нужно перечитать и повторить
case errors.Is(err, domainuser.ErrUserNotFound):
    // Пользователь не найден
}
```

## Optimistic Locking

При конкурентных обновлениях используется optimistic locking:

```go
// Поток 1
user1, _ := repo.GetByID(ctx, userID)
user1.UpdateProfile("Name 1", nil, nil)
repo.Update(ctx, user1) // OK

// Поток 2 (параллельно)
user2, _ := repo.GetByID(ctx, userID)
user2.UpdateProfile("Name 2", nil, nil)
repo.Update(ctx, user2) // ErrVersionConflict

// Правильная обработка в потоке 2:
for retries := 0; retries < 3; retries++ {
    user, _ := repo.GetByID(ctx, userID)
    user.UpdateProfile("Name 2", nil, nil)
    
    err := repo.Update(ctx, user)
    if !errors.Is(err, domainuser.ErrVersionConflict) {
        break
    }
    // Retry
}
```

## Batch операции

```go
// Получение нескольких пользователей за один запрос
ids := []uuid.UUID{id1, id2, id3}
users, err := repo.GetByIDs(ctx, ids)
```

## Поиск

```go
// Поиск по username или display_name (case-insensitive)
users, err := repo.Search(ctx, "john", 20, 0)

// Пагинация
page1, _ := repo.Search(ctx, "john", 10, 0)  // первые 10
page2, _ := repo.Search(ctx, "john", 10, 10) // следующие 10
```

## UpdateLastSeen

Специальный метод для обновления last_seen без изменения version:

```go
// Вызывается очень часто (при каждом запросе пользователя)
// НЕ бампит version, чтобы не создавать конфликты
err := repo.UpdateLastSeen(ctx, userID)
```

## Тестирование

```bash
# Запустить MongoDB
make docker-up

# Запустить integration тесты
make test-integration

# Или напрямую
go test ./internal/adapters/mongo/... -v
```

## Production настройки

### Connection Pool

```go
clientOpts := options.Client().
    ApplyURI(cfg.URI).
    SetMaxPoolSize(100).
    SetMinPoolSize(10).
    SetMaxConnIdleTime(30 * time.Second)
```

### Retry логика

```go
clientOpts := options.Client().
    ApplyURI(cfg.URI).
    SetRetryWrites(true).
    SetRetryReads(true)
```

### Мониторинг

```go
import "go.mongodb.org/mongo-driver/event"

monitor := &event.CommandMonitor{
    Started: func(ctx context.Context, e *event.CommandStartedEvent) {
        log.Printf("MongoDB command started: %s", e.CommandName)
    },
    Succeeded: func(ctx context.Context, e *event.CommandSucceededEvent) {
        log.Printf("MongoDB command succeeded: %s (duration: %v)", 
            e.CommandName, e.Duration)
    },
    Failed: func(ctx context.Context, e *event.CommandFailedEvent) {
        log.Printf("MongoDB command failed: %s (error: %v)", 
            e.CommandName, e.Failure)
    },
}

clientOpts := options.Client().
    ApplyURI(cfg.URI).
    SetMonitor(monitor)
```

## Индексы

Индексы создаются автоматически при вызове `NewUserRepository()`:

- `username` — unique (для быстрого поиска и уникальности)
- `email` — unique, sparse (sparse = null значения не индексируются)
- `phone` — unique, sparse
- `deleted_at` — для фильтрации удалённых пользователей
- Text index на `username` + `profile.display_name` — для полнотекстового поиска

### Проверка индексов

```javascript
// В mongo shell или Mongo Express
db.users.getIndexes()
```

## Миграции

Для production рекомендуется использовать инструмент миграций (например, migrate):

```bash
# Установка
go install -tags 'mongodb' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Создание миграции
migrate create -ext json -dir migrations -seq create_users_collection
```

Пример миграции (`001_create_users_collection.up.json`):

```json
[
  {
    "createIndexes": "users",
    "indexes": [
      {
        "key": {"username": 1},
        "name": "username_unique",
        "unique": true
      }
    ]
  }
]
```
