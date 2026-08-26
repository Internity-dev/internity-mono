-- Always one row per recipient (never a nullable-user_id "broadcast" row like
-- legacy) — broadcast fan-out happens at write time, in the service layer,
-- one insert per affected user, so per-user read_at tracking is always correct.
CREATE TABLE notifications (
    id              BIGSERIAL PRIMARY KEY,
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type            VARCHAR(50) NOT NULL,
    title           VARCHAR(255) NOT NULL,
    body            TEXT,
    read_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user_unread ON notifications (user_id, read_at);
