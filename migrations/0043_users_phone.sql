-- +goose Up
ALTER TABLE users ADD COLUMN IF NOT EXISTS phone VARCHAR(20) UNIQUE;

CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone);

-- +goose Down
DROP INDEX IF EXISTS idx_users_phone;
ALTER TABLE users DROP COLUMN IF EXISTS phone;
