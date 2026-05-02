package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dreamers-be/internal/domain/auction"
)

// --- TournamentEventRepository ---

var _ auction.TournamentEventRepository = (*TournamentEventRepository)(nil)

type TournamentEventRepository struct {
	db *sql.DB
}

func NewTournamentEventRepository(db *sql.DB) *TournamentEventRepository {
	return &TournamentEventRepository{db: db}
}

func (r *TournamentEventRepository) Create(ctx context.Context, te *auction.TournamentEvent) error {
	if te == nil {
		return fmt.Errorf("tournament event is nil")
	}
	attrsRaw, err := json.Marshal(te.Attrs)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(
		ctx,
		`INSERT INTO tournament_events (id, tournament_id, name, attrs, parent_event_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		te.ID,
		te.TournamentID,
		te.Name,
		attrsRaw,
		nullIfEmpty(te.ParentEventID),
		te.CreatedAt,
	)
	return err
}

func (r *TournamentEventRepository) GetByID(ctx context.Context, id string) (*auction.TournamentEvent, error) {
	var te auction.TournamentEvent
	var attrsRaw []byte
	var parentID sql.NullString
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, tournament_id, name, attrs, parent_event_id, created_at
		 FROM tournament_events WHERE id = $1`,
		id,
	).Scan(&te.ID, &te.TournamentID, &te.Name, &attrsRaw, &parentID, &te.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		te.ParentEventID = parentID.String
	}
	if len(attrsRaw) > 0 {
		if err := json.Unmarshal(attrsRaw, &te.Attrs); err != nil {
			return nil, err
		}
	}
	return &te, nil
}

// --- AuctionRepository ---

var _ auction.AuctionRepository = (*AuctionRepository)(nil)

type AuctionRepository struct {
	db *sql.DB
}

func NewAuctionRepository(db *sql.DB) *AuctionRepository {
	return &AuctionRepository{db: db}
}

func (r *AuctionRepository) Create(ctx context.Context, a *auction.Auction) error {
	if a == nil {
		return fmt.Errorf("auction is nil")
	}
	filterRaw, err := json.Marshal(a.FilterPresets)
	if err != nil {
		return err
	}
	rulesRaw, err := json.Marshal(a.Rules)
	if err != nil {
		return err
	}
	runMode := string(a.RunMode)
	if runMode == "" {
		runMode = string(auction.AuctionRunModeTest)
	}
	_, err = r.db.ExecContext(
		ctx,
		`INSERT INTO auctions (id, tournament_id, tournament_event_id, mode, run_mode, filter_presets, rules, display_auction_player_id, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NULL, $8)`,
		a.ID, a.TournamentID, a.TournamentEventID, string(a.Mode), runMode, filterRaw, rulesRaw, a.CreatedAt,
	)
	return err
}

