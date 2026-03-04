# Dreamers Backend

Go backend for the Dreamers player registration and management system. Uses **Gin**, **PostgreSQL**, **AWS S3** for file storage, and **Clean Architecture** with SOLID principles.

## Features

- **Create Player** – Register players with profile photo and Aadhar card (via upload)
- **List Players** – Filter by name, TNBA ID, gender, age brackets; pagination
- **File Upload** – Upload profile photo and Aadhar images to S3 (private, presigned URLs for reads)

## Setup

### Prerequisites

- Go 1.22+
- PostgreSQL 16+
- AWS account with S3 (optional, for file upload)

### Database

Use a local or remote PostgreSQL instance. Create the `dreamers` database, then set the connection in `config.toml` or via env:

```bash
export DATABASE_URL="postgres://user:pass@host:5432/dreamers?sslmode=disable"
```

### S3 (optional)

For local dev without S3 configured, the app uses a no-op uploader. Objects are stored **private**; reads use **presigned URLs** (1hr expiry).

Configure in `config.toml` or env:
```bash
AWS_S3_BUCKET=your-bucket
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...
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
| POST | `/api/v1/upload` | Upload file (form `file`), returns `{ "key": "...", "url": "presigned..." }` |
| POST | `/api/v1/players` | Create player (JSON body) |
| GET | `/api/v1/players` | List players with filters |

### Create Player

```json
POST /api/v1/players
{
  "name": "Player Name",
  "imageURL": "uploads/2024/01/02/photo-abc123.jpg",
  "aadharCardImageURL": "uploads/2024/01/02/aadhar-xyz456.jpg",
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

[s3]
bucket = ""
region = "us-east-1"
max_size_mb = 2
```

### Environment overrides

| Variable | Overrides | Description |
|----------|-----------|-------------|
| `PORT` | `server.port` | Server port |
| `DATABASE_URL` | `database.url` | PostgreSQL connection |
| `MIGRATION_PATH` | `database.migration_path` | Migration path |
| `AWS_S3_BUCKET` | `s3.bucket` | S3 bucket for uploads |
| `AWS_REGION` | `s3.region` | AWS region |
