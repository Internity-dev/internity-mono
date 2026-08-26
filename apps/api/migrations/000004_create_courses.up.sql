-- "courses" = kelas/class-cohort (e.g. "XII RPL 1"), not an academic course —
-- naming kept from the legacy domain since it's what the invite-code /
-- registration flow (Phase 1) keys off.
CREATE TABLE courses (
    id              BIGSERIAL PRIMARY KEY,
    department_id   BIGINT NOT NULL REFERENCES departments (id) ON DELETE RESTRICT,
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (department_id, name)
);

CREATE INDEX idx_courses_department_id ON courses (department_id);
