# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

ENV GOPROXY=https://goproxy.io,https://goproxy.cn,direct

# Copy common module first (referenced by replace directive)
COPY common/ /common/

# Copy module files and download dependencies
COPY analytic-service/go.mod analytic-service/go.sum ./

RUN go mod download

# Copy service source
COPY analytic-service/ .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /analytic-service-bin ./cmd

# Runtime stage
FROM alpine:3.19
WORKDIR /app

COPY --from=builder /analytic-service-bin /app/analytic-service
COPY --from=builder /app/config/config.yaml /app/config/config.yaml

RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8082

ENTRYPOINT ["/app/analytic-service"]
