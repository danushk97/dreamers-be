package player

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/dreamers-be/internal/domain/player"
	"github.com/dreamers-be/internal/domain/storage"
	"github.com/dreamers-be/internal/pkg/sanitize"
)

// ValidationError indicates a validation failure (client fault, 400).
type ValidationError struct{ Err error }

func (e *ValidationError) Error() string { return e.Err.Error() }
func (e *ValidationError) Unwrap() error { return e.Err }

// IsValidationError returns true if err is a ValidationError.
func IsValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}

// CreateInput holds validated input for creating a player.
type CreateInput struct {
	Name               string
	ImageURL           string // Pre-uploaded URL from storage
	AadharCardImageURL string
	Gender             string
	DateOfBirth        time.Time
	TNBAID             string
	District           string
	Phone              string
	RecentAchievements string
	TshirtSize         string
}

// CreateUseCase handles player registration.
type CreateUseCase struct {
	repo   player.Repository
	upload storage.FileUploader
}

// NewCreateUseCase returns a new create player use case.
func NewCreateUseCase(repo player.Repository, upload storage.FileUploader) *CreateUseCase {
	return &CreateUseCase{repo: repo, upload: upload}
}

// Create registers a new player. Image URLs must be provided (from upload step).
func (uc *CreateUseCase) Create(ctx context.Context, in *CreateInput) (*player.Entity, error) {
	if err := uc.validate(in); err != nil {
		return nil, &ValidationError{Err: err}
	}

	p := &player.Entity{
		ID:                 uuid.New().String(),
		Name:               sanitize.String(in.Name),
		ImageURL:           in.ImageURL,
		Gender:             in.Gender,
		DateOfBirth:        in.DateOfBirth,
		TNBAID:             sanitize.String(sanitize.AlphanumericID(in.TNBAID)),
		District:           sanitize.OneOf(in.District, player.TamilNaduDistricts),
		Phone:              sanitize.Phone(in.Phone),
		RecentAchievements: sanitize.MaxLen(sanitize.String(in.RecentAchievements), 300),
		TshirtSize:         sanitize.OneOf(in.TshirtSize, player.ValidTshirtSizes),
		AadharCardImageURL: in.AadharCardImageURL,
		CreatedAt:          time.Now(),
	}

	if err := uc.repo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("create player: %w", err)
	}
	return p, nil
}

func (uc *CreateUseCase) validate(in *CreateInput) error {
	if in.Name == "" {
		return fmt.Errorf("name is required")
	}
	if in.ImageURL == "" {
		return fmt.Errorf("profile photo is required")
	}
	if in.AadharCardImageURL == "" {
		return fmt.Errorf("aadhar card image is required")
	}
	if g := sanitize.OneOf(in.Gender, []string{player.GenderMale, player.GenderFemale}); g == "" {
		return fmt.Errorf("invalid gender")
	}
	if in.DateOfBirth.IsZero() {
		return fmt.Errorf("date of birth is required")
	}
	if in.TNBAID == "" {
		return fmt.Errorf("tnba id is required")
	}
	if sanitize.OneOf(in.District, player.TamilNaduDistricts) == "" {
		return fmt.Errorf("invalid district")
	}
	if len(sanitize.Phone(in.Phone)) != 10 {
		return fmt.Errorf("phone must be 10 digits")
	}
	if sanitize.OneOf(in.TshirtSize, player.ValidTshirtSizes) == "" {
		return fmt.Errorf("invalid tshirt size")
	}
	return nil
}
