-- On-the-fly, not pre-materialized: a row exists only once the student
-- actually checks in or files an excuse — there is no daily cron/job that
-- bulk-inserts a blank row per day of the internship range (see plan
-- section 2.5 for why: legacy's write-amplification job, its deletion-on-
-- shrink edge cases, and its "Pending" placeholder status all go away for
-- free once "no row" simply means "not yet reported" instead of a fake event).
CREATE TABLE presences (
    id                  BIGSERIAL PRIMARY KEY,
    user_id             UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    company_id          BIGINT NOT NULL REFERENCES companies (id) ON DELETE RESTRICT,
    presence_status_id  BIGINT NOT NULL REFERENCES presence_statuses (id) ON DELETE RESTRICT,
    date                DATE NOT NULL,
    check_in_at         TIMESTAMPTZ,
    check_out_at        TIMESTAMPTZ,
    -- Legacy captured a check-in photo but never location — added here as a
    -- deliberate value-add: a photo alone doesn't prove physical presence at
    -- the workplace, lat/lng closes that gap for a two-field cost.
    check_in_lat        DOUBLE PRECISION,
    check_in_lng        DOUBLE PRECISION,
    attachment_key      VARCHAR(500),
    is_approved         BOOLEAN NOT NULL DEFAULT false,
    description         TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, company_id, date)
);

CREATE INDEX idx_presences_company_approved ON presences (company_id, is_approved);

CREATE TRIGGER trg_presences_set_updated_at
    BEFORE UPDATE ON presences
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
