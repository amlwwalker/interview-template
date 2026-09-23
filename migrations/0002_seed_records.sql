-- 0002_seed_records.sql
-- Three rows so READ is not empty on first load. Only fires when the table is
-- empty, so it is safe to leave in place.

INSERT INTO records (name, description, active)
SELECT * FROM (VALUES
    ('alpha', 'first seeded row',  TRUE),
    ('beta',  'second seeded row', FALSE),
    ('gamma', 'third seeded row',  TRUE)
) AS seed(name, description, active)
WHERE NOT EXISTS (SELECT 1 FROM records);
