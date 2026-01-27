.PHONY: migrate migrate-create migrate-up migrate-down migrate-status db-shell
 
#Загружаем переменные из .env
include .env
export

MIGRATIONS_DIR=./db/migrations
GOOSE_DIR=/app/db/migrations
DB_URL_DOCKER=postgres://$(DB_USER):$(DB_PASSWORD)@postgres:5432/$(DB_NAME)?sslmode=disable

#goose

goose-create:
	@echo "migration name :"
	@read -p "> " name && \
	docker compose exec backend goose -dir /app/db/migrations create $$name sql

goose-status:
	docker compose exec backend goose -dir $(GOOSE_DIR) postgres postgres://$(DB_USER):$(DB_PASSWORD)@postgres:5432/$(DB_NAME)?sslmode=disable  status
goose-up:
	docker compose exec backend goose -dir $(GOOSE_DIR) postgres postgres://$(DB_USER):$(DB_PASSWORD)@postgres:5432/$(DB_NAME)?sslmode=disable   up	
goose-rollback:
	docker compose exec backend goose -dir $(GOOSE_DIR) postgres postgres://$(DB_USER):$(DB_PASSWORD)@postgres:5432/$(DB_NAME)?sslmode=disable   down	

main:
	docker compose exec backend go run main.go		
bash:
	docker compose exec backend sh

right:
	 sudo chown -R $$USER:$$USER ./
 
test:
	 go test -race ./...
lint:
	golangci-lint run --timeout=5m ./...

## build: Собрать бинарник
build:
	 go build -o app -v ./...