func (r *AuctionRepository) GetByID(ctx context.Context, id string) (*auction.Auction, error) {
	var a auction.Auction
	var filterRaw []byte
	var rulesRaw []byte
	var modeStr string
	var runModeStr string
	var displayLotID sql.NullString
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, tournament_id, tournament_event_id, mode, run_mode, filter_presets, rules, display_auction_player_id, created_at
		 FROM auctions WHERE id = $1`,
		id,
	).Scan(&a.ID, &a.TournamentID, &a.TournamentEventID, &modeStr, &runModeStr, &filterRaw, &rulesRaw, &displayLotID, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.Mode = auction.AuctionMode(modeStr)
	if runModeStr != "" {
		a.RunMode = auction.AuctionRunMode(runModeStr)
	} else {
		a.RunMode = auction.AuctionRunModeTest
	}

	if len(filterRaw) > 0 {
		if err := json.Unmarshal(filterRaw, &a.FilterPresets); err != nil {
			return nil, err
		}
	}
	if len(rulesRaw) > 0 && string(rulesRaw) != "null" {
		if err := json.Unmarshal(rulesRaw, &a.Rules); err != nil {
			return nil, err
		}
	}
	if a.Rules.MinBidAmount <= 0 && a.Rules.MaxBidAmount <= 0 {
		a.Rules = auction.ApplyDefaultAuctionRules(a.Rules)
	}
	if displayLotID.Valid {
		a.DisplayAuctionPlayerID = displayLotID.String
	}
	return &a, nil
}

func (r *AuctionRepository) UpdateDisplayAuctionPlayer(ctx context.Context, auctionID, auctionPlayerID string) error {
	if auctionID == "" {
		return fmt.Errorf("auction_id is required")
	}
	var err error
	if auctionPlayerID == "" {
		_, err = r.db.ExecContext(
			ctx,
			`UPDATE auctions SET display_auction_player_id = NULL WHERE id = $1`,
			auctionID,
		)
	} else {
		_, err = r.db.ExecContext(
			ctx,
			`UPDATE auctions SET display_auction_player_id = $1 WHERE id = $2`,
			auctionPlayerID, auctionID,
		)
	}
	if err != nil {
		return err
	}
	return nil
}

func (r *AuctionRepository) UpdateRunMode(ctx context.Context, auctionID string, runMode auction.AuctionRunMode) error {
	if auctionID == "" {
		return fmt.Errorf("auction_id is required")
	}
	rs := string(runMode)
	if rs != string(auction.AuctionRunModeTest) && rs != string(auction.AuctionRunModeLive) {
		return fmt.Errorf("invalid run_mode")
	}
	res, err := r.db.ExecContext(
		ctx,
		`UPDATE auctions SET run_mode = $1 WHERE id = $2`,
		rs, auctionID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("auction not found")
	}
	return nil
}

// --- TeamRepository ---

var _ auction.TeamRepository = (*TeamRepository)(nil)

type TeamRepository struct {
	db *sql.DB
}

func NewTeamRepository(db *sql.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

func (r *TeamRepository) Create(ctx context.Context, team *auction.TournamentTeamRegistration) error {
	if team == nil {
		return fmt.Errorf("team is nil")
	}
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO tournament_team_registrations (id, tournament_id, tournament_event_id, team_name, team_logo_url, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		team.ID, team.TournamentID, team.TournamentEventID, team.TeamName, team.TeamLogoURL, team.CreatedAt,
	)
	return err
}

func (r *TeamRepository) GetByID(ctx context.Context, id string) (*auction.TournamentTeamRegistration, error) {
	var team auction.TournamentTeamRegistration
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, tournament_id, tournament_event_id, team_name, team_logo_url, created_at
		 FROM tournament_team_registrations WHERE id = $1`,
		id,
	).Scan(&team.ID, &team.TournamentID, &team.TournamentEventID, &team.TeamName, &team.TeamLogoURL, &team.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &team, nil
}

func (r *TeamRepository) ListByTournamentEvent(ctx context.Context, tournamentID, tournamentEventID string) ([]*auction.TournamentTeamRegistration, error) {
	if tournamentID == "" || tournamentEventID == "" {
		return nil, fmt.Errorf("tournament_id and tournament_event_id are required")
	}
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, tournament_id, tournament_event_id, team_name, team_logo_url, created_at
		 FROM tournament_team_registrations
		 WHERE tournament_id = $1 AND tournament_event_id = $2
		 ORDER BY team_name ASC, created_at ASC`,
		tournamentID, tournamentEventID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*auction.TournamentTeamRegistration
	for rows.Next() {
		var t auction.TournamentTeamRegistration
		if err := rows.Scan(&t.ID, &t.TournamentID, &t.TournamentEventID, &t.TeamName, &t.TeamLogoURL, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &t)
	}
	return out, rows.Err()
}

// --- RegistrationRepository ---

var _ auction.RegistrationRepository = (*RegistrationRepository)(nil)

type RegistrationRepository struct {
	db *sql.DB
}

func NewRegistrationRepository(db *sql.DB) *RegistrationRepository {
	return &RegistrationRepository{db: db}
}

func (r *RegistrationRepository) GetByID(ctx context.Context, id string) (*auction.TournamentPlayerRegistration, error) {
	var reg auction.TournamentPlayerRegistration
	var playerID sql.NullString
	var teamID sql.NullString
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, tournament_id, tournament_event_id, player_id, team_id, serial_number, created_at
		 FROM tournament_player_registrations WHERE id = $1`,
		id,
	).Scan(&reg.ID, &reg.TournamentID, &reg.TournamentEventID, &playerID, &teamID, &reg.SerialNumber, &reg.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if playerID.Valid {
		reg.PlayerID = playerID.String
	}
	if teamID.Valid {
		reg.TeamID = teamID.String
	}
	return &reg, nil
}

