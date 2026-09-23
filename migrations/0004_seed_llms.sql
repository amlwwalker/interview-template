-- 0004_seed_llms.sql
-- The mock LLMs available during development. No real provider exists yet;
-- these are what the routing layer is built and tested against.
--
-- Four of the five have a registered implementation. `ghost-mock` deliberately
-- does not: it is how the "configured but not runnable" path gets exercised
-- against the real database rather than only against a hand-built registry in
-- a unit test. Do not "fix" it by adding a provider — it is load-bearing.

INSERT INTO llms (slug, name, provider_key, model, enabled, sort_order) VALUES
    ('echo-mock',   'Echo (mock)',          'echo',        'echo-v1',   TRUE, 10),
    ('script-mock', 'Scripted (mock)',      'script',      'script-v1', TRUE, 20),
    ('error-mock',  'Always fails (mock)',  'error',       'error-v1',  TRUE, 30),
    ('slow-mock',   'Never responds (mock)','slow',        'slow-v1',   TRUE, 40),
    ('ghost-mock',  'Unregistered (mock)',  'nonexistent', 'ghost-v1',  TRUE, 50)
ON CONFLICT (slug) DO NOTHING;
