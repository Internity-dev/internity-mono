-- Letter-grade bands, per school. The predicate itself is never stored on a
-- score row — it's derived at read time by matching a score against these
-- min/max bands (kept from legacy as-is; it's a clean rule).
CREATE TABLE score_predicates (
    id              BIGSERIAL PRIMARY KEY,
    school_id       BIGINT NOT NULL REFERENCES schools (id) ON DELETE CASCADE,
    name            VARCHAR(50) NOT NULL,
    description     TEXT,
    color           VARCHAR(20),
    min             NUMERIC(5,2) NOT NULL,
    max             NUMERIC(5,2) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (school_id, name),
    CONSTRAINT chk_score_predicates_range CHECK (min <= max)
);

CREATE INDEX idx_score_predicates_school_id ON score_predicates (school_id);
