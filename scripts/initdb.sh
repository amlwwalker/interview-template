#!/usr/bin/env bash
#
# initdb.sh — create the local database and apply migrations.
#
# Targets an *existing* local Postgres (Homebrew or Postgres.app). Safe to run
# repeatedly: the database is only created if missing, and each migration is
# recorded so it runs exactly once.
#
#   ./scripts/initdb.sh              # create + migrate
#   ./scripts/initdb.sh --reset      # drop and recreate first
#   ./scripts/initdb.sh --no-env     # skip writing .env (used for the test database)
#
# DB_NAME overrides which database is created, which is how the test database
# is built from the same script.
#
set -euo pipefail

DB_NAME="${DB_NAME:-crud_dev}"
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-$(whoami)}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
MIGRATIONS_DIR="$ROOT_DIR/migrations"

# One schema, two backend implementations. Each reads the .env this writes.

RESET=false
WRITE_ENV=true
for arg in "$@"; do
  case "$arg" in
    --reset)  RESET=true ;;
    --no-env) WRITE_ENV=false ;;
    *) printf 'unknown argument: %s\n' "$arg" >&2; exit 1 ;;
  esac
done

bold() { printf '\033[1m%s\033[0m\n' "$1"; }
ok()   { printf '  \033[32m✓\033[0m %s\n' "$1"; }
warn() { printf '  \033[33m!\033[0m %s\n' "$1"; }
die()  { printf '  \033[31m✗\033[0m %s\n' "$1" >&2; exit 1; }

# --- 1. Locate psql ----------------------------------------------------------
# Homebrew keeps the versioned client off the default PATH, and Postgres.app
# hides it in the bundle, so look in the usual places before giving up.
if ! command -v psql >/dev/null 2>&1; then
  for candidate in \
    /opt/homebrew/opt/postgresql@17/bin \
    /opt/homebrew/opt/postgresql@16/bin \
    /opt/homebrew/opt/postgresql@15/bin \
    /usr/local/opt/postgresql@16/bin \
    /Applications/Postgres.app/Contents/Versions/latest/bin
  do
    if [[ -x "$candidate/psql" ]]; then
      export PATH="$candidate:$PATH"
      warn "using psql from $candidate (not on your PATH)"
      break
    fi
  done
fi

command -v psql >/dev/null 2>&1 || die \
  "psql not found. Install with: brew install postgresql@16 && brew services start postgresql@16"

# --- 2. Check the server is up ----------------------------------------------
if ! pg_isready -h "$DB_HOST" -p "$DB_PORT" -q 2>/dev/null; then
  die "no Postgres server responding on $DB_HOST:$DB_PORT.
    Start it with:  brew services start postgresql@16
    Or if you use Postgres.app, open the app and click Start."
fi
ok "Postgres is up on $DB_HOST:$DB_PORT"

# Connect to the maintenance database for the create/drop statements. `postgres`
# exists on Homebrew installs; Postgres.app creates one named after your user.
ADMIN_DB="postgres"
if ! psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$ADMIN_DB" -tAc 'SELECT 1' >/dev/null 2>&1; then
  ADMIN_DB="$DB_USER"
  psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$ADMIN_DB" -tAc 'SELECT 1' >/dev/null 2>&1 \
    || die "cannot connect as role '$DB_USER'. Create it with: createuser -s $DB_USER"
fi

psql_admin() { psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$ADMIN_DB" "$@"; }
psql_app()   { psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 "$@"; }

# --- 3. Create (or recreate) the database ------------------------------------
db_exists() {
  [[ "$(psql_admin -tAc "SELECT 1 FROM pg_database WHERE datname = '$DB_NAME'")" == "1" ]]
}

if $RESET && db_exists; then
  # Boot any lingering connections, or the DROP will block forever.
  psql_admin -q -c \
    "SELECT pg_terminate_backend(pid) FROM pg_stat_activity
     WHERE datname = '$DB_NAME' AND pid <> pg_backend_pid()" >/dev/null
  psql_admin -q -c "DROP DATABASE \"$DB_NAME\""
  warn "dropped existing database '$DB_NAME'"
fi

if db_exists; then
  ok "database '$DB_NAME' already exists"
else
  psql_admin -q -c "CREATE DATABASE \"$DB_NAME\" OWNER \"$DB_USER\""
  ok "created database '$DB_NAME'"
fi

# --- 4. Apply migrations ------------------------------------------------------
# A tracking table keeps this idempotent without pulling in a migration tool.
psql_app -q -c "
  CREATE TABLE IF NOT EXISTS schema_migrations (
    version    TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
  )"

applied_count=0
for migration in "$MIGRATIONS_DIR"/*.sql; do
  [[ -e "$migration" ]] || die "no migrations found in $MIGRATIONS_DIR"

  version="$(basename "$migration")"

  if [[ "$(psql_app -tAc "SELECT 1 FROM schema_migrations WHERE version = '$version'")" == "1" ]]; then
    continue
  fi

  # Run the migration and record it in one transaction: a failure halfway
  # through leaves nothing behind, so a re-run starts clean.
  psql_app -q --single-transaction \
    -f "$migration" \
    -c "INSERT INTO schema_migrations (version) VALUES ('$version')" \
    || die "migration failed: $version"

  ok "applied $version"
  applied_count=$((applied_count + 1))
done

[[ $applied_count -eq 0 ]] && ok "schema already up to date"

# --- 5. Write .env ------------------------------------------------------------
DATABASE_URL="postgres://${DB_USER}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"
ENV_FILE="$ROOT_DIR/.env"

write_env() {
  local target="$1"
  [[ -d "$(dirname "$target")" ]] || return 0
  if [[ -f "$target" ]]; then
    warn "$(basename "$(dirname "$target")")/.env already exists, leaving it alone"
    return 0
  fi
  cat > "$target" <<EOF
DATABASE_URL=${DATABASE_URL}
PORT=8080
CORS_ALLOWED_ORIGINS=http://localhost:5173,http://127.0.0.1:5173
EOF
  ok "wrote $(basename "$(dirname "$target")")/.env"
}

# Skipped for the test database, which must never become the one the app
# points at.
if $WRITE_ENV; then
  write_env "$ROOT_DIR/backend/.env"
fi

echo
bold "Ready."
echo "  DATABASE_URL=${DATABASE_URL}"
echo "  Next:  make run"
