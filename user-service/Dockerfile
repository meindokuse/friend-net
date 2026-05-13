# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

ENV GOPROXY=https://proxy.golang.org,direct


# 1. Сначала копируем common, так как он указан в replace
COPY common/ /common/

# 2. Копируем файлы модулей user-service
COPY user-service/go.mod user-service/go.sum ./

# Теперь go mod download увидит /common в корне образа (т.к. в go.mod стоит ../common)
RUN go mod download

# 3. Копируем исходники самого сервиса
COPY user-service/ .

# 4. Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /user-service-bin ./cmd

# Runtime stage
FROM alpine:3.19
WORKDIR /app


# Копируем бинарник из билдера
COPY --from=builder /user-service-bin /app/user-service
# Убедись, что папка config и файл существуют в user-service/config/config.yaml
COPY --from=builder /app/config/config.yaml /app/config/config.yaml

# Создаем пользователя для безопасности
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8081

ENTRYPOINT ["/app/user-service"]