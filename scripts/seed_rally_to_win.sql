-- Seed data: tournament "Rally to Win", team event (4–4 roster, auction 1500–4000 INR in paise),
-- 2 teams, wallets 10_000 INR each (paise), 20 players + registrations (player_id linked; player_info snapshot).
-- Run after migrations. Adjust UUIDs if they collide with existing rows.

BEGIN;

-- 1) Tournament: Rally to Win
INSERT INTO tournaments (id, name, sport_id, start_date, end_date)
VALUES (
    'a0000001-0000-4000-8000-000000000001',
    'Rally to Win',
    'badminton',
    (EXTRACT(EPOCH FROM TIMESTAMPTZ '2026-05-01 00:00:00+00') * 1000)::BIGINT,
    (EXTRACT(EPOCH FROM TIMESTAMPTZ '2026-05-07 23:59:59+00') * 1000)::BIGINT
);

-- 2) Tournament event: Team Event — min/max 4 players, base 1500 INR, max 4000 INR (paise)
--    JSON keys match Go structs (no json tags): TeamEventRules, MinPlayersPerTeam, BaseBid, MaxBid, IsAuction
INSERT INTO tournament_events (id, tournament_id, name, attrs, parent_event_id)
VALUES (
    'a0000002-0000-4000-8000-000000000001',
    'a0000001-0000-4000-8000-000000000001',
    'Team Event',
    '{
      "TeamEventRules": {
        "MinPlayersPerTeam": 4,
        "MaxPlayersPerTeam": 4,
        "IsAuction": true,
        "BaseBid": 150000,
        "MaxBid": 400000
      }
    }'::jsonb,
    NULL
);

-- 3) Two teams
INSERT INTO tournament_team_registrations (id, tournament_id, tournament_event_id, team_name, team_logo_url)
VALUES
    ('a0000003-0000-4000-8000-000000000001', 'a0000001-0000-4000-8000-000000000001', 'a0000002-0000-4000-8000-000000000001', 'Smash Kings', ''),
    ('a0000004-0000-4000-8000-000000000001', 'a0000001-0000-4000-8000-000000000001', 'a0000002-0000-4000-8000-000000000001', 'Net Ninjas', '');