func (r *RegistrationRepository) ListEligible(ctx context.Context, tournamentID, tournamentEventID string, f auction.RegistrationFilter) ([]*auction.TournamentPlayerRegistration, error) {
	if tournamentID == "" || tournamentEventID == "" {
		return nil, fmt.Errorf("tournament_id and tournament_event_id are required")
	}
	now := time.Now().UTC().Truncate(24 * time.Hour)

	conds := []string{
		"tr.tournament_id = $1",
		"tr.tournament_event_id = $2",
		"(tr.team_id IS NULL OR tr.team_id = '')",
		"tr.player_id IS NOT NULL",
	}
	args := []any{tournamentID, tournamentEventID}
	argIdx := 3

	if f.Gender != "" {
		conds = append(conds, "p.gender = $"+fmt.Sprint(argIdx))
		args = append(args, f.Gender)
		argIdx++
	}
	if f.MinAgeYears > 0 {
		cutoff := now.AddDate(-f.MinAgeYears, 0, 0)
		conds = append(conds, "p.date_of_birth <= $"+fmt.Sprint(argIdx)+"::date")
		args = append(args, cutoff)
		argIdx++
	}
	if f.MaxAgeYears > 0 {
		// Age <= MaxAgeYears corresponds to DOB >= now-(MaxAgeYears+1)
		cutoff := now.AddDate(-(f.MaxAgeYears + 1), 0, 0)
		conds = append(conds, "p.date_of_birth >= $"+fmt.Sprint(argIdx)+"::date")
		args = append(args, cutoff)
		argIdx++
	}

	query := `SELECT tr.id, tr.tournament_id, tr.tournament_event_id, tr.player_id, tr.team_id, tr.serial_number, tr.created_at
			  FROM tournament_player_registrations tr
			  JOIN players p ON p.id = tr.player_id
			  WHERE ` + joinConds(conds, " AND ") + `
			  ORDER BY tr.created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*auction.TournamentPlayerRegistration
	for rows.Next() {
		var reg auction.TournamentPlayerRegistration
		var playerID sql.NullString
		var teamID sql.NullString
		if err := rows.Scan(
			&reg.ID,
			&reg.TournamentID,
			&reg.TournamentEventID,
			&playerID,
			&teamID,
			&reg.SerialNumber,
			&reg.CreatedAt,
		); err != nil {
			return nil, err
		}
		if playerID.Valid {
			reg.PlayerID = playerID.String
		}
		if teamID.Valid {
			reg.TeamID = teamID.String
		}
		out = append(out, &reg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *RegistrationRepository) AssignToTeam(ctx context.Context, registrationID string, teamRegistrationID string) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE tournament_player_registrations
		 SET team_id = $1
		 WHERE id = $2`,
		teamRegistrationID, registrationID,
	)
	return err
}

func (r *RegistrationRepository) UnassignTeam(ctx context.Context, registrationID string) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE tournament_player_registrations
		 SET team_id = NULL
		 WHERE id = $1`,
		registrationID,
	)
	return err
}

// --- AuctionRepository lots ---

var _ auction.AuctionPlayerRepository = (*AuctionPlayerRepository)(nil)

type AuctionPlayerRepository struct {
	db *sql.DB
}

func NewAuctionPlayerRepository(db *sql.DB) *AuctionPlayerRepository {
	return &AuctionPlayerRepository{db: db}
}

func (r *AuctionPlayerRepository) Create(ctx context.Context, ap *auction.AuctionPlayer) error {
	if ap == nil {
		return fmt.Errorf("auction player is nil")
	}
	notesRaw, err := json.Marshal(ap.Notes)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(
		ctx,
		`INSERT INTO auction_players (
			id, auction_id, tournament_player_registration_id,
			status, notes, base_price, final_price, sold_to_team_registration_id,
			lot_number, is_active, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		ap.ID,
		ap.AuctionID,
		ap.TournamentPlayerRegistrationID,
		string(ap.Status),
		notesRaw,
		ap.BasePrice,
		ap.FinalPrice,
		nullIfEmpty(ap.SoldToTeamRegistrationID),
		ap.LotNumber,
		ap.IsActive,
		ap.CreatedAt,
	)
	return err
}

func (r *AuctionPlayerRepository) GetByID(ctx context.Context, id string) (*auction.AuctionPlayer, error) {
	var ap auction.AuctionPlayer
	var soldTo sql.NullString
	var notesRaw []byte
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, auction_id, tournament_player_registration_id,
			 status, notes, base_price, final_price, sold_to_team_registration_id,
			 lot_number, is_active, created_at
		 FROM auction_players WHERE id = $1`,
		id,
	).Scan(
		&ap.ID, &ap.AuctionID, &ap.TournamentPlayerRegistrationID,
		&ap.Status, &notesRaw, &ap.BasePrice, &ap.FinalPrice, &soldTo,
		&ap.LotNumber, &ap.IsActive, &ap.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if soldTo.Valid {
		ap.SoldToTeamRegistrationID = soldTo.String
	}
	if len(notesRaw) > 0 && string(notesRaw) != "null" {
		if err := json.Unmarshal(notesRaw, &ap.Notes); err != nil {
			return nil, err
		}
	}
	return &ap, nil
}

func (r *AuctionPlayerRepository) GetByAuctionAndRegistrationSerial(ctx context.Context, auctionID string, serialNumber int) (*auction.AuctionPlayer, error) {
	if auctionID == "" || serialNumber < 1 {
		return nil, fmt.Errorf("auction_id and positive serial_number are required")
	}
	var ap auction.AuctionPlayer
	var soldTo sql.NullString
	var notesRaw []byte
	err := r.db.QueryRowContext(
		ctx,
		`SELECT ap.id, ap.auction_id, ap.tournament_player_registration_id,
				ap.status, ap.notes, ap.base_price, ap.final_price, ap.sold_to_team_registration_id,
				ap.lot_number, ap.is_active, ap.created_at
		 FROM auction_players ap
		 INNER JOIN tournament_player_registrations tr ON tr.id = ap.tournament_player_registration_id
		 INNER JOIN auctions a ON a.id = ap.auction_id
		 WHERE ap.auction_id = $1
		   AND tr.serial_number = $2
		   AND tr.tournament_id = a.tournament_id
		   AND tr.tournament_event_id = a.tournament_event_id`,
		auctionID, serialNumber,
	).Scan(
		&ap.ID, &ap.AuctionID, &ap.TournamentPlayerRegistrationID,
		&ap.Status, &notesRaw, &ap.BasePrice, &ap.FinalPrice, &soldTo,
		&ap.LotNumber, &ap.IsActive, &ap.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if soldTo.Valid {
		ap.SoldToTeamRegistrationID = soldTo.String
	}
	if len(notesRaw) > 0 && string(notesRaw) != "null" {
		if err := json.Unmarshal(notesRaw, &ap.Notes); err != nil {
			return nil, err
		}
	}
	return &ap, nil
}

func (r *AuctionPlayerRepository) ListByAuction(ctx context.Context, auctionID string) ([]*auction.AuctionPlayer, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, auction_id, tournament_player_registration_id,
			 status, notes, base_price, final_price, sold_to_team_registration_id,
			 lot_number, is_active, created_at
		 FROM auction_players WHERE auction_id = $1
		 ORDER BY lot_number ASC, created_at DESC`,
		auctionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*auction.AuctionPlayer
	for rows.Next() {
		var ap auction.AuctionPlayer
		var soldTo sql.NullString
		var notesRaw []byte
		if err := rows.Scan(
			&ap.ID, &ap.AuctionID, &ap.TournamentPlayerRegistrationID,
			&ap.Status, &notesRaw, &ap.BasePrice, &ap.FinalPrice, &soldTo,
			&ap.LotNumber, &ap.IsActive, &ap.CreatedAt,
		); err != nil {
			return nil, err
		}
		if soldTo.Valid {
			ap.SoldToTeamRegistrationID = soldTo.String
		}
		if len(notesRaw) > 0 && string(notesRaw) != "null" {
			if err := json.Unmarshal(notesRaw, &ap.Notes); err != nil {
				return nil, err
			}
		}
		out = append(out, &ap)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *AuctionPlayerRepository) ListByAuctionWithPlayerFilter(ctx context.Context, auctionID string, f auction.RegistrationFilter, registrationSerial int, soldToTeamRegistrationID string) ([]*auction.AuctionPlayer, error) {
	if auctionID == "" {
		return nil, fmt.Errorf("auction_id is required")
	}
	needPlayerJoin := f.Gender != "" || f.MinAgeYears > 0 || f.MaxAgeYears > 0
	now := time.Now().UTC().Truncate(24 * time.Hour)
	conds := []string{
		"ap.auction_id = $1",
		"tr.player_id IS NOT NULL",
	}
	args := []any{auctionID}
	argIdx := 2
	if soldToTeamRegistrationID != "" {
		conds = append(conds, "ap.sold_to_team_registration_id = $"+fmt.Sprint(argIdx))
		args = append(args, soldToTeamRegistrationID)
		argIdx++
	}
	if registrationSerial > 0 {
		conds = append(conds, "tr.serial_number = $"+fmt.Sprint(argIdx))
		args = append(args, registrationSerial)
		argIdx++
	}
	if f.Gender != "" {
		conds = append(conds, "p.gender = $"+fmt.Sprint(argIdx))
		args = append(args, f.Gender)
		argIdx++
	}
	if f.MinAgeYears > 0 {
		cutoff := now.AddDate(-f.MinAgeYears, 0, 0)
		conds = append(conds, "p.date_of_birth <= $"+fmt.Sprint(argIdx)+"::date")
		args = append(args, cutoff)
		argIdx++
	}
	if f.MaxAgeYears > 0 {
		cutoff := now.AddDate(-(f.MaxAgeYears + 1), 0, 0)
		conds = append(conds, "p.date_of_birth >= $"+fmt.Sprint(argIdx)+"::date")
		args = append(args, cutoff)
		argIdx++
	}
	from := `FROM auction_players ap
		 INNER JOIN auctions a ON a.id = ap.auction_id
		 INNER JOIN tournament_player_registrations tr
			   ON tr.id = ap.tournament_player_registration_id
			  AND tr.tournament_id = a.tournament_id
			  AND tr.tournament_event_id = a.tournament_event_id`
	if needPlayerJoin {
		from += `
		 INNER JOIN players p ON p.id = tr.player_id`
	}
	query := `SELECT ap.id, ap.auction_id, ap.tournament_player_registration_id,
			 ap.status, ap.notes, ap.base_price, ap.final_price, ap.sold_to_team_registration_id,
			 ap.lot_number, ap.is_active, ap.created_at
		` + from + `
		 WHERE ` + joinConds(conds, " AND ") + `
		 ORDER BY ap.lot_number ASC, ap.created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*auction.AuctionPlayer
	for rows.Next() {
		var ap auction.AuctionPlayer
		var soldTo sql.NullString
		var notesRaw []byte
		if err := rows.Scan(
			&ap.ID, &ap.AuctionID, &ap.TournamentPlayerRegistrationID,
			&ap.Status, &notesRaw, &ap.BasePrice, &ap.FinalPrice, &soldTo,
			&ap.LotNumber, &ap.IsActive, &ap.CreatedAt,
		); err != nil {
			return nil, err
		}
		if soldTo.Valid {
			ap.SoldToTeamRegistrationID = soldTo.String
		}
		if len(notesRaw) > 0 && string(notesRaw) != "null" {
			if err := json.Unmarshal(notesRaw, &ap.Notes); err != nil {
				return nil, err
			}
		}
		out = append(out, &ap)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *AuctionPlayerRepository) UpdateStatus(ctx context.Context, id string, status auction.AuctionPlayerStatus, isActive bool) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE auction_players
		 SET status = $1, is_active = $2
		 WHERE id = $3`,
		string(status), isActive, id,
	)
	return err
}

