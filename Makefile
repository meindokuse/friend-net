# Переменные для удобства
DC_TRAEFIK = docker compose -f ./traefikV2/docker-compose.yaml
DC_AUTH    = docker compose -f ./auth-service/docker-compose.yaml
DC_USER    = docker compose -f ./user-service/docker-compose.yaml
DC_INFRA   = docker compose -f ./infra/docker-compose.yaml # если вынес кафку отдельно

.PHONY: network up down restart logs ps clean

# 1. Создание внешней сети (нужно запустить один раз)
network:
	docker network inspect traefik-net >/dev/null 2>&1 || \
	docker network create traefik-net

# 2. Запуск всего стека
up: network
	$(DC_TRAEFIK) up -d
	$(DC_INFRA) up -d
	$(DC_AUTH) up -d
	$(DC_USER) up -d
	@echo "🚀 Все сервисы запущены!"

# 3. Остановка всего
down:
	$(DC_USER) down
	$(DC_AUTH) down
	$(DC_TRAEFIK) down
	@echo "🛑 Все сервисы остановлены."

# 4. Рестарт (удобно при правке конфигов)
restart: down up

# 5. Сборка и запуск (если поменял код в Go)
build: network
	$(DC_AUTH) up -d --build
	$(DC_USER) up -d --build

# 6. Просмотр логов (всех сразу)
logs:
	docker compose -f ./traefikV2/docker-compose.yaml -f ./auth-service/docker-compose.yaml -f ./user-service/docker-compose.yaml logs -f

# 7. Статус контейнеров
ps:
	docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"