-- +goose Up

ALTER TABLE auction_players
    DROP COLUMN IF EXISTS order_index;

-- +goose Down

ALTER TABLE auction_players
    ADD COLUMN IF NOT EXISTS order_index INT NOT NULL DEFAULT 0;
