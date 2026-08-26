CREATE TABLE appliances (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    vacancy_id      BIGINT NOT NULL REFERENCES vacancies (id) ON DELETE RESTRICT,
    status          VARCHAR(10) NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'processed', 'accepted', 'rejected', 'canceled')),
    message         TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_appliances_vacancy_status ON appliances (vacancy_id, status);
CREATE INDEX idx_appliances_user_id ON appliances (user_id);

-- One *active* application per (user, vacancy) — pending/processed/accepted
-- are mutually exclusive; a student can freely re-apply after a rejection or
-- their own cancellation (see plan section 2.4).
CREATE UNIQUE INDEX uq_appliances_active_per_user_vacancy
    ON appliances (user_id, vacancy_id)
    WHERE status IN ('pending', 'processed', 'accepted');

CREATE TRIGGER trg_appliances_set_updated_at
    BEFORE UPDATE ON appliances
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
