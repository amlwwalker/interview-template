# Root Makefile — every command you need, in one place.
#
# Run `make` with no arguments to see them all. CI calls these same targets,
# so what you run locally is exactly what runs on GitHub.
#
# There are two interchangeable backends behind one schema and one frontend:
#   backend/     Go + chi
#   backend-ts/  TypeScript + Fastify
# Both serve the same API on :8080. Run one or the other.

.DEFAULT_GOAL := help
.PHONY: help setup setup-go setup-ts \
        test test-backend test-backend-ts test-frontend \
        test-integration test-integration-ts test-all \
        watch-backend watch-backend-ts watch-frontend coverage \
        lint lint-go lint-ts fmt build ci ci-ts \
        db db-test db-reset psql dev-backend dev-backend-ts dev-frontend clean

GO_BACKEND := backend
TS_BACKEND := backend-ts
FRONTEND   := frontend

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
	@printf "  $(C)make setup$(R)                everything: both backends, frontend, database\n"
	@printf "  $(C)make db$(R)                   create the dev database and run migrations\n"
	@printf "  $(C)make db-test$(R)              create the separate test database\n"
	@printf "  $(C)make db-reset$(R)             drop and recreate the dev database (wipes data)\n"
	@printf "\n$(B)Tests$(R)  $(D)(go + ts + frontend)$(R)\n"
	@printf "  $(C)make test$(R)                 all unit tests  $(B)<- the usual one$(R)\n"
	@printf "  $(C)make test-backend$(R)         Go only\n"
	@printf "  $(C)make test-backend-ts$(R)      TypeScript API only\n"
	@printf "  $(C)make test-frontend$(R)        Vitest (frontend) only\n"
	@printf "  $(C)make test-integration$(R)     Go against the test database\n"
	@printf "  $(C)make test-integration-ts$(R)  TypeScript against the test database\n"
	@printf "  $(C)make test-all$(R)             everything, including integration\n"
	@printf "  $(C)make watch-frontend$(R)       re-run on save\n"
	@printf "  $(C)make watch-backend$(R) / $(C)watch-backend-ts$(R)\n"
	@printf "  $(C)make coverage$(R)             coverage reports\n"
	@printf "\n$(B)Quality$(R)\n"
	@printf "  $(C)make lint$(R)                 vet, gofmt, tsc --noEmit (both backends + frontend)\n"
	@printf "  $(C)make build$(R)                compile everything\n"
	@printf "  $(C)make ci$(R)                   exactly what CI runs (Go stack)\n"
	@printf "  $(C)make ci-ts$(R)                exactly what CI runs (TypeScript stack)\n"
	@printf "\n$(B)Run$(R)  $(D)(pick ONE backend — both use :8080)$(R)\n"
	@printf "  $(C)make dev-backend$(R)          Go API on :8080\n"
	@printf "  $(C)make dev-backend-ts$(R)       TypeScript API on :8080\n"
	@printf "  $(C)make dev-frontend$(R)         console on :5173\n"
	@printf "  $(C)make psql$(R)                 psql shell on the dev database\n"

# --- setup -------------------------------------------------------------------

setup: setup-go setup-ts ## Install everything and prepare the database
	@cd $(FRONTEND) && npm install
	@$(MAKE) db
	@$(MAKE) db-test
	@printf "\nReady. Start one backend and the frontend in two terminals:\n"
	@printf "  make dev-backend      (or: make dev-backend-ts)\n"
	@printf "  make dev-frontend\n"

setup-go:
	@$(MAKE) -C $(GO_BACKEND) setup

setup-ts:
	@$(MAKE) -C $(TS_BACKEND) setup

db: ## Create the dev database and run migrations
	@./scripts/initdb.sh

db-test: ## Create the TEST database (integration tests truncate, so it is separate)
	@DB_NAME=crud_test ./scripts/initdb.sh --no-env

db-reset: ## Drop, recreate and re-migrate the dev database (wipes data)
	@./scripts/initdb.sh --reset

psql: ## psql shell on the dev database
	@$(MAKE) -C $(GO_BACKEND) psql

# --- tests -------------------------------------------------------------------

test: test-backend test-backend-ts test-frontend ## All unit tests

test-backend: ## Go tests (no database needed)
	@printf "$(B)backend (go)$(R)\n"
	@$(MAKE) -C $(GO_BACKEND) test

test-backend-ts: ## TypeScript API tests (no database needed)
	@printf "$(B)backend-ts$(R)\n"
	@$(MAKE) -C $(TS_BACKEND) test

test-frontend: ## Vitest (frontend)
	@printf "$(B)frontend$(R)\n"
	@cd $(FRONTEND) && npm test

test-integration: ## Go tests against a real Postgres
	@printf "$(B)backend (go) integration$(R)\n"
	@$(MAKE) -C $(GO_BACKEND) test-integration

test-integration-ts: ## TypeScript tests against a real Postgres
	@printf "$(B)backend-ts integration$(R)\n"
	@$(MAKE) -C $(TS_BACKEND) test-integration

test-all: test test-integration test-integration-ts ## Everything

watch-backend: ## Re-run Go tests on save
	@$(MAKE) -C $(GO_BACKEND) watch

watch-backend-ts: ## Re-run TypeScript API tests on save
	@$(MAKE) -C $(TS_BACKEND) watch

watch-frontend: ## Re-run frontend tests on save
	@cd $(FRONTEND) && npm run test:watch

coverage: ## Coverage for every suite
	@$(MAKE) -C $(GO_BACKEND) coverage
	@$(MAKE) -C $(TS_BACKEND) coverage
	@cd $(FRONTEND) && npm run test:coverage

# --- quality -----------------------------------------------------------------

lint: lint-go lint-ts ## Lint everything
	@cd $(FRONTEND) && npm run typecheck

lint-go:
	@$(MAKE) -C $(GO_BACKEND) lint

lint-ts:
	@$(MAKE) -C $(TS_BACKEND) lint

fmt: ## Format Go sources
	@$(MAKE) -C $(GO_BACKEND) fmt

build: ## Compile both backends and the frontend bundle
	@$(MAKE) -C $(GO_BACKEND) build
	@$(MAKE) -C $(TS_BACKEND) build
	@cd $(FRONTEND) && npm run build

# What CI runs, as one target per stack, so local and CI cannot drift apart.
ci: lint-go test-backend test-frontend ## Exactly what CI runs (Go stack)
	@$(MAKE) -C $(GO_BACKEND) build

ci-ts: lint-ts test-backend-ts ## Exactly what CI runs (TypeScript stack)
	@$(MAKE) -C $(TS_BACKEND) build

# --- run ---------------------------------------------------------------------

dev-backend: ## Go API on :8080
	@$(MAKE) -C $(GO_BACKEND) run

dev-backend-ts: ## TypeScript API on :8080
	@$(MAKE) -C $(TS_BACKEND) run

dev-frontend: ## Console on :5173
	@cd $(FRONTEND) && npm run dev

clean: ## Remove build artefacts
	@rm -rf $(GO_BACKEND)/bin $(GO_BACKEND)/coverage.out \
	        $(TS_BACKEND)/dist $(TS_BACKEND)/coverage \
	        $(FRONTEND)/dist $(FRONTEND)/coverage
	@printf "cleaned\n"
