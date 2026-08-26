-- Per-school configurable statuses (e.g. Hadir/Alpa/Izin/Sakit/Libur) — the
-- display name/color/icon are freely configurable per school like legacy,
-- but `kind` is a fixed, code-meaningful classification the application
-- resolves against (e.g. "give me this school's `present` status for
-- check-in"). This directly replaces legacy's fragile string-matching
-- against presence_statuses.name ("Pending") found during the system audit.
CREATE TABLE presence_statuses (
    id              BIGSERIAL PRIMARY KEY,
    school_id       BIGINT NOT NULL REFERENCES schools (id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    kind            VARCHAR(20) NOT NULL CHECK (kind IN ('present', 'permitted', 'sick', 'absent', 'holiday')),
    description     TEXT,
    color           VARCHAR(20),
    icon            VARCHAR(50),
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (school_id, name),
    -- Exactly one status per (school, kind) for the four kinds the
    -- application resolves programmatically — a school could otherwise
    -- define two "present" statuses and make check-in's lookup ambiguous.
    UNIQUE (school_id, kind)
);

CREATE INDEX idx_presence_statuses_school_id ON presence_statuses (school_id);
