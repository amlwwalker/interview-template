-- 0003_create_llms.sql
-- The catalogue of LLMs a user may route a conversation to.
--
-- A row here is *configuration*: it says an LLM is on offer and names the
-- implementation that should serve it. It does not say that implementation
-- exists in this build. That question is answered by the registry in Go, at
-- call time, so one stale row cannot stop the service from starting.

CREATE TABLE IF NOT EXISTS llms (
    id           BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    -- The stable, public selector. This is what a client sends to choose an
    -- LLM, and what a dropdown uses as its value. Never shown as a label.
    slug         TEXT        NOT NULL UNIQUE
                             CHECK (length(trim(slug)) > 0),

    -- Display only. Safe to reword at any time; no client selects by it.
    name         TEXT        NOT NULL
                             CHECK (length(trim(name)) > 0),

    -- Binds this row to a Provider registered in code. Several rows may share
    -- one provider_key while differing in model — which is how a single real
    -- provider will later serve several models.
    provider_key TEXT        NOT NULL
                             CHECK (length(trim(provider_key)) > 0),

    -- Passed through to the provider as Request.Model.
    model        TEXT        NOT NULL
                             CHECK (length(trim(model)) > 0),

    -- Retire an LLM without deleting it, so existing references still resolve
    -- for audit while it stops being offered.
    enabled      BOOLEAN     NOT NULL DEFAULT TRUE,

    -- Deterministic ordering for the picker. Ties break on id.
    sort_order   INTEGER     NOT NULL DEFAULT 0,

    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- The list endpoint reads enabled rows in display order and nothing else.
CREATE INDEX IF NOT EXISTS llms_enabled_order_idx
    ON llms (sort_order, id) WHERE enabled;
