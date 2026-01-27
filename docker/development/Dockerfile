FROM golang:1.25-alpine

WORKDIR /app
 
RUN apk add --no-cache curl tar postgresql-client && \
    curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.1/migrate.linux-amd64.tar.gz | tar xvz 

RUN go install github.com/pressly/goose/v3/cmd/goose@latest

# Устанавливаем Air
RUN go install github.com/air-verse/air@latest

#Установнка sqlc
RUN go install github.com/pressly/goose/v3/cmd/goose@latest


COPY scripts/migrate.sh /usr/local/bin/migrate
RUN chmod +x /usr/local/bin/migrate

# Копируем зависимости
COPY go.mod go.sum ./
RUN go mod download
 
#COPY . .

EXPOSE 8080

#CMD ["air", "-c", ".air.toml"]
CMD ["/bin/sh"]