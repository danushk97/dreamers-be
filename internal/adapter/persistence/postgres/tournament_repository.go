package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/dreamers-be/internal/domain/auction"
)

var _ auction.TournamentRepository = (*TournamentRepository)(nil)

type TournamentRepository struct {
	db *sql.DB
}

func NewTournamentRepository(db *sql.DB) *TournamentRepository {
	return &TournamentRepository{db: db}
}

func (r *TournamentRepository) Create(ctx context.Context, t *auction.Tournament) error {
	if t == nil {
		return fmt.Errorf("tournament is nil")
	}
	sponsors := t.Sponsors
	if sponsors == nil {
		sponsors = []auction.TournamentSponsorGroup{}
	}
	raw, err := json.Marshal(sponsors)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(
		ctx,
		`INSERT INTO tournaments (id, name, sport_id, start_date, end_date, created_at, logo_url, sponsors)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		t.ID, t.Name, t.SportID, t.StartDate, t.EndDate, t.CreatedAt, t.LogoURL, raw,
	)
	return err
}

func (r *TournamentRepository) GetByID(ctx context.Context, id string) (*auction.Tournament, error) {
	var t auction.Tournament
	var sponsorsRaw []byte
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, name, sport_id, start_date, end_date, created_at, logo_url, sponsors
		 FROM tournaments WHERE id = $1`,
		id,
	).Scan(&t.ID, &t.Name, &t.SportID, &t.StartDate, &t.EndDate, &t.CreatedAt, &t.LogoURL, &sponsorsRaw)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(sponsorsRaw) > 0 && string(sponsorsRaw) != "null" {
		if err := json.Unmarshal(sponsorsRaw, &t.Sponsors); err != nil {
			return nil, err
		}
	}
	if t.Sponsors == nil {
		t.Sponsors = []auction.TournamentSponsorGroup{}
	}
	return &t, nil
}
