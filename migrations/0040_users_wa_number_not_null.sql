-- +goose Up
UPDATE users SET wa_number = '' WHERE wa_number IS NULL;
ALTER TABLE users ALTER COLUMN wa_number SET DEFAULT '';
ALTER TABLE users ALTER COLUMN wa_number SET NOT NULL;

-- +goose Down
ALTER TABLE users ALTER COLUMN wa_number DROP NOT NULL;
ALTER TABLE users ALTER COLUMN wa_number DROP DEFAULT;
