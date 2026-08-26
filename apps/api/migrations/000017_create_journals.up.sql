CREATE TABLE journals (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    company_id      BIGINT NOT NULL REFERENCES companies (id) ON DELETE RESTRICT,
    date            DATE NOT NULL,
    work_type       VARCHAR(255),
    description     TEXT,
    is_approved     BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, company_id, date)
);

CREATE INDEX idx_journals_company_approved ON journals (company_id, is_approved);

CREATE TRIGGER trg_journals_set_updated_at
    BEFORE UPDATE ON journals
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
