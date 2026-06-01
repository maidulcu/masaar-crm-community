-- +goose Up
-- Mark companies as demo accounts. Demo companies are read-only:
-- write operations (POST/PATCH/DELETE) are blocked by middleware except auth routes.
ALTER TABLE companies
    ADD COLUMN IF NOT EXISTS is_demo BOOLEAN NOT NULL DEFAULT FALSE;

-- The seeded demo company gets flagged at seed time, not here, so existing
-- production data is untouched (all rows default to FALSE).

-- +goose Down
ALTER TABLE companies DROP COLUMN IF EXISTS is_demo;
