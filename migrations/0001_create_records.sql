-- 0001_create_records.sql
-- The single resource this template exposes. Deliberately generic: a name, a
-- description and a flag, which is enough to demonstrate every verb.

CREATE TABLE IF NOT EXISTS records (
    id          BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name        TEXT        NOT NULL CHECK (length(trim(name)) > 0),
    description TEXT        NOT NULL DEFAULT '',
    active      BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Partial index for the common "only the active ones" filter.
CREATE INDEX IF NOT EXISTS records_active_idx ON records (id) WHERE active;
