.PHONY: help up down build test lint loadtest migrate logs

help:
	@echo "Targets:"
	@echo "  up        - start full dev stack (docker-compose)"
	@echo "  down      - stop dev stack"
	@echo "  build     - build api + worker binaries"
	@echo "  test      - run unit tests with race detector"
	@echo "  lint      - golangci-lint"
	@echo "  loadtest  - run k6 baseline"
	@echo "  migrate   - apply SQL migrations to local Postgres"
	@echo "  logs      - tail api + worker logs"

up:
	docker-compose up -d --build

down:
	docker-compose down

build:
	CGO_ENABLED=0 go build -o bin/api    ./cmd/api
	CGO_ENABLED=0 go build -o bin/worker ./cmd/worker

test:
	go test ./... -race -cover

lint:
	golangci-lint run ./...

loadtest:
	k6 run loadtest/baseline.js

migrate:
	psql $${POSTGRES_URL:-postgres://app:app@localhost:5432/notifications?sslmode=disable} -f migrations/001_init.sql

logs:
	docker-compose logs -f api worker
