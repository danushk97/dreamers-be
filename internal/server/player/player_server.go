package playerserver

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dreamers-be/internal/domain/player"
	"github.com/dreamers-be/internal/pkg/sanitize"
	playersvc "github.com/dreamers-be/internal/service/player"
)

// CreateRequest represents the JSON body for player registration.
type CreateRequest struct {
	Name               string `json:"name"`
	ImageURL           string `json:"imageURL"`
	AadharCardImageURL string `json:"aadharCardImageURL"`
	Gender             string `json:"gender"`
	DateOfBirth        string `json:"dateOfBirth"`
	TNBAID             string `json:"tnbaId"`
	District           string `json:"district"`
	Phone              any    `json:"phone"` // number or string from JSON
	RecentAchievements string `json:"recentAchievements"`
	TshirtSize         string `json:"tshirtSize"`
}

// ListQuery represents GET query params for listing players.
type ListQuery struct {
	Name      string
	TNBAID    string
	Gender    string
	AgeFilter string
	Page      string
	Limit     string
}

// PlayerServer is a thin application layer that translates HTTP-ish inputs into service inputs.
type PlayerServer struct {
	playerService *playersvc.PlayerService
}

// NewPlayerServer returns a new PlayerServer.
func NewPlayerServer(
	playerService *playersvc.PlayerService,
) *PlayerServer {
	return &PlayerServer{playerService: playerService}
}

// Create creates a new player from already-bound request fields.
func (s *PlayerServer) Create(ctx context.Context, req *CreateRequest) (*player.Entity, error) {
	if req == nil {
		return nil, &playersvc.ValidationError{Err: fmt.Errorf("invalid request body")}
	}

	dob, err := time.Parse("2006-01-02", sanitize.String(req.DateOfBirth))
	if err != nil {
		return nil, &playersvc.ValidationError{Err: fmt.Errorf("invalid date of birth (use YYYY-MM-DD)")}
	}

	phoneStr := ""
	switch v := req.Phone.(type) {
	case float64:
		phoneStr = strconv.FormatInt(int64(v), 10)
	case string:
		phoneStr = v
	default:
		phoneStr = ""
	}

	// imageURL / aadharCardImageURL: S3 key (uploads/...) or external URL.
	imageKey := strings.TrimSpace(req.ImageURL)
	aadharKey := strings.TrimSpace(req.AadharCardImageURL)

	in := &playersvc.CreateInput{
		Name:               sanitize.String(req.Name),
		ImageURL:           imageKey,
		AadharCardImageURL: aadharKey,
		Gender:             sanitize.OneOf(req.Gender, []string{player.GenderMale, player.GenderFemale}),
		DateOfBirth:        dob,
		TNBAID:             sanitize.TNBAID(req.TNBAID),
		District:           sanitize.OneOf(req.District, player.TamilNaduDistricts),
		Phone:              sanitize.Phone(phoneStr),
		RecentAchievements: sanitize.MaxLen(sanitize.String(req.RecentAchievements), 300),
		TshirtSize:         sanitize.OneOf(req.TshirtSize, player.ValidTshirtSizes),
	}

	if s.playerService == nil {
		return nil, fmt.Errorf("player service not configured")
	}
	return s.playerService.Create(ctx, in)
}

// List lists players with filters from query params.
func (s *PlayerServer) List(ctx context.Context, q *ListQuery) (*player.ListResult, error) {
	if q == nil {
		if s.playerService == nil {
			return nil, fmt.Errorf("list service not configured")
		}
		return s.playerService.List(ctx, nil)
	}

	page := 0
	if v, err := strconv.Atoi(strings.TrimSpace(q.Page)); err == nil {
		page = v
	}

	limit := 20
	if v, err := strconv.Atoi(strings.TrimSpace(q.Limit)); err == nil {
		limit = v
	}

	f := &player.ListFilter{
		Name:      sanitize.String(q.Name),
		TNBAID:    sanitize.String(q.TNBAID),
		Gender:    sanitize.OneOf(q.Gender, []string{player.GenderMale, player.GenderFemale}),
		AgeFilter: sanitize.OneOf(q.AgeFilter, []string{"all", "below-30", "31-40", "41-50", "50+", "above-30"}),
		Page:      page,
		Limit:     limit,
	}

	if s.playerService == nil {
		return nil, fmt.Errorf("player service not configured")
	}
	return s.playerService.List(ctx, f)
}

// Get returns a single player by ID.
func (s *PlayerServer) Get(ctx context.Context, id string) (*player.Entity, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, &playersvc.ValidationError{Err: fmt.Errorf("player ID required")}
	}
	if s.playerService == nil {
		return nil, fmt.Errorf("player service not configured")
	}
	return s.playerService.Get(ctx, id)
}
