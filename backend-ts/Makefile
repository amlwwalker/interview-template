# TypeScript backend commands. The root Makefile delegates here; this is also
# safe to call directly. `make` with no arguments lists everything.

.DEFAULT_GOAL := help
.PHONY: help setup run build start test test-integration watch coverage lint fmt

# The dev database, for `make psql` and for running the app.
DB_URL ?= postgres://$(shell whoami)@localhost:5432/crud_dev?sslmode=disable
# Integration tests TRUNCATE before every test, so they get their own database.
# Pointing them at crud_dev silently wipes your seed data every time you run
# the suite, and then the app looks broken for no visible reason.
TEST_DB_URL ?= postgres://$(shell whoami)@localhost:5432/crud_test?sslmode=disable

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-17s\033[0m %s\n", $$1, $$2}'

setup: ## Install dependencies
	@npm install

run: ## Run the API on :8080 with reload
	@npm run dev

build: ## Compile to dist/
	@npm run build

start: ## Run the compiled build
	@npm start

test: ## Unit tests — no database required
	@npm test

test-integration: ## Tests against the TEST database (needs `make db-test` first)
	@DATABASE_URL="$(TEST_DB_URL)" npm run test:integration

watch: ## Re-run tests on save
	@npm run test:watch

coverage: ## Coverage report
	@npm run test:coverage

lint: ## Typecheck
	@npm run typecheck
	@echo "lint ok"
