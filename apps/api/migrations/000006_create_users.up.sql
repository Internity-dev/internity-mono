-- Plain VARCHAR + CHECK instead of a native Postgres ENUM: gives the same
-- validation guarantee, is simpler to extend later (ALTER TABLE ... DROP/ADD
-- CONSTRAINT vs ALTER TYPE's own transactional quirks), and avoids the
-- well-known GORM/pgx friction with native enum parameter binding.
--
-- UUID PKs for users specifically: this table is what legacy protected with an
-- `encrypt($id)`-in-URL hack (see plan section 0 / ADR). An opaque, non-enumerable
-- UUID removes the need for that trick while every authorization check still
-- happens server-side via role+scope (never "security through obscurity").
CREATE TABLE users (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role                VARCHAR(20) NOT NULL CHECK (role IN ('admin', 'coordinator', 'mentor', 'student')),

    -- Scope FKs: exactly one combination is valid per role (enforced below).
    -- This replaces legacy's school_user/department_user/company_user pivot
    -- tables queried with a fragile `->first()` — see plan section 2.2.
    school_id           BIGINT REFERENCES schools (id) ON DELETE RESTRICT,
    department_id       BIGINT REFERENCES departments (id) ON DELETE RESTRICT,
    company_id          BIGINT REFERENCES companies (id) ON DELETE RESTRICT,
    course_id           BIGINT REFERENCES courses (id) ON DELETE RESTRICT,

    name                VARCHAR(255) NOT NULL,
    email               VARCHAR(255) NOT NULL UNIQUE,
    password_hash       VARCHAR(255) NOT NULL,
    nis                 VARCHAR(50) UNIQUE,
    gender              VARCHAR(10) CHECK (gender IN ('male', 'female')),
    bio                 TEXT,
    address             TEXT,
    phone               VARCHAR(50),
    date_of_birth       DATE,
    avatar_key          VARCHAR(500),
    resume_key          VARCHAR(500),
    skills              TEXT,
    is_active           BOOLEAN NOT NULL DEFAULT true,
    last_login_at       TIMESTAMPTZ,
    last_login_ip       VARCHAR(64),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_users_scope_matches_role CHECK (
        CASE role
            WHEN 'admin' THEN
                school_id IS NULL AND department_id IS NULL AND company_id IS NULL AND course_id IS NULL
            WHEN 'coordinator' THEN
                school_id IS NOT NULL AND company_id IS NULL AND course_id IS NULL
            WHEN 'mentor' THEN
                company_id IS NOT NULL AND school_id IS NULL AND department_id IS NULL AND course_id IS NULL
            WHEN 'student' THEN
                school_id IS NOT NULL AND department_id IS NOT NULL AND course_id IS NOT NULL AND company_id IS NULL
        END
    )
);

CREATE INDEX idx_users_role_school ON users (role, school_id);
CREATE INDEX idx_users_role_department ON users (role, department_id);
CREATE INDEX idx_users_role_company ON users (role, company_id);
CREATE INDEX idx_users_role_course ON users (role, course_id);

CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_users_set_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
