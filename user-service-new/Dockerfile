FROM golang:1.25-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates tzdata
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /user-service-new ./cmd

FROM alpine:3.19
WORKDIR /app
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /user-service-new /app/user-service-new
COPY --from=builder /app/config/config.yaml /app/config/config.yaml
EXPOSE 8081
ENTRYPOINT ["/app/user-service-new"]
