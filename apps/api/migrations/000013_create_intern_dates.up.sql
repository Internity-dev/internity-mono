CREATE TABLE intern_dates (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    company_id      BIGINT NOT NULL REFERENCES companies (id) ON DELETE RESTRICT,
    appliance_id    BIGINT NOT NULL UNIQUE REFERENCES appliances (id) ON DELETE RESTRICT,
    start_date      DATE,
    end_date        DATE,
    extended_until  DATE,
    status          VARCHAR(10) NOT NULL DEFAULT 'scheduled'
                        CHECK (status IN ('scheduled', 'active', 'completed')),
    version         INT NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (user_id, company_id),
    CONSTRAINT chk_intern_dates_range CHECK (start_date IS NULL OR end_date IS NULL OR start_date < end_date),

    -- A student's placements may span multiple companies over time, but no
    -- two of their date ranges may overlap — this is the direct, DB-enforced
    -- fix for a rule legacy only checked ad hoc in application code
    -- (ApplianceController@editDate's manual overlap query). Only applies once
    -- both dates are set (a freshly-accepted, not-yet-scheduled row has neither).
    CONSTRAINT excl_intern_dates_no_overlap_per_user
        EXCLUDE USING gist (
            user_id WITH =,
            daterange(start_date, end_date, '[]') WITH &&
        ) WHERE (start_date IS NOT NULL AND end_date IS NOT NULL)
);

CREATE INDEX idx_intern_dates_company_id ON intern_dates (company_id);

CREATE TRIGGER trg_intern_dates_set_updated_at
    BEFORE UPDATE ON intern_dates
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
