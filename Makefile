.PHONY: run build test test-integration migrate migrate-up seed db-up db-down

run:
	go run ./cmd/server

build:
	go build ./...

test:
	go test ./...

test-integration:
	ALLOW_INTEGRATION_DB_RESET=true go test ./internal/integration -v

migrate migrate-up:
	go run ./cmd/migrate

seed:
	go run ./cmd/seed

db-up:
	docker compose up -d postgres

db-down:
	docker compose down
