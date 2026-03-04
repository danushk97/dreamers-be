package postgres

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/dreamers-be/internal/domain/player"
)

var _ player.Repository = (*PlayerRepository)(nil)

// PlayerRepository implements player.Repository with PostgreSQL.
type PlayerRepository struct {
	db *sql.DB
}

// NewPlayerRepository returns a new postgres player repository.
func NewPlayerRepository(db *sql.DB) *PlayerRepository {
	return &PlayerRepository{db: db}
}

// ExistsByTNBAID returns true if a player with the given TNBA ID already exists.
func (r *PlayerRepository) ExistsByTNBAID(ctx context.Context, tnbaID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM players WHERE LOWER(tnba_id) = LOWER($1)",
		tnbaID,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Create inserts a player.
func (r *PlayerRepository) Create(ctx context.Context, p *player.Entity) error {
	query := `INSERT INTO players (
		id, name, image_url, gender, date_of_birth, tnba_id, district,
		phone, recent_achievements, tshirt_size, aadhar_card_image_url, created_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := r.db.ExecContext(ctx, query,
		p.ID, p.Name, p.ImageURL, p.Gender, p.DateOfBirth, p.TNBAID,
		p.District, p.Phone, nullIfEmpty(p.RecentAchievements), p.TshirtSize,
		p.AadharCardImageURL, p.CreatedAt.UnixMilli(),
	)
	return err
}

// List returns players matching the filter with pagination.
func (r *PlayerRepository) List(ctx context.Context, f *player.ListFilter) (*player.ListResult, error) {
	args := []any{}
	conditions := []string{}
	argIdx := 1

	if f.Name != "" {
		conditions = append(conditions, "LOWER(name) LIKE $"+placeholder(argIdx))
		args = append(args, "%"+strings.ToLower(f.Name)+"%")
		argIdx++
	}
	if f.TNBAID != "" {
		conditions = append(conditions, "LOWER(tnba_id) LIKE $"+placeholder(argIdx))
		args = append(args, "%"+strings.ToLower(f.TNBAID)+"%")
		argIdx++
	}
	if f.Gender != "" {
		conditions = append(conditions, "gender = $"+placeholder(argIdx))
		args = append(args, f.Gender)
		argIdx++
	}

	// Age filter: compute DOB bounds from age brackets
	if f.AgeFilter != "" && f.AgeFilter != "all" {
		now := time.Now()
		ageConds := ageConditions(f.Gender, f.AgeFilter, now)
		for _, c := range ageConds {
			conditions = append(conditions, "date_of_birth "+c.op+" $"+placeholder(argIdx))
			args = append(args, c.value)
			argIdx++
		}
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM players" + where
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	// Pagination
	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	page := f.Page
	if page < 0 {
		page = 0
	}
	offset := page * limit
	pageCount := int((total + int64(limit) - 1) / int64(limit))
	if total == 0 {
		pageCount = 0
	}

	args = append(args, limit, offset)
	limitArg, offsetArg := argIdx, argIdx+1
	listQuery := `SELECT id, name, image_url, gender, date_of_birth, tnba_id, district,
		phone, COALESCE(recent_achievements, ''), tshirt_size, aadhar_card_image_url, created_at
		FROM players` + where + ` ORDER BY created_at DESC LIMIT $` + placeholder(limitArg) + ` OFFSET $` + placeholder(offsetArg)

	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []*player.Entity
	for rows.Next() {
		var p player.Entity
		var dob time.Time
		var recentAch sql.NullString
		var createdAtMs int64
		if err := rows.Scan(
			&p.ID, &p.Name, &p.ImageURL, &p.Gender, &dob, &p.TNBAID, &p.District,
			&p.Phone, &recentAch, &p.TshirtSize, &p.AadharCardImageURL, &createdAtMs,
		); err != nil {
			return nil, err
		}
		p.DateOfBirth = dob
		p.CreatedAt = time.UnixMilli(createdAtMs)
		if recentAch.Valid {
			p.RecentAchievements = recentAch.String
		}
		players = append(players, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &player.ListResult{
		Players:   players,
		Total:     total,
		Page:      page,
		Limit:     limit,
		PageCount: pageCount,
	}, nil
}

func placeholder(i int) string {
	return strconv.Itoa(i)
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

type ageCond struct {
	op    string
	value time.Time
}

// ageConditions returns SQL conditions for the age filter.
// below-30: DOB > (now-30) | 31-40, 41-50: DOB between range | 50+, above-30: DOB <= cutoff
func ageConditions(gender, ageFilter string, now time.Time) []ageCond {
	var conds []ageCond
	switch ageFilter {
	case "below-30":
		conds = append(conds, ageCond{op: ">", value: now.AddDate(-30, 0, 0)})
	case "31-40":
		if gender != player.GenderFemale {
			conds = append(conds, ageCond{op: ">=", value: now.AddDate(-41, 0, 0)})
			conds = append(conds, ageCond{op: "<=", value: now.AddDate(-31, 0, 0)})
		}
	case "41-50":
		if gender != player.GenderFemale {
			conds = append(conds, ageCond{op: ">=", value: now.AddDate(-51, 0, 0)})
			conds = append(conds, ageCond{op: "<=", value: now.AddDate(-41, 0, 0)})
		}
	case "50+":
		if gender != player.GenderFemale {
			conds = append(conds, ageCond{op: "<=", value: now.AddDate(-51, 0, 0)})
		}
	case "above-30":
		if gender == player.GenderFemale {
			conds = append(conds, ageCond{op: "<=", value: now.AddDate(-31, 0, 0)})
		}
	}
	return conds
}
