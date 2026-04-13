.PHONY: migrate migrate-create migrate-up migrate-down migrate-status db-shell
 
#Загружаем переменные из .env
include .env
export

MIGRATIONS_DIR=./db/migrations
GOOSE_DIR=/app/db/migrations   
DB_URL_DOCKER=postgres://$(DB_USER):$(DB_PASSWORD)@postgres:5432/$(DB_NAME)?sslmode=disable
TEST_DB_URL_DOCKER=postgres://$(DB_USER_TEST):$(DB_PASSWORD_TEST)@postgres_test:5432/$(DB_TEST_NAME)?sslmode=disable

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

# ==================== ТЕСТОВАЯ БД ====================
 
goose-test-up:
	docker compose exec backend goose -dir $(GOOSE_DIR) postgres $(TEST_DB_URL_DOCKER) up

goose-test-status:
	docker compose exec backend goose -dir $(GOOSE_DIR) postgres $(TEST_DB_URL_DOCKER) status

 

main:
	docker compose exec backend go run main.go	

start:
	npx concurrently "docker compose exec backend go run main.go" "npm run start:backend" "npx start-hexlet-url-shortener-frontend"  
bash:
	docker compose exec backend sh

right:
	 sudo chown -R $$USER:$$USER ./
 
lint:
	@echo "Running golangci-lint in container..."
	@docker compose exec -e GOFLAGS="-buildvcs=false" backend golangci-lint run --timeout=5m ./...

## build: Собрать бинарник
build:
	 go build -o app -v ./...


# ==================== ТЕСТЫ ====================

.PHONY: test test-unit test-integration test-integration-single test-coverage test-clean

# Переменные для тестов
TEST_DB_HOST ?= localhost
TEST_DB_PORT ?= 5432
TEST_DB_USER ?= testuser
TEST_DB_PASSWORD ?= testpass
TEST_DB_NAME ?= testdb

# Запуск всех тестов

test:
	@echo "Running tests locally..."
	@docker compose exec -T backend sh -c "DOCKER_ENV=true go test -v -tags=integration ./internal/handlers/..."

 
# Запуск тестов с покрытием
test-coverage:
	@echo "📊 Running tests with coverage..."
	go test -v -tags=integration -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"


# Очистка кэша тестов
test-clean:
	@echo "Cleaning test cache..."
	go clean -testcache

# Запуск тестов в CI среде
test-ci: test-unit
	@echo "🏃 Running integration tests in CI mode..."
	go test -v -tags=integration -timeout 10m ./...


