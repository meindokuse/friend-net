# User Service

Микросервис управления пользователями для cloud-drive проекта.

## Архитектура

- **Domain Layer**: Бизнес-логика, Value Objects, агрегаты
- **Usecase Layer**: Сценарии использования, контракты репозиториев
- **Adapters**: MongoDB репозиторий
- **Controllers**: HTTP handlers (chi router) + Kafka consumer
- **DTO**: Модели запросов/ответов для HTTP API

## Основные возможности

### HTTP API

- **POST /users** - Создание пользователя
- **GET /users/{id}** - Получение пользователя по ID
- **GET /users/username/{username}** - Получение по username
- **POST /users/batch** - Batch получение по списку ID
- **GET /users/search?q=query** - Поиск пользователей

#### Authenticated endpoints (требуют auth middleware)

- **GET /users/me** - Получение текущего пользователя
- **PATCH /users/me/profile** - Обновление профиля
- **PATCH /users/me/settings** - Обновление настроек
- **PATCH /users/me/email** - Смена email
- **PATCH /users/me/phone** - Смена телефона
- **DELETE /users/me** - Soft-delete пользователя
- **POST /users/me/last-seen** - Обновление last_seen_at
- **GET /users/me/list?cursor=...&limit=...** - Keyset пагинация

### Kafka Consumer

Слушает топик `accounts.events` и обрабатывает события:

- **AccountCreated** - Создаёт User с тем же ID что и Account (идемпотентно)

## Конфигурация

### Переменные окружения (.env)

```env
# Application
APP_ENV=local

# HTTP Server
HTTP_ADDR=:8081
SHUTDOWN_TIMEOUT=10s

# MongoDB
MONGO_URI=mongodb://localhost:27017
MONGO_DATABASE=user_service
MONGO_TIMEOUT=10s

# Kafka
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC=accounts.events
KAFKA_GROUP_ID=user-service
KAFKA_ENABLED=true

# Logging
LOG_LEVEL=info
```

### YAML конфигурация (config/config.yaml)

Используется для локальной разработки. В production все настройки берутся из ENV.

## Запуск

### Локально

1. Запустите MongoDB:
```bash
docker run -d -p 27017:27017 --name mongo mongo:latest
```

2. Запустите Kafka (опционально):
```bash
docker-compose up -d kafka
```

3. Скопируйте .env.example в .env и настройте:
```bash
cp .env.example .env
```

4. Запустите сервис:
```bash
go run cmd/main.go
```

Или соберите бинарник:
```bash
go build -o bin/user-service.exe ./cmd/main.go
./bin/user-service.exe
```

### Docker

```bash
docker build -t user-service .
docker run -p 8081:8081 --env-file .env user-service
```

## Зависимости

- **MongoDB** - основная БД
- **Kafka** - event streaming (опционально, можно отключить через `KAFKA_ENABLED=false`)
- **common module** - общие события (AccountCreated)

## Особенности реализации

### Soft Delete

Все операции чтения автоматически фильтруют soft-deleted пользователей (`deleted_at != nil`).

### Optimistic Locking

Update операции используют `version` поле для предотвращения race conditions. При конфликте возвращается `ErrVersionConflict`.

### Keyset Pagination

List endpoint использует keyset пагинацию по `(username, _id)` для стабильной и эффективной пагинации больших датасетов.

### Idempotency

Kafka consumer обрабатывает дубликаты событий идемпотентно:
- Если User с таким ID уже существует → игнорируем (успешная обработка)
- Используем `User.ID = Account.ID` для гарантии идемпотентности

### Error Handling

Трёхуровневая система ошибок:
1. **Domain errors** (`domain/user/errors.go`) - бизнес-логика
2. **Usecase errors** (`usecase/user/errors.go`) - сценарии использования
3. **Adapter errors** - MongoDB драйвер → domain errors

## Тестирование

```bash
# Запустить все тесты
go test ./...

# Тесты MongoDB (требуют запущенный MongoDB)
go test ./internal/adapters/mongo/...

# Тесты с verbose
go test -v ./...
```

## Документация

Подробная документация по domain модели, usecase и DTO находится в [DOCUMENTATION.md](./DOCUMENTATION.md).

## Graceful Shutdown

Сервис корректно завершает работу при получении SIGINT/SIGTERM:
1. Останавливает HTTP server (с таймаутом из `SHUTDOWN_TIMEOUT`)
2. Останавливает Kafka consumer
3. Закрывает соединение с MongoDB
