-- +goose Up
ALTER TABLE auction_players
    ADD COLUMN IF NOT EXISTS notes JSONB NOT NULL DEFAULT '{}'::jsonb;

-- +goose Down
ALTER TABLE auction_players
    DROP COLUMN IF EXISTS notes;
