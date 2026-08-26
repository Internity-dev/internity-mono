CREATE TABLE news (
    id              BIGSERIAL PRIMARY KEY,
    author_id       UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    scope_type      VARCHAR(10) NOT NULL CHECK (scope_type IN ('school', 'department')),
    scope_id        BIGINT NOT NULL,
    title           VARCHAR(255) NOT NULL,
    slug            VARCHAR(255) NOT NULL UNIQUE,
    content         TEXT NOT NULL,
    image_key       VARCHAR(500),
    status          VARCHAR(10) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_news_scope_status_published ON news (scope_type, scope_id, status, published_at DESC);

CREATE TRIGGER trg_news_set_updated_at
    BEFORE UPDATE ON news
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