func (r *AuctionPlayerRepository) MarkSold(ctx context.Context, id string, finalPrice int64, soldToTeamRegistrationID string, notes auction.AuctionPlayerNotes) error {
	notesRaw, err := json.Marshal(notes)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(
		ctx,
		`UPDATE auction_players
		 SET status = $1,
			 notes = $2,
			 final_price = $3,
			 sold_to_team_registration_id = $4,
			 is_active = FALSE
		 WHERE id = $5`,
		string(auction.AuctionPlayerSold), notesRaw, finalPrice, soldToTeamRegistrationID, id,
	)
	return err
}

func (r *AuctionPlayerRepository) ClearSale(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE auction_players
		 SET status = $1,
			 notes = '{}'::jsonb,
			 final_price = 0,
			 sold_to_team_registration_id = NULL,
			 is_active = FALSE
		 WHERE id = $2`,
		string(auction.AuctionPlayerPending), id,
	)
	return err
}

func (r *AuctionPlayerRepository) ResetAllLotsToPending(ctx context.Context, auctionID string) error {
	if auctionID == "" {
		return fmt.Errorf("auction_id is required")
	}
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE auction_players
		 SET status = $1,
			 notes = '{}'::jsonb,
			 final_price = 0,
			 sold_to_team_registration_id = NULL,
			 is_active = FALSE
		 WHERE auction_id = $2`,
		string(auction.AuctionPlayerPending), auctionID,
	)
	return err
}

