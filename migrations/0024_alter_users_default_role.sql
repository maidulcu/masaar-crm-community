-- +goose Up
-- Least-privilege default: new users start as viewer until explicitly promoted.
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'viewer';

-- +goose Down
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'agent';
