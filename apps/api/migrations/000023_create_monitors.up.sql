CREATE TABLE monitors (
    id              BIGSERIAL PRIMARY KEY,
    coordinator_id  UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    student_id      UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    company_id      BIGINT NOT NULL REFERENCES companies (id) ON DELETE RESTRICT,
    date            DATE NOT NULL,
    attachment_key  VARCHAR(500),
    notes           TEXT,
    suggest         TEXT,
    match_rating    INT NOT NULL CHECK (match_rating BETWEEN 1 AND 4),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_monitors_student_id ON monitors (student_id);
CREATE INDEX idx_monitors_company_id ON monitors (company_id);

CREATE TRIGGER trg_monitors_set_updated_at
    BEFORE UPDATE ON monitors
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