// --- BidRepository ---

var _ auction.BidRepository = (*BidRepository)(nil)

type BidRepository struct {
	db *sql.DB
}

func NewBidRepository(db *sql.DB) *BidRepository {
	return &BidRepository{db: db}
}

func (r *BidRepository) Create(ctx context.Context, b *auction.Bid) error {
	if b == nil {
		return fmt.Errorf("bid is nil")
	}
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO bids (id, auction_player_id, team_registration_id, amount, recorded_at)
		 VALUES ($1,$2,$3,$4,$5)`,
		b.ID, b.AuctionPlayerID, b.TeamRegistrationID, b.Amount, b.RecordedAt,
	)
	return err
}

func (r *BidRepository) ListByAuctionPlayer(ctx context.Context, auctionPlayerID string) ([]*auction.Bid, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, auction_player_id, team_registration_id, amount, recorded_at
		 FROM bids WHERE auction_player_id = $1
		 ORDER BY recorded_at ASC`,
		auctionPlayerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*auction.Bid
	for rows.Next() {
		var b auction.Bid
		if err := rows.Scan(&b.ID, &b.AuctionPlayerID, &b.TeamRegistrationID, &b.Amount, &b.RecordedAt); err != nil {
			return nil, err
		}
		out = append(out, &b)
	}
	return out, rows.Err()
}

func (r *BidRepository) GetHighestBid(ctx context.Context, auctionPlayerID string) (*auction.Bid, error) {
	var b auction.Bid
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, auction_player_id, team_registration_id, amount, recorded_at
		 FROM bids WHERE auction_player_id = $1
		 ORDER BY amount DESC, recorded_at DESC
		 LIMIT 1`,
		auctionPlayerID,
	).Scan(&b.ID, &b.AuctionPlayerID, &b.TeamRegistrationID, &b.Amount, &b.RecordedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *BidRepository) DeleteLatestByAuctionPlayer(ctx context.Context, auctionPlayerID string) (*auction.Bid, error) {
	var b auction.Bid
	err := r.db.QueryRowContext(
		ctx,
		`DELETE FROM bids
		 WHERE id = (
			SELECT id FROM bids
			WHERE auction_player_id = $1
			ORDER BY recorded_at DESC, amount DESC, id DESC
			LIMIT 1
		 )
		 RETURNING id, auction_player_id, team_registration_id, amount, recorded_at`,
		auctionPlayerID,
	).Scan(&b.ID, &b.AuctionPlayerID, &b.TeamRegistrationID, &b.Amount, &b.RecordedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *BidRepository) DeleteByAuction(ctx context.Context, auctionID string) error {
	if auctionID == "" {
		return fmt.Errorf("auction_id is required")
	}
	_, err := r.db.ExecContext(
		ctx,
		`DELETE FROM bids
		 USING auction_players
		 WHERE bids.auction_player_id = auction_players.id
		   AND auction_players.auction_id = $1`,
		auctionID,
	)
	return err
}

// --- WalletRepository ---

var _ auction.WalletRepository = (*WalletRepository)(nil)

type WalletRepository struct {
	db *sql.DB
}

func NewWalletRepository(db *sql.DB) *WalletRepository {
	return &WalletRepository{db: db}
}

func (r *WalletRepository) GetByID(ctx context.Context, walletID string) (*auction.Wallet, error) {
	if walletID == "" {
		return nil, fmt.Errorf("wallet_id is required")
	}
	var w auction.Wallet
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, tournament_id, tournament_event_id, team_id, balance, max_bid_amount, created_at, updated_at
		 FROM wallets
		 WHERE id = $1`,
		walletID,
	).Scan(&w.ID, &w.TournamentID, &w.TournamentEventID, &w.TeamID, &w.Balance, &w.MaxBidAmount, &w.CreatedAt, &w.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *WalletRepository) GetByTournamentEventTeam(ctx context.Context, tournamentID, tournamentEventID, teamRegistrationID string) (*auction.Wallet, error) {
	var w auction.Wallet
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, tournament_id, tournament_event_id, team_id, balance, max_bid_amount, created_at, updated_at
		 FROM wallets
		 WHERE tournament_id = $1 AND tournament_event_id = $2 AND team_id = $3`,
		tournamentID, tournamentEventID, teamRegistrationID,
	).Scan(&w.ID, &w.TournamentID, &w.TournamentEventID, &w.TeamID, &w.Balance, &w.MaxBidAmount, &w.CreatedAt, &w.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *WalletRepository) UpdateBalance(ctx context.Context, walletID string, newBalance int64, updatedAtMs int64) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE wallets
		 SET balance = $1, updated_at = $2
		 WHERE id = $3`,
		newBalance, updatedAtMs, walletID,
	)
	return err
}

