CREATE TABLE scores (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    company_id      BIGINT NOT NULL REFERENCES companies (id) ON DELETE RESTRICT,
    name            VARCHAR(255) NOT NULL,
    score           INT NOT NULL CHECK (score BETWEEN 0 AND 100),
    type            VARCHAR(20) NOT NULL CHECK (type IN ('teknis', 'non-teknis')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_scores_user_company ON scores (user_id, company_id);

CREATE TRIGGER trg_scores_set_updated_at
    BEFORE UPDATE ON scores
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
