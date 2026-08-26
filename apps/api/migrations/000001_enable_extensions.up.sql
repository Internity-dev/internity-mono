-- pgcrypto: gen_random_uuid() for user-facing PKs (avoids legacy's encrypted-ID-in-URL
-- hack — an opaque UUID needs no encryption trick and still isn't enumerable).
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- pg_trgm: trigram index support for the vacancy name/skills text search (section 3.3
-- of the plan) — no external search engine dependency needed at this scale.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- btree_gist: required for the EXCLUDE constraint that prevents a student's
-- intern_date ranges from overlapping across companies (added in a later migration).
CREATE EXTENSION IF NOT EXISTS btree_gist;
