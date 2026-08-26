CREATE TABLE vacancies (
    id              BIGSERIAL PRIMARY KEY,
    company_id      BIGINT NOT NULL REFERENCES companies (id) ON DELETE RESTRICT,
    name            VARCHAR(255) NOT NULL,
    category        VARCHAR(255),
    description     TEXT,
    skills          TEXT,
    slots           INT NOT NULL DEFAULT 1 CHECK (slots >= 1),
    status          VARCHAR(10) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_vacancies_company_status ON vacancies (company_id, status);
-- Trigram index backs the "recommended vacancies" skill-match search (name +
-- skills) without needing a dedicated search-engine dependency at this scale.
CREATE INDEX idx_vacancies_name_trgm ON vacancies USING gin (name gin_trgm_ops);
CREATE INDEX idx_vacancies_skills_trgm ON vacancies USING gin (skills gin_trgm_ops);
