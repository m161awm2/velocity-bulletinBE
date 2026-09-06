.PHONY: run build test test-integration migrate-up migrate-down seed db-up db-down

run:
	go run ./cmd/server

build:
	go build ./...

test:
	go test ./...

test-integration:
	ALLOW_INTEGRATION_DB_RESET=true go test ./internal/integration -v

migrate-up:
	go run ./cmd/migrate -action up

migrate-down:
	go run ./cmd/migrate -action down -steps 1

seed:
	go run ./cmd/seed

db-up:
	docker compose up -d postgres

db-down:
	docker compose down