func (r *WalletRepository) ListTransactionsByWalletID(ctx context.Context, walletID string) ([]*auction.WalletTransaction, error) {
	if walletID == "" {
		return nil, fmt.Errorf("wallet_id is required")
	}
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, wallet_id, amount, type, reference_id, wallet_balance, created_at
		 FROM wallet_transactions
		 WHERE wallet_id = $1
		 ORDER BY created_at DESC`,
		walletID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*auction.WalletTransaction
	for rows.Next() {
		var tx auction.WalletTransaction
		if err := rows.Scan(&tx.ID, &tx.WalletID, &tx.Amount, &tx.Type, &tx.ReferenceID, &tx.WalletBalance, &tx.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &tx)
	}
	return out, rows.Err()
}

func (r *WalletRepository) CreateTransaction(ctx context.Context, tx *auction.WalletTransaction) error {
	if tx == nil {
		return fmt.Errorf("wallet transaction is nil")
	}
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO wallet_transactions (id, wallet_id, amount, type, reference_id, wallet_balance, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		tx.ID, tx.WalletID, tx.Amount, string(tx.Type), tx.ReferenceID, tx.WalletBalance, tx.CreatedAt,
	)
	return err
}

func (r *WalletRepository) DeleteAllWalletTransactionsForTournamentEvent(ctx context.Context, tournamentID, tournamentEventID string) error {
	if tournamentID == "" || tournamentEventID == "" {
		return fmt.Errorf("tournament_id and tournament_event_id are required")
	}
	_, err := r.db.ExecContext(
		ctx,
		`DELETE FROM wallet_transactions wt
		 USING wallets w
		 WHERE wt.wallet_id = w.id
		   AND w.tournament_id = $1
		   AND w.tournament_event_id = $2`,
		tournamentID, tournamentEventID,
	)
	return err
}

func (r *WalletRepository) ZeroWalletBalancesForTournamentEvent(ctx context.Context, tournamentID, tournamentEventID string, updatedAtMs int64) error {
	if tournamentID == "" || tournamentEventID == "" {
		return fmt.Errorf("tournament_id and tournament_event_id are required")
	}
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE wallets
		 SET balance = 0, updated_at = $3
		 WHERE tournament_id = $1 AND tournament_event_id = $2`,
		tournamentID, tournamentEventID, updatedAtMs,
	)
	return err
}

func joinConds(conds []string, sep string) string {
	if len(conds) == 0 {
		return ""
	}
	out := conds[0]
	for i := 1; i < len(conds); i++ {
		out += sep + conds[i]
	}
	return out
}
