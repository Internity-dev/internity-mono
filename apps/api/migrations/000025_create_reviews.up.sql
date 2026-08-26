-- Flattened from legacy's dual-nullable-FK-plus-morph into one explicit
-- pair of nullable FKs with a CHECK ensuring exactly one is set — a review
-- targets either a student (mentor rating a student) or a company (an
-- aggregate company rating), never neither/both.
CREATE TABLE reviews (
    id                  BIGSERIAL PRIMARY KEY,
    reviewer_id         UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    question_id         BIGINT REFERENCES questions (id) ON DELETE SET NULL,
    reviewee_user_id    UUID REFERENCES users (id) ON DELETE CASCADE,
    reviewee_company_id BIGINT REFERENCES companies (id) ON DELETE CASCADE,
    title               VARCHAR(255),
    body                TEXT,
    rating              INT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_reviews_exactly_one_reviewee CHECK (
        (reviewee_user_id IS NOT NULL)::int + (reviewee_company_id IS NOT NULL)::int = 1
    )
);

CREATE INDEX idx_reviews_reviewee_user ON reviews (reviewee_user_id);
CREATE INDEX idx_reviews_reviewee_company ON reviews (reviewee_company_id);

CREATE TRIGGER trg_reviews_set_updated_at
    BEFORE UPDATE ON reviews
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
