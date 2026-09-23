# Root Makefile — every command you need, in one place.
#
# Run `make` with no arguments to see them all. CI calls these same targets,
# so what you run locally is exactly what runs on GitHub.

.DEFAULT_GOAL := help
.PHONY: help setup test test-backend test-frontend \
        test-integration test-all watch-backend watch-frontend coverage \
        lint fmt build ci \
        db db-test db-reset psql dev-backend dev-frontend clean

BACKEND  := backend
FRONTEND := frontend

ifneq (,$(findstring xterm,$(TERM)))
  C := \033[36m
  B := \033[1m
  D := \033[2m
  R := \033[0m
else
  C :=
  B :=
  D :=
  R :=
endif

help: ## Show this help
	@printf "$(B)Setup$(R)\n"
	@printf "  $(C)make setup$(R)                backend, frontend and both databases\n"
	@printf "  $(C)make db$(R)                   create the dev database and run migrations\n"
	@printf "  $(C)make db-test$(R)              create the separate test database\n"
	@printf "  $(C)make db-reset$(R)             drop and recreate the dev database (wipes data)\n"
	@printf "\n$(B)Tests$(R)\n"
	@printf "  $(C)make test$(R)                 backend + frontend unit tests  $(B)<- the usual one$(R)\n"
	@printf "  $(C)make test-backend$(R)         Go only, no database\n"
	@printf "  $(C)make test-frontend$(R)        frontend only\n"
	@printf "  $(C)make test-integration$(R)     Go against the test database\n"
	@printf "  $(C)make test-all$(R)             everything, including integration\n"
	@printf "  $(C)make watch-backend$(R) / $(C)make watch-frontend$(R)   re-run on save\n"
	@printf "  $(C)make coverage$(R)             coverage reports\n"
	@printf "\n$(B)Quality$(R)\n"
	@printf "  $(C)make lint$(R)                 go vet, gofmt, tsc --noEmit\n"
	@printf "  $(C)make build$(R)                compile the binary and the bundle\n"
	@printf "  $(C)make ci$(R)                   exactly what CI runs\n"
	@printf "\n$(B)Run$(R)\n"
	@printf "  $(C)make dev-backend$(R)          API on :8080\n"
	@printf "  $(C)make dev-frontend$(R)         console on :5173\n"
	@printf "  $(C)make psql$(R)                 psql shell on the dev database\n"

# --- setup -------------------------------------------------------------------

setup: ## Install everything and prepare the databases
	@$(MAKE) -C $(BACKEND) setup
	@cd $(FRONTEND) && npm install
	@$(MAKE) db
	@$(MAKE) db-test
	@printf "\nReady. In two terminals:\n"
	@printf "  make dev-backend\n"
	@printf "  make dev-frontend\n

db: ## Create the dev database and run migrations
	@./scripts/initdb.sh

db-test: ## Create the TEST database (integration tests truncate, so it is separate)
	@DB_NAME=crud_test ./scripts/initdb.sh --no-env

db-reset: ## Drop, recreate and re-migrate the dev database (wipes data)
	@./scripts/initdb.sh --reset

psql: ## psql shell on the dev database
	@$(MAKE) -C $(BACKEND) psql

# --- tests -------------------------------------------------------------------

test: test-backend test-frontend ## Backend + frontend unit tests

test-backend: ## Go tests (no database needed)
	@printf "$(B)backend (go)$(R)\n"
	@$(MAKE) -C $(BACKEND) test

test-frontend: ## Vitest (frontend)
	@printf "$(B)frontend$(R)\n"
	@cd $(FRONTEND) && npm test

test-integration: ## Go tests against a real Postgres
	@printf "$(B)backend (go) integration$(R)\n"
	@$(MAKE) -C $(BACKEND) test-integration

test-all: test test-integration ## Everything

watch-backend: ## Re-run Go tests on save
	@$(MAKE) -C $(BACKEND) watch

watch-frontend: ## Re-run frontend tests on save
	@cd $(FRONTEND) && npm run test:watch

coverage: ## Coverage for every suite
	@$(MAKE) -C $(BACKEND) coverage
	@cd $(FRONTEND) && npm run test:coverage

# --- quality -----------------------------------------------------------------

lint: ## go vet, gofmt, tsc --noEmit
	@$(MAKE) -C $(BACKEND) lint
	@cd $(FRONTEND) && npm run typecheck

fmt: ## Format Go sources
	@$(MAKE) -C $(BACKEND) fmt

build: ## Compile both backends and the frontend bundle
	@$(MAKE) -C $(BACKEND) build
	@cd $(FRONTEND) && npm run build

# What CI runs, as one target per stack, so local and CI cannot drift apart.
ci: lint test-backend test-frontend ## Exactly what CI runs
	@$(MAKE) -C $(BACKEND) build

# --- run ---------------------------------------------------------------------

dev-backend: ## Go API on :8080
	@$(MAKE) -C $(BACKEND) run

dev-frontend: ## Console on :5173
	@cd $(FRONTEND) && npm run dev

clean: ## Remove build artefacts
	@rm -rf $(BACKEND)/bin $(BACKEND)/coverage.out \
	        $(FRONTEND)/dist $(FRONTEND)/coverage
	@printf "cleaned\n"
