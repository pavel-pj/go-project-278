# ==================== STAGE 1: Builder ====================
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Устанавливаем инструменты для сборки
RUN apk add --no-cache curl tar postgresql-client git

# Устанавливаем goose, air, sqlc (ТОЛЬКО для сборки!)
RUN go install github.com/pressly/goose/v3/cmd/goose@latest && \
    go install github.com/air-verse/air@latest && \
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Копируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем всё для разработки
COPY . .

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/app ./main.go

# ==================== STAGE 2: Dev Runtime ====================
FROM golang:1.25-alpine

WORKDIR /app

# Устанавливаем ТОЛЬКО необходимое для разработки
RUN apk add --no-cache postgresql-client curl git

# Копируем бинарник из builder (чтобы сразу было что запускать)
COPY --from=builder /app/app /app/app

# Копируем goose, air, sqlc из builder
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY --from=builder /go/bin/air /usr/local/bin/air
COPY --from=builder /go/bin/sqlc /usr/local/bin/sqlc

# Копируем зависимости go modules
COPY --from=builder /go/pkg/mod /go/pkg/mod

# Копируем скрипт запуска
COPY bin/run.sh /app/run.sh
RUN chmod +x /app/run.sh

# Монтируем исходный код при запуске
VOLUME /app

EXPOSE 8080

# Можно запускать по-разному
#CMD ["air", "-c", ".air.toml"]
# или
CMD ["/bin/sh"]