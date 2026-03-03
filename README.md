# Dreamers Backend

Go backend for the Dreamers player registration and management system. Uses **Gin**, **PostgreSQL**, **Google Drive** for file storage, and **Clean Architecture** with SOLID principles.

## Features

- **Create Player** – Register players with profile photo and Aadhar card (via upload)
- **List Players** – Filter by name, TNBA ID, gender, age brackets; pagination
- **File Upload** – Upload profile photo and Aadhar images to Google Drive

## Setup

### Prerequisites

- Go 1.22+
- PostgreSQL 16+
- Google Cloud project with Drive API enabled (for file upload)

### Database

Use a local or remote PostgreSQL instance. Create the `dreamers` database, then set the connection in `config.toml` or via env:

```bash
export DATABASE_URL="postgres://user:pass@host:5432/dreamers?sslmode=disable"
```

### Google Drive (optional)

For local dev without GDrive, the app uses a no-op uploader that returns placeholder URLs.

1. Create a [Google Cloud project](https://console.cloud.google.com/)
2. Enable [Google Drive API](https://console.cloud.google.com/apis/library/drive.googleapis.com)
3. Create a service account and download JSON credentials
4. Create a folder in Drive and share it with the service account email (Editor)
5. Set env:

```bash
export GDRIVE_CREDENTIALS_JSON="/path/to/service-account.json"
export GDRIVE_FOLDER_ID="optional-folder-id"
# Or inline:
export GDRIVE_CREDENTIALS='{"type":"service_account",...}'
```

### Migrations

Migrations live in `./migrations/` at the project root and use [goose](https://github.com/pressly/goose). Run with:

```bash
go run ./cmd/migrate up
# or: make migrate
```

Create new migrations:
```bash
go run github.com/pressly/goose/v3/cmd/goose@latest -dir migrations create add_new_table sql
```

### Run

```bash
go mod download
go run ./cmd/server
```

Server runs on `http://localhost:8080` (override with `PORT`).

## API

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/upload` | Upload file (form `file`), returns `{ "url": "..." }` |
| POST | `/api/v1/players` | Create player (JSON body) |
| GET | `/api/v1/players` | List players with filters |

### Create Player

```json
POST /api/v1/players
{
  "name": "Player Name",
  "imageURL": "https://drive.google.com/...",
  "aadharCardImageURL": "https://drive.google.com/...",
  "gender": "MALE",
  "dateOfBirth": "1990-05-15",
  "tnbaId": "TNBA123",
  "district": "Chennai",
  "phone": 9876543210,
  "recentAchievements": "Optional bio",
  "tshirtSize": "M"
}
```

### List Players

```
GET /api/v1/players?name=&tnbaId=&gender=&ageFilter=&page=0&limit=20
```

- `name` – substring search (case-insensitive)
- `tnbaId` – substring search
- `gender` – `MALE` | `FEMALE`
- `ageFilter` – `all` | `below-30` | `31-40` | `41-50` | `50+` (men) | `above-30` (women)

## Project Structure (Clean Architecture)

```
internal/
  domain/          # Entities, repository interfaces
  usecase/         # Business logic
  adapter/         # Implementations
    persistence/   # PostgreSQL
    storage/       # Google Drive
    http/          # Gin handlers
  pkg/sanitize/   # Input sanitization
migrations/        # SQL migrations
cmd/server/        # Entry point
```

## Tests

```bash
go test ./...
```

If you see `dyld: missing LC_UUID` on macOS, try `go clean -cache` and rerun.

## Configuration

Configuration is loaded from **config.toml** (Viper) with env overrides. Copy `config.example.toml` to `config.toml`.

```toml
[server]
port = "8080"

[database]
url = "postgres://postgres:postgres@localhost:5432/dreamers?sslmode=disable"
migration_path = "./migrations"

[gdrive]
credentials_path = ""
folder_id = ""
max_size_mb = 2
```

### Environment overrides

| Variable | Overrides | Description |
|----------|-----------|-------------|
| `PORT` | `server.port` | Server port |
| `DATABASE_URL` | `database.url` | PostgreSQL connection |
| `MIGRATION_PATH` | `database.migration_path` | Migration path |
| `GDRIVE_CREDENTIALS_JSON` | - | Path to service account JSON |
| `GDRIVE_CREDENTIALS` | - | Inline JSON credentials |
| `GDRIVE_FOLDER_ID` | `gdrive.folder_id` | Optional parent folder ID |
