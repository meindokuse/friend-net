# User Service - Setup Complete ✅

## Что было реализовано

### 1. Kafka Consumer (`internal/controllers/kafka/consumer.go`)
- ✅ Подписка на топик `accounts.events`
- ✅ Десериализация события `AccountCreated` из common module
- ✅ Создание User с тем же ID что и Account (идемпотентность)
- ✅ Обработка дубликатов через `ErrUsernameAlreadyTaken`
- ✅ Manual commit после успешной обработки
- ✅ Graceful shutdown

### 2. Конфигурация (`internal/config/config.go`)
- ✅ Структура Config с Server, Mongo, Kafka, Logger
- ✅ Использование `cleanenv` для загрузки из YAML + ENV
- ✅ Приоритет: env-default → YAML → ENV
- ✅ Поддержка `CONFIG_PATH` для кастомного пути к YAML
- ✅ `.env` и `.env.example` с полными настройками
- ✅ `config/config.yaml` для локальной разработки

### 3. Main Application (`cmd/main.go`)
- ✅ Инициализация MongoDB с retry логикой
- ✅ Создание репозитория и usecase
- ✅ Настройка HTTP router (chi) с middleware
- ✅ Регистрация всех HTTP handlers
- ✅ Запуск Kafka consumer в отдельной горутине
- ✅ Graceful shutdown для HTTP + Kafka + MongoDB
- ✅ Structured logging (slog) с уровнями

### 4. Недостающие методы
- ✅ Добавлен `Search()` метод в MongoDB adapter (`internal/adapters/mongo/read.go`)
  - Поиск по username или display_name (case-insensitive regex)
  - Поддержка limit/offset пагинации
  - Фильтрация soft-deleted пользователей

### 5. Зависимости
- ✅ Исправлен `go.mod` - все зависимости теперь direct
- ✅ Добавлен replace для common module
- ✅ `go mod tidy` выполнен успешно
- ✅ Workspace (`go.work`) корректно настроен

### 6. Документация
- ✅ `README.md` - полное руководство по запуску и использованию
- ✅ `DOCUMENTATION.md` - детальная документация domain/usecase/DTO (уже была)
- ✅ Комментарии в коде

### 7. DevOps
- ✅ `Dockerfile` - multi-stage build для production
- ✅ `docker-compose.yml` - полный стек (MongoDB + Kafka + User Service)
- ✅ `Makefile` - удобные команды для разработки
- ✅ `.dockerignore` (если нужен)

## Структура проекта

```
user-service/
├── cmd/
│   └── main.go                    # Entry point
├── internal/
│   ├── adapters/
│   │   └── mongo/                 # MongoDB implementation
│   │       ├── repository.go      # Constructor + indexes
│   │       ├── read.go            # Read operations + Search
│   │       ├── write.go           # Create, Update
│   │       ├── mapper.go          # Domain ↔ Document
│   │       └── errors.go          # Error translation
│   ├── config/
│   │   └── config.go              # Configuration loader
│   ├── controllers/
│   │   ├── handlers/              # HTTP handlers
│   │   │   ├── user_handler.go
│   │   │   └── router.go
│   │   └── kafka/                 # Kafka consumer
│   │       └── consumer.go
│   ├── domain/
│   │   ├── shared/vo/             # Value Objects
│   │   └── user/                  # User aggregate
│   ├── dto/                       # HTTP request/response models
│   └── usecase/
│       └── user/                  # Business logic
├── config/
│   └── config.yaml                # Local development config
├── .env                           # Environment variables
├── .env.example                   # Example environment variables
├── Dockerfile                     # Production build
├── docker-compose.yml             # Local development stack
├── Makefile                       # Development commands
├── README.md                      # User guide
├── DOCUMENTATION.md               # Technical documentation
└── go.mod                         # Dependencies
```

## Как запустить

### Вариант 1: Локально (без Docker)

1. Запустите MongoDB:
```bash
make mongo-up
```

2. (Опционально) Запустите Kafka:
```bash
docker-compose up -d zookeeper kafka
```

3. Запустите сервис:
```bash
make run
# или
go run cmd/main.go
```

### Вариант 2: Docker Compose (полный стек)

```bash
docker-compose up -d
```

Это запустит:
- MongoDB на порту 27017
- Kafka на порту 9092
- User Service на порту 8081

### Вариант 3: Только сборка

```bash
make build
./bin/user-service.exe
```

## Проверка работы

### HTTP API

```bash
# Health check (если добавить endpoint)
curl http://localhost:8081/health

# Создание пользователя
curl -X POST http://localhost:8081/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "email": "alice@example.com",
    "display_name": "Alice"
  }'

# Поиск пользователей
curl "http://localhost:8081/users/search?q=alice"
```

### Kafka Consumer

Отправьте событие в Kafka:

```bash
# Используя kafka-console-producer
docker exec -it user-service-kafka kafka-console-producer \
  --broker-list localhost:9092 \
  --topic accounts.events

# Вставьте JSON:
{"account_id":"550e8400-e29b-41d4-a716-446655440000","email":"test@example.com","username":"testuser","display_name":"Test User","created_at":"2026-04-27T20:00:00Z"}
```

Проверьте логи:
```bash
docker logs -f user-service
```

## Следующие шаги

### 1. Интеграция с auth-service
- [ ] Настроить Debezium CDC для PostgreSQL outbox
- [ ] Создать outbox_events таблицу в auth-service
- [ ] Настроить Debezium connector для публикации в Kafka
- [ ] Протестировать end-to-end flow: Account создан → User создан

### 2. Authentication Middleware
- [ ] Добавить JWT middleware для защищённых endpoints
- [ ] Извлекать user_id из токена и проставлять в context
- [ ] Подключить middleware в router.go

### 3. Observability
- [ ] Добавить Prometheus metrics
- [ ] Добавить distributed tracing (OpenTelemetry)
- [ ] Настроить health check endpoint
- [ ] Добавить readiness/liveness probes для k8s

### 4. Testing
- [ ] Добавить integration tests для HTTP handlers
- [ ] Добавить tests для Kafka consumer
- [ ] Настроить CI/CD pipeline

### 5. Production Readiness
- [ ] Добавить rate limiting
- [ ] Настроить CORS
- [ ] Добавить request validation middleware
- [ ] Настроить connection pooling для MongoDB
- [ ] Добавить circuit breaker для внешних зависимостей

## Известные ограничения

1. **Auth Middleware не подключен** - все endpoints доступны без аутентификации
2. **Нет health check endpoint** - нужен для k8s probes
3. **Нет metrics** - нужен для мониторинга
4. **Нет rate limiting** - может быть DDoS
5. **Kafka consumer не имеет retry с backoff** - при ошибке сразу retry

## Troubleshooting

### MongoDB connection failed
```bash
# Проверьте что MongoDB запущен
docker ps | grep mongo

# Проверьте логи
docker logs user-service-mongo
```

### Kafka consumer не получает сообщения
```bash
# Проверьте что топик создан
docker exec -it user-service-kafka kafka-topics --list --bootstrap-server localhost:9092

# Проверьте consumer group
docker exec -it user-service-kafka kafka-consumer-groups \
  --bootstrap-server localhost:9092 \
  --describe --group user-service
```

### Build failed
```bash
# Очистите кеш и пересоберите
go clean -cache
go mod tidy
go build ./...
```

## Контакты и поддержка

Для вопросов и предложений создавайте issue в репозитории.
