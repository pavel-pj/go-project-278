#!/bin/sh
set -e

echo "Running goose migrations..."
cd /app && goose postgres "postgres://${DB_USER}:${DB_PASSWORD}@postgres:5432/${DB_NAME}?sslmode=disable" up

echo "Migrations completed!"