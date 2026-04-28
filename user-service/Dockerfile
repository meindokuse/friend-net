# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum from workspace
COPY go.work go.work.sum* ./
COPY common/go.mod common/go.sum* ./common/
COPY user-service/go.mod user-service/go.sum* ./user-service/

# Download dependencies
WORKDIR /app/user-service
RUN go mod download

# Copy source code
WORKDIR /app
COPY common/ ./common/
COPY user-service/ ./user-service/

# Build
WORKDIR /app/user-service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /app/bin/user-service ./cmd/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary
COPY --from=builder /app/bin/user-service .

# Copy config (optional, can use ENV only)
COPY --from=builder /app/user-service/config ./config

# Expose port
EXPOSE 8081

# Run
CMD ["./user-service"]
