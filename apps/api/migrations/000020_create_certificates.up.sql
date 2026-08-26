CREATE TABLE certificates (
    id                  BIGSERIAL PRIMARY KEY,
    user_id             UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    department_id       BIGINT NOT NULL REFERENCES departments (id) ON DELETE RESTRICT,
    company_id          BIGINT NOT NULL REFERENCES companies (id) ON DELETE RESTRICT,
    certificate_number  VARCHAR(100) NOT NULL UNIQUE,
    file_key            VARCHAR(500),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, company_id)
);
