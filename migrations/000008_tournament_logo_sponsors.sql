-- +goose Up
ALTER TABLE tournaments ADD COLUMN IF NOT EXISTS logo_url VARCHAR(2048) NOT NULL DEFAULT '';
ALTER TABLE tournaments ADD COLUMN IF NOT EXISTS sponsors JSONB NOT NULL DEFAULT '[]'::jsonb;

-- +goose Down
ALTER TABLE tournaments DROP COLUMN IF EXISTS sponsors;
ALTER TABLE tournaments DROP COLUMN IF EXISTS logo_url;
