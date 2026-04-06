-- +goose Up

ALTER TABLE tournament_player_registrations
    ADD COLUMN serial_number INTEGER;

UPDATE tournament_player_registrations AS t
SET serial_number = s.rn
FROM (
    SELECT id, ROW_NUMBER() OVER (ORDER BY created_at ASC, id ASC) AS rn
    FROM tournament_player_registrations
) AS s
WHERE t.id = s.id;

CREATE SEQUENCE tournament_player_registrations_serial_number_seq;
SELECT setval(
    'tournament_player_registrations_serial_number_seq',
    COALESCE((SELECT MAX(serial_number) FROM tournament_player_registrations), 1),
    EXISTS (SELECT 1 FROM tournament_player_registrations)
);

ALTER TABLE tournament_player_registrations
    ALTER COLUMN serial_number SET DEFAULT nextval('tournament_player_registrations_serial_number_seq');

ALTER TABLE tournament_player_registrations
    ALTER COLUMN serial_number SET NOT NULL;

ALTER SEQUENCE tournament_player_registrations_serial_number_seq OWNED BY tournament_player_registrations.serial_number;

-- +goose Down

ALTER TABLE tournament_player_registrations
    DROP COLUMN IF EXISTS serial_number;

DROP SEQUENCE IF EXISTS tournament_player_registrations_serial_number_seq;
