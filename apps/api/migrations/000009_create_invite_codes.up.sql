-- Replaces legacy's polymorphic `codes` table (codeable = School|Course). In
-- practice only student self-registration ever consumed a code, always
-- resolving down to a course (which already implies department + school via
-- FK chain) — so this is a direct course_id FK instead of a morph, not a
-- generalization nothing uses (see plan's "don't build unused flexibility").
CREATE TABLE invite_codes (
    id              BIGSERIAL PRIMARY KEY,
    code            VARCHAR(64) NOT NULL UNIQUE,
    course_id       BIGINT NOT NULL REFERENCES courses (id) ON DELETE CASCADE,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_invite_codes_course_id ON invite_codes (course_id);
