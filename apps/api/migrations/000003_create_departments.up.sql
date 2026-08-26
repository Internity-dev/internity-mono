CREATE TABLE departments (
    id              BIGSERIAL PRIMARY KEY,
    -- RESTRICT, not CASCADE: deleting a school that still has departments
    -- should fail loudly (409) rather than silently wipe the whole tree.
    school_id       BIGINT NOT NULL REFERENCES schools (id) ON DELETE RESTRICT,
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    study_program   VARCHAR(255),
    logo_key        VARCHAR(500),
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (school_id, name)
);

CREATE INDEX idx_departments_school_id ON departments (school_id);
