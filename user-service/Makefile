.PHONY: build run test clean docker-build docker-run tidy

# Build the service
build:
	go build -o bin/user-service.exe ./cmd/main.go

# Run the service
run:
	go run ./cmd/main.go

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Tidy dependencies
tidy:
	go mod tidy

# Build Docker image
docker-build:
	docker build -t user-service:latest .

# Run Docker container
docker-run:
	docker run -p 8081:8081 --env-file .env user-service:latest

# Run MongoDB for local development
mongo-up:
	docker run -d -p 27017:27017 --name mongo mongo:latest

# Stop MongoDB
mongo-down:
	docker stop mongo && docker rm mongo

# Format code
fmt:
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Run all checks before commit
pre-commit: fmt tidy test
	@echo "All checks passed!"
