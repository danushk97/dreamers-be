-- +goose Up

-- Core tournament/event catalog (CRUD/backfill friendly)
CREATE TABLE IF NOT EXISTS tournaments (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    sport_id VARCHAR(100) NOT NULL,
    start_date BIGINT NOT NULL,
    end_date BIGINT NOT NULL,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT
);

CREATE TABLE IF NOT EXISTS events (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    attrs JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT
);

-- Tournament ↔ event mapping (supports parent/child event trees)
CREATE TABLE IF NOT EXISTS tournament_events (
    id VARCHAR(36) PRIMARY KEY,
    tournament_id VARCHAR(36) NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    event_id VARCHAR(36) NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    parent_event_id VARCHAR(36),
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT
);

CREATE INDEX IF NOT EXISTS idx_tournament_events_tournament_id ON tournament_events(tournament_id);
CREATE INDEX IF NOT EXISTS idx_tournament_events_event_id ON tournament_events(event_id);

-- Teams registered for a tournament + event (this is what wallet/bids reference)
CREATE TABLE IF NOT EXISTS tournament_team_registrations (
    id VARCHAR(36) PRIMARY KEY,
    tournament_id VARCHAR(36) NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    event_id VARCHAR(36) NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    team_name VARCHAR(255) NOT NULL,
    team_logo_url VARCHAR(2048) NOT NULL DEFAULT '',
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT
);

CREATE INDEX IF NOT EXISTS idx_team_regs_tournament_event ON tournament_team_registrations(tournament_id, event_id);

-- Player registrations for a tournament/event (source pool for auction lots)
CREATE TABLE IF NOT EXISTS tournament_player_registrations (
    id VARCHAR(36) PRIMARY KEY,
    tournament_id VARCHAR(36) NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    event_id VARCHAR(36) NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    player_id VARCHAR(36) REFERENCES players(id) ON DELETE SET NULL,
    team_id VARCHAR(36) REFERENCES tournament_team_registrations(id) ON DELETE SET NULL,
    player_info JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT
);

CREATE INDEX IF NOT EXISTS idx_player_regs_tournament_event ON tournament_player_registrations(tournament_id, event_id);
CREATE INDEX IF NOT EXISTS idx_player_regs_team_id ON tournament_player_registrations(team_id);

-- Auction session (context only) + saved UI filter presets as JSON.
CREATE TABLE IF NOT EXISTS auctions (
    id VARCHAR(36) PRIMARY KEY,
    tournament_id VARCHAR(36) NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    event_id VARCHAR(36) NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    mode VARCHAR(50) NOT NULL DEFAULT 'open',
    filter_presets JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT
);

CREATE INDEX IF NOT EXISTS idx_auctions_tournament_event ON auctions(tournament_id, event_id);

-- Lots inside an auction. One lot points to one tournament registration.
CREATE TABLE IF NOT EXISTS auction_players (
    id VARCHAR(36) PRIMARY KEY,
    auction_id VARCHAR(36) NOT NULL REFERENCES auctions(id) ON DELETE CASCADE,
    tournament_player_registration_id VARCHAR(36) NOT NULL REFERENCES tournament_player_registrations(id) ON DELETE CASCADE,

    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'sold', 'unsold', 'skipped')),
    base_price BIGINT NOT NULL DEFAULT 0,
    final_price BIGINT NOT NULL DEFAULT 0,
    sold_to_team_registration_id VARCHAR(36) REFERENCES tournament_team_registrations(id) ON DELETE SET NULL,

    lot_number INT NOT NULL DEFAULT 0,
    order_index INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,

    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT
);

CREATE INDEX IF NOT EXISTS idx_auction_players_auction_id ON auction_players(auction_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_auction_players_auction_lot_number ON auction_players(auction_id, lot_number);
CREATE INDEX IF NOT EXISTS idx_auction_players_status ON auction_players(status);

-- Team wallets for tournament/event. Balance is denormalized for fast reads.
CREATE TABLE IF NOT EXISTS wallets (
    id VARCHAR(36) PRIMARY KEY,
    tournament_id VARCHAR(36) NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    event_id VARCHAR(36) NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    team_id VARCHAR(36) NOT NULL REFERENCES tournament_team_registrations(id) ON DELETE CASCADE,
    balance BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT,
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_wallets_tournament_event_team ON wallets(tournament_id, event_id, team_id);

-- Append-only wallet ledger.
CREATE TABLE IF NOT EXISTS wallet_transactions (
    id VARCHAR(36) PRIMARY KEY,
    wallet_id VARCHAR(36) NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL,
    type VARCHAR(10) NOT NULL CHECK (type IN ('credit', 'debit')),
    reference_id VARCHAR(36) NOT NULL,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT
);

CREATE INDEX IF NOT EXISTS idx_wallet_tx_wallet_id ON wallet_transactions(wallet_id);
CREATE INDEX IF NOT EXISTS idx_wallet_tx_reference_id ON wallet_transactions(reference_id);

-- Bids on a lot.
CREATE TABLE IF NOT EXISTS bids (
    id VARCHAR(36) PRIMARY KEY,
    auction_player_id VARCHAR(36) NOT NULL REFERENCES auction_players(id) ON DELETE CASCADE,
    team_registration_id VARCHAR(36) NOT NULL REFERENCES tournament_team_registrations(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL,
    recorded_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT
);

CREATE INDEX IF NOT EXISTS idx_bids_auction_player_id ON bids(auction_player_id);
CREATE INDEX IF NOT EXISTS idx_bids_team_registration_id ON bids(team_registration_id);
CREATE INDEX IF NOT EXISTS idx_bids_recorded_at ON bids(recorded_at);

-- +goose Down

DROP TABLE IF EXISTS bids;
DROP TABLE IF EXISTS wallet_transactions;
DROP TABLE IF EXISTS wallets;
DROP TABLE IF EXISTS auction_players;
DROP TABLE IF EXISTS auctions;
DROP TABLE IF EXISTS tournament_player_registrations;
DROP TABLE IF EXISTS tournament_team_registrations;
DROP TABLE IF EXISTS tournament_events;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS tournaments;

