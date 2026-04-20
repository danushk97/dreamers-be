import csv
import uuid
from datetime import datetime

input_file = "rally2win_apr13.csv"   # your input CSV
output_file = "tournament_registrations.sql"

TOURNAMENT_ID = "a0000001-0000-4000-8000-000000000001"
EVENT_ID = "a0000002-0000-4000-8000-000000000001"

def clean(value):
    if value is None or value.strip() == "" or value == "NULL":
        return "NULL"
    return "'" + value.replace("'", "''") + "'"

with open(input_file, newline='', encoding='utf-8') as csvfile, open(output_file, 'w', encoding='utf-8') as outfile:
    reader = csv.DictReader(csvfile)

    serial_number = 1

    for row in reader:
        record_id = str(uuid.uuid4())
        player_id = clean(row.get("id"))  # assuming player_id = id from your CSV
        team_id = clean(row.get("team_id")) if "team_id" in row else "NULL"

        created_at = "'" + datetime.now().strftime('%Y-%m-%d %H:%M:%S') + "'"

        query = f"""
INSERT INTO public.tournament_player_registrations(
    id, tournament_id, tournament_event_id, player_id, team_id, created_at, serial_number
) VALUES (
    '{record_id}',
    '{TOURNAMENT_ID}',
    '{EVENT_ID}',
    {player_id},
    {team_id},
    '1774847772229',
    {serial_number}
);
"""
        outfile.write(query.strip() + "\n\n")

        serial_number += 1

print("SQL file generated successfully!")