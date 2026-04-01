package playerservice

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/dreamers-be/internal/domain/player"
)

// CreateInput holds validated input for creating a player.
//
// Caller is expected to provide already-clean/sanitized fields (e.g. phone digits,
// TNBAID normalization, valid enums).
type CreateInput struct {
	Name               string
	ImageURL           string
	AadharCardImageURL string
	Gender             string
	DateOfBirth        time.Time
	TNBAID             string
	District           string
	Phone              string
	RecentAchievements string
	TshirtSize         string
}

// PlayerService handles player registration business rules.
type PlayerService struct {
	repo player.Repository
}

func NewPlayerService(repo player.Repository) *PlayerService {
	return &PlayerService{repo: repo}
}

// Create registers a new player.
func (uc *PlayerService) Create(ctx context.Context, in *CreateInput) (*player.Entity, error) {
	if err := uc.validate(in); err != nil {
		return nil, &ValidationError{Err: err}
	}

	exists, err := uc.repo.ExistsByTNBAID(ctx, in.TNBAID)
	if err != nil {
		log.Printf("Create player: ExistsByTNBAID error tnbaId=%s: %v", in.TNBAID, err)
		return nil, fmt.Errorf("check tnba id: %w", err)
	}
	if exists {
		log.Printf("Create player: duplicate tnbaId=%s", in.TNBAID)
		return nil, &ValidationError{Err: fmt.Errorf("player already registered")}
	}

	p := &player.Entity{
		ID:                 uuid.New().String(),
		Name:               in.Name,
		ImageURL:           in.ImageURL,
		Gender:             in.Gender,
		DateOfBirth:        in.DateOfBirth,
		TNBAID:             in.TNBAID,
		District:           in.District,
		Phone:              in.Phone,
		RecentAchievements: in.RecentAchievements,
		TshirtSize:         in.TshirtSize,
		AadharCardImageURL: in.AadharCardImageURL,
		CreatedAt:          time.Now().UnixMilli(),
	}

	if err := uc.repo.Create(ctx, p); err != nil {
		log.Printf("Create player: repo.Create error id=%s: %v", p.ID, err)
		return nil, fmt.Errorf("create player: %w", err)
	}

	return p, nil
}

func (uc *PlayerService) validate(in *CreateInput) error {
	if in == nil {
		return errors.New("invalid request")
	}
	if in.Name == "" {
		return fmt.Errorf("name is required")
	}
	if in.ImageURL == "" {
		return fmt.Errorf("profile photo is required")
	}
	if in.AadharCardImageURL == "" {
		return fmt.Errorf("aadhar card image is required")
	}
	if in.Gender != player.GenderMale && in.Gender != player.GenderFemale {
		return fmt.Errorf("invalid gender")
	}
	if in.DateOfBirth.IsZero() {
		return fmt.Errorf("date of birth is required")
	}
	// TNBAID normalization/format validation is expected from server input,
	// but we keep the same regex as a safety net.
	if !isValidTNBAID(in.TNBAID) {
		return fmt.Errorf("tnba id is required and must match format number/number (e.g. 7/0515)")
	}
	if !isValidDistrict(in.District) {
		return fmt.Errorf("invalid district")
	}
	if len(in.Phone) != 10 {
		return fmt.Errorf("phone must be 10 digits")
	}
	if !isValidTshirtSize(in.TshirtSize) {
		return fmt.Errorf("invalid tshirt size")
	}
	return nil
}

func isValidTNBAID(v string) bool {
	// Keep behavior close to sanitize.TNBAID: digits/digits, case-insensitive.
	// We assume server already trimmed it, so just validate format.
	if v == "" {
		return false
	}
	// Simple format check without allocations/regex dependencies.
	// Expected shape: "<digits>/<digits>"
	slash := -1
	for i := 0; i < len(v); i++ {
		if v[i] == '/' {
			slash = i
			break
		}
	}
	if slash <= 0 || slash >= len(v)-1 {
		return false
	}
	for i := 0; i < slash; i++ {
		if v[i] < '0' || v[i] > '9' {
			return false
		}
	}
	for i := slash + 1; i < len(v); i++ {
		if v[i] < '0' || v[i] > '9' {
			return false
		}
	}
	return true
}

func isValidDistrict(v string) bool {
	for _, d := range player.TamilNaduDistricts {
		if d == v {
			return true
		}
	}
	return false
}

func isValidTshirtSize(v string) bool {
	for _, s := range player.ValidTshirtSizes {
		if s == v {
			return true
		}
	}
	return false
}

