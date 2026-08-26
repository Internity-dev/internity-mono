-- Backs the admin dashboard's attendance-breakdown chart, which filters by
-- a date range only (no company_id) — the existing composite indexes on
-- this table both lead with user_id/company_id, so a bare date range scan
-- couldn't use either of them.
CREATE INDEX idx_presences_date ON presences (date);
