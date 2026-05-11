# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /auth-service ./cmd

# Runtime stage
FROM alpine:3.19

WORKDIR /app


# Copy binary from builder
COPY --from=builder /auth-service /app/auth-service
COPY --from=builder /app/config/config.yaml /app/config/config.yaml

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/auth-service"]
