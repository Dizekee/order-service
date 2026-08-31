FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Исправляем: бинарник называется order, не alertmanager
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o order ./cmd/order-service

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/order .
COPY --from=builder /app/.env .

# Миграции копируем, но применять будем при старте приложения
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./order"]