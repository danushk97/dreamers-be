-- +goose Up

ALTER TABLE IF EXISTS tournament_player_registrations
    DROP COLUMN IF EXISTS player_info;

-- +goose Down

ALTER TABLE IF EXISTS tournament_player_registrations
    ADD COLUMN IF NOT EXISTS player_info JSONB NOT NULL DEFAULT '{}'::jsonb;

