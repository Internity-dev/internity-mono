-- Opaque cookie-backed sessions (never JWT — see plan section 2/ADR "cookie-session
-- vs JWT"). One row per access OR refresh token. `family_id` ties an access+refresh
-- pair (and every rotation descendant) together so reuse of an already-rotated
-- refresh token can revoke the whole family (theft detection).
CREATE TABLE sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash      VARCHAR(64) NOT NULL UNIQUE, -- sha256 hex of the raw cookie value; raw value never stored
    kind            VARCHAR(10) NOT NULL CHECK (kind IN ('access', 'refresh')),
    family_id       UUID NOT NULL,
    user_agent      TEXT,
    ip              VARCHAR(64),
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_sessions_user_id ON sessions (user_id, revoked_at);
CREATE INDEX idx_sessions_family_id ON sessions (family_id);
