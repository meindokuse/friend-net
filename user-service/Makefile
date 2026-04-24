.PHONY: help deps test test-unit test-integration lint fmt docker-up docker-down clean

help: ## Показать помощь
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

deps: ## Установить зависимости
	go mod download
	go mod tidy

test: ## Запустить все тесты
	go test -v -race -coverprofile=coverage.out ./...

test-unit: ## Запустить только unit тесты
	go test -v -race ./internal/domain/...

test-integration: docker-up ## Запустить integration тесты
	@echo "Waiting for MongoDB to be ready..."
	@sleep 3
	go test -v -race ./internal/adapters/mongo/...

lint: ## Запустить линтер
	golangci-lint run ./...

fmt: ## Форматировать код
	go fmt ./...
	goimports -w .

docker-up: ## Запустить MongoDB
	docker-compose up -d
	@echo "MongoDB running on localhost:27017"
	@echo "Mongo Express UI: http://localhost:8081"

docker-down: ## Остановить MongoDB
	docker-compose down

docker-clean: ## Удалить MongoDB с данными
	docker-compose down -v

clean: ## Очистить артефакты
	rm -f coverage.out
	go clean -testcache

coverage: test ## Показать покрытие тестами
	go tool cover -html=coverage.out

.DEFAULT_GOAL := help
