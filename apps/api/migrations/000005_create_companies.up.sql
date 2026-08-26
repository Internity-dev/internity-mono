CREATE TABLE companies (
    id              BIGSERIAL PRIMARY KEY,
    department_id   BIGINT NOT NULL REFERENCES departments (id) ON DELETE RESTRICT,
    name            VARCHAR(255) NOT NULL,
    category        VARCHAR(255),
    city            VARCHAR(255),
    state           VARCHAR(255),
    country         VARCHAR(255),
    address         TEXT,
    email           VARCHAR(255),
    phone           VARCHAR(50),
    website         VARCHAR(255),
    logo_key        VARCHAR(500),
    contact_person  VARCHAR(255),
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_companies_department_id ON companies (department_id);
CREATE INDEX idx_companies_is_active ON companies (is_active);
