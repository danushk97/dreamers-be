-- +goose Up
CREATE TABLE IF NOT EXISTS players (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    image_url VARCHAR(2048) NOT NULL,
    gender VARCHAR(10) NOT NULL CHECK (gender IN ('MALE', 'FEMALE')),
    date_of_birth DATE NOT NULL,
    tnba_id VARCHAR(100) NOT NULL,
    district VARCHAR(100) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    recent_achievements VARCHAR(300),
    tshirt_size VARCHAR(10) NOT NULL CHECK (tshirt_size IN ('XS', 'S', 'M', 'L', 'XL', 'XXL', 'XXXL')),
    aadhar_card_image_url VARCHAR(2048) NOT NULL,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT
);

CREATE INDEX idx_players_tnba_id ON players(tnba_id);
CREATE INDEX idx_players_gender ON players(gender);
CREATE INDEX idx_players_district ON players(district);
CREATE INDEX idx_players_date_of_birth ON players(date_of_birth);
CREATE INDEX idx_players_name_lower ON players(LOWER(name));
CREATE INDEX idx_players_tnba_id_lower ON players(LOWER(tnba_id));