-- 4) Wallets: 10_000 INR = 1_000_000 paise per team
INSERT INTO wallets (id, tournament_id, tournament_event_id, team_id, balance, created_at, updated_at)
VALUES
    (
        'a0000005-0000-4000-8000-000000000001',
        'a0000001-0000-4000-8000-000000000001',
        'a0000002-0000-4000-8000-000000000001',
        'a0000003-0000-4000-8000-000000000001',
        1000000,
        (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT,
        (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT
    ),
    (
        'a0000006-0000-4000-8000-000000000001',
        'a0000001-0000-4000-8000-000000000001',
        'a0000002-0000-4000-8000-000000000001',
        'a0000004-0000-4000-8000-000000000001',
        1000000,
        (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT,
        (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT
    );

-- Optional: ledger credits for initial float (reference_id must fit VARCHAR(36))
INSERT INTO wallet_transactions (id, wallet_id, amount, type, reference_id, created_at)
VALUES
    ('b0000001-0000-4000-8000-000000000001', 'a0000005-0000-4000-8000-000000000001', 1000000, 'credit', 'b0000001-0000-4000-8000-000000000001', (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT),
    ('b0000002-0000-4000-8000-000000000001', 'a0000006-0000-4000-8000-000000000001', 1000000, 'credit', 'b0000002-0000-4000-8000-000000000001', (EXTRACT(EPOCH FROM NOW()) * 1000)::BIGINT);

-- 5) Canonical players (IDs d0000001…d0000014 = 20 rows; date_of_birth matches registration DateOfBirthMs UTC)
INSERT INTO players (id, name, image_url, gender, date_of_birth, tnba_id, district, phone, recent_achievements, tshirt_size, aadhar_card_image_url)
VALUES
    ('d0000001-0000-4000-8000-000000000001', 'Player 01', 'https://example.com/players/01.jpg', 'MALE', DATE '2000-01-01', 'TNBA-RTW-2026-01', 'Chennai', '+919000000001', NULL, 'M', 'https://example.com/aadhar/01.jpg'),
    ('d0000002-0000-4000-8000-000000000002', 'Player 02', 'https://example.com/players/02.jpg', 'FEMALE', DATE '2001-01-01', 'TNBA-RTW-2026-02', 'Chennai', '+919000000002', NULL, 'M', 'https://example.com/aadhar/02.jpg'),
    ('d0000003-0000-4000-8000-000000000003', 'Player 03', 'https://example.com/players/03.jpg', 'MALE', DATE '1999-01-01', 'TNBA-RTW-2026-03', 'Chennai', '+919000000003', NULL, 'M', 'https://example.com/aadhar/03.jpg'),
    ('d0000004-0000-4000-8000-000000000004', 'Player 04', 'https://example.com/players/04.jpg', 'FEMALE', DATE '2002-01-01', 'TNBA-RTW-2026-04', 'Chennai', '+919000000004', NULL, 'M', 'https://example.com/aadhar/04.jpg'),
    ('d0000005-0000-4000-8000-000000000005', 'Player 05', 'https://example.com/players/05.jpg', 'MALE', DATE '1998-01-01', 'TNBA-RTW-2026-05', 'Chennai', '+919000000005', NULL, 'M', 'https://example.com/aadhar/05.jpg'),
    ('d0000006-0000-4000-8000-000000000006', 'Player 06', 'https://example.com/players/06.jpg', 'FEMALE', DATE '2003-01-01', 'TNBA-RTW-2026-06', 'Chennai', '+919000000006', NULL, 'M', 'https://example.com/aadhar/06.jpg'),
    ('d0000007-0000-4000-8000-000000000007', 'Player 07', 'https://example.com/players/07.jpg', 'MALE', DATE '1995-01-01', 'TNBA-RTW-2026-07', 'Chennai', '+919000000007', NULL, 'M', 'https://example.com/aadhar/07.jpg'),
    ('d0000008-0000-4000-8000-000000000008', 'Player 08', 'https://example.com/players/08.jpg', 'FEMALE', DATE '2004-01-01', 'TNBA-RTW-2026-08', 'Chennai', '+919000000008', NULL, 'M', 'https://example.com/aadhar/08.jpg'),
    ('d0000009-0000-4000-8000-000000000009', 'Player 09', 'https://example.com/players/09.jpg', 'MALE', DATE '1996-12-31', 'TNBA-RTW-2026-09', 'Chennai', '+919000000009', NULL, 'M', 'https://example.com/aadhar/09.jpg'),
    ('d000000a-0000-4000-8000-00000000000a', 'Player 10', 'https://example.com/players/10.jpg', 'FEMALE', DATE '2005-01-01', 'TNBA-RTW-2026-10', 'Chennai', '+919000000010', NULL, 'M', 'https://example.com/aadhar/10.jpg'),
    ('d000000b-0000-4000-8000-00000000000b', 'Player 11', 'https://example.com/players/11.jpg', 'MALE', DATE '1996-01-01', 'TNBA-RTW-2026-11', 'Chennai', '+919000000011', NULL, 'M', 'https://example.com/aadhar/11.jpg'),
    ('d000000c-0000-4000-8000-00000000000c', 'Player 12', 'https://example.com/players/12.jpg', 'FEMALE', DATE '2006-01-01', 'TNBA-RTW-2026-12', 'Chennai', '+919000000012', NULL, 'M', 'https://example.com/aadhar/12.jpg'),
    ('d000000d-0000-4000-8000-00000000000d', 'Player 13', 'https://example.com/players/13.jpg', 'MALE', DATE '1994-01-01', 'TNBA-RTW-2026-13', 'Chennai', '+919000000013', NULL, 'M', 'https://example.com/aadhar/13.jpg'),
    ('d000000e-0000-4000-8000-00000000000e', 'Player 14', 'https://example.com/players/14.jpg', 'FEMALE', DATE '2007-01-01', 'TNBA-RTW-2026-14', 'Chennai', '+919000000014', NULL, 'M', 'https://example.com/aadhar/14.jpg'),
    ('d000000f-0000-4000-8000-00000000000f', 'Player 15', 'https://example.com/players/15.jpg', 'MALE', DATE '1992-01-01', 'TNBA-RTW-2026-15', 'Chennai', '+919000000015', NULL, 'M', 'https://example.com/aadhar/15.jpg'),
    ('d0000010-0000-4000-8000-000000000010', 'Player 16', 'https://example.com/players/16.jpg', 'FEMALE', DATE '2008-01-01', 'TNBA-RTW-2026-16', 'Chennai', '+919000000016', NULL, 'M', 'https://example.com/aadhar/16.jpg'),
    ('d0000011-0000-4000-8000-000000000011', 'Player 17', 'https://example.com/players/17.jpg', 'MALE', DATE '1991-01-01', 'TNBA-RTW-2026-17', 'Chennai', '+919000000017', NULL, 'M', 'https://example.com/aadhar/17.jpg'),
    ('d0000012-0000-4000-8000-000000000012', 'Player 18', 'https://example.com/players/18.jpg', 'FEMALE', DATE '2009-01-01', 'TNBA-RTW-2026-18', 'Chennai', '+919000000018', NULL, 'M', 'https://example.com/aadhar/18.jpg'),
    ('d0000013-0000-4000-8000-000000000013', 'Player 19', 'https://example.com/players/19.jpg', 'MALE', DATE '1990-01-01', 'TNBA-RTW-2026-19', 'Chennai', '+919000000019', NULL, 'M', 'https://example.com/aadhar/19.jpg'),
    ('d0000014-0000-4000-8000-000000000014', 'Player 20', 'https://example.com/players/20.jpg', 'FEMALE', DATE '2010-01-01', 'TNBA-RTW-2026-20', 'Chennai', '+919000000020', NULL, 'M', 'https://example.com/aadhar/20.jpg');

-- 6) Tournament player registrations (team_id NULL; player_id links to players above — same order as your seed)
INSERT INTO tournament_player_registrations (id, tournament_id, tournament_event_id, player_id, team_id)
VALUES
    ('c0000001-0000-4000-8000-000000000001', 'a0000001-0000-4000-8000-000000000001', 'a0000002-0000-4000-8000-000000000001', 'd0000001-0000-4000-8000-000000000001', NULL),
    ('c0000002-0000-4000-8000-000000000001', 'a0000001-0000-4000-8000-000000000001', 'a0000002-0000-4000-8000-000000000001', 'd0000002-0000-4000-8000-000000000002', NULL),
    ('c0000003-0000-4000-8000-000000000001', 'a0000001-0000-4000-8000-000000000001', 'a0000002-0000-4000-8000-000000000001', 'd0000003-0000-4000-8000-000000000003', NULL),
    ('c0000004-0000-4000-8000-000000000001', 'a0000001-0000-4000-8000-000000000001', 'a0000002-0000-4000-8000-000000000001', 'd0000004-0000-4000-8000-000000000004', NULL),
    ('c0000005-0000-4000-8000-000000000001', 'a0000001-0000-4000-8000-000000000001', 'a0000002-0000-4000-8000-000000000001', 'd0000005-0000-4000-8000-000000000005', NULL),
    ('c0000006-0000-4000-8000-000000000001', 'a0000001-0000-4000-8000-000000000001', 'a0000002-0000-4000-8000-000000000001', 'd0000006-0000-4000-8000-000000000006', NULL),
    ('c0000007-0000-4000-8000-000000000001', 'a0000001-0000-4000-8000-000000000001', 'a0000002-0000-4000-8000-000000000001', 'd0000007-0000-4000-8000-000000000007', NULL),
    ('c0000008-0000-4000-8000-000000000001', 'a0000001-0000-4000-8000-000000000001', 'a0000002-0000-4000-8000-000000000001', 'd0000008-0000-4000-8000-000000000008', NULL),
    ('c0000009-0000-4000-8000-000000000001', 'a0000001-0000-4000-8000-000000000001', 'a0000002-0000-4000-8000-000000000001', 'd0000009-0000-4000-8000-000000000009', NULL),
    ('c000000a-0000-4000-8000-000000000001', 'a0000001-0000-4000-8000-000000000001', 'a0000002-0000-4000-8000-000000000001', 'd000000a-0000-4000-8000-00000000000a', NULL),
    ('c000000b-0000-4000-8000-000000000001', 'a0000001-0000-4000-8000-000000000001', 'a0000002-0000-4000-8000-000000000001', 'd000000b-0000-4000-8000-00000000000b', NULL),
    ('c000000c-0000-4000-8000-000000000001', 'a0000001-0000-4000-8000-000000000001', 'a0000002-0000-4000-8000-000000000001', 'd000000c-0000-4000-8000-00000000000c', NULL),
    ('c000000d-0000-4000-8000-000000000001', 'a0000001-0000-4000-8000-000000000001', 'a0000002-0000-4000-8000-000000000001', 'd000000d-0000-4000-8000-00000000000d', NULL),
    ('c000000e-0000-4000-8000-000000000001', 'a0000001-0000-4000-8000-000000000001', 'a0000002-0000-4000-8000-000000000001', 'd000000e-0000-4000-8000-00000000000e', NULL),
    ('c000000f-0000-4000-8000-000000000001', 'a0000001-0000-4000-8000-000000000001', 'a0000002-0000-4000-8000-000000000001', 'd000000f-0000-4000-8000-00000000000f', NULL),
    ('c0000010-0000-4000-8000-000000000001', 'a0000001-0000-4000-8000-000000000001', 'a0000002-0000-4000-8000-000000000001', 'd0000010-0000-4000-8000-000000000010', NULL),
    ('c0000011-0000-4000-8000-000000000001', 'a0000001-0000-4000-8000-000000000001', 'a0000002-0000-4000-8000-000000000001', 'd0000011-0000-4000-8000-000000000011', NULL),
    ('c0000012-0000-4000-8000-000000000001', 'a0000001-0000-4000-8000-000000000001', 'a0000002-0000-4000-8000-000000000001', 'd0000012-0000-4000-8000-000000000012', NULL),
    ('c0000013-0000-4000-8000-000000000001', 'a0000001-0000-4000-8000-000000000001', 'a0000002-0000-4000-8000-000000000001', 'd0000013-0000-4000-8000-000000000013', NULL),
    ('c0000014-0000-4000-8000-000000000001', 'a0000001-0000-4000-8000-000000000001', 'a0000002-0000-4000-8000-000000000001', 'd0000014-0000-4000-8000-000000000014', NULL);

COMMIT;
