# Backend commands. The root Makefile delegates here; both are safe to call
# directly. `make` with no arguments lists everything.

.DEFAULT_GOAL := help
.PHONY: help setup db db-reset run build test test-integration watch coverage lint fmt tidy psql

# The dev database, for `make psql` and for running the app.
DB_URL ?= postgres://$(shell whoami)@localhost:5432/crud_dev?sslmode=disable
# Integration tests TRUNCATE before every test, so they get their own database.
# Pointing them at crud_dev silently wipes your seed data every time you run
# the suite, and then the app looks broken for no visible reason.
TEST_DB_URL ?= postgres://$(shell whoami)@localhost:5432/crud_test?sslmode=disable

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-17s\033[0m %s\n", $$1, $$2}'

setup: ## Resolve dependencies and write go.sum
	@go mod tidy
	@echo "dependencies resolved"

# Schema lives at the repo root: one set of migrations, two backends.
db: ## Create the database and apply migrations
	@../scripts/initdb.sh

db-reset: ## Drop, recreate and re-migrate (wipes data)
	@../scripts/initdb.sh --reset

run: ## Run the API on :8080
	@go run ./cmd/api

build: ## Compile to bin/api
	@go build -o bin/api ./cmd/api
	@echo "built bin/api"

test: ## Unit tests — no database required
	@go test ./...

test-integration: ## Tests against the TEST database (needs `make db-test` first)
	@DATABASE_URL="$(TEST_DB_URL)" go test -tags=integration -count=1 ./...

watch: ## Re-run tests on save (needs: go install github.com/mitranim/gow@latest)
	@command -v gow >/dev/null 2>&1 \
		|| (echo "gow not installed. Run: go install github.com/mitranim/gow@latest"; exit 1)
	@gow -c test ./...

coverage: ## Coverage report
	@go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1
	@echo "HTML report: go tool cover -html=coverage.out"

lint: ## go vet and gofmt check
	@go vet ./...
	@test -z "$$(gofmt -l .)" || (echo "needs gofmt:"; gofmt -l .; exit 1)
	@echo "lint ok"

fmt: ## Format all Go files
	@gofmt -w .

tidy: ## Tidy go.mod
	@go mod tidy

psql: ## psql shell on the dev database
	@psql "$(DB_URL)"
