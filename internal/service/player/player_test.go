package playerservice

import (
	"context"
	"testing"
	"time"

	"github.com/dreamers-be/internal/domain/player"
	"github.com/dreamers-be/internal/mocks"
	"github.com/golang/mock/gomock"
)

func TestPlayerService_Create_ValidationMissingName(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockRepository(ctrl)
	svc := NewPlayerService(repo)

	in := &CreateInput{
		Name:               "",
		ImageURL:           "profile_photo/x.jpg",
		AadharCardImageURL: "aadhar/x.jpg",
		Gender:             player.GenderMale,
		DateOfBirth:        time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
		TNBAID:             "7/0515",
		District:           "Chennai",
		Phone:              "9876543210",
		RecentAchievements: "Achv",
		TshirtSize:         "M",
	}

	_, err := svc.Create(ctx, in)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestPlayerService_Create_DuplicateTNBAID(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockRepository(ctrl)
	svc := NewPlayerService(repo)

	in := &CreateInput{
		Name:               "Test Player",
		ImageURL:           "profile_photo/x.jpg",
		AadharCardImageURL: "aadhar/x.jpg",
		Gender:             player.GenderMale,
		DateOfBirth:        time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
		TNBAID:             "7/0515",
		District:           "Chennai",
		Phone:              "9876543210",
		RecentAchievements: "Achv",
		TshirtSize:         "M",
	}

	repo.EXPECT().ExistsByTNBAID(gomock.Any(), in.TNBAID).Return(true, nil).Times(1)

	p, err := svc.Create(ctx, in)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if p != nil {
		t.Fatalf("expected nil entity, got %#v", p)
	}
}

func TestPlayerService_Create_Success(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockRepository(ctrl)
	svc := NewPlayerService(repo)

	dob := time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC)
	in := &CreateInput{
		Name:               "Test Player",
		ImageURL:           "profile_photo/x.jpg",
		AadharCardImageURL: "aadhar/x.jpg",
		Gender:             player.GenderMale,
		DateOfBirth:        dob,
		TNBAID:             "7/0515",
		District:           "Chennai",
		Phone:              "9876543210",
		RecentAchievements: "Achv",
		TshirtSize:         "M",
	}

	repo.EXPECT().ExistsByTNBAID(gomock.Any(), in.TNBAID).Return(false, nil).Times(1)

	var created *player.Entity
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, p *player.Entity) error {
			created = p
			return nil
		},
	).Times(1)

	p, err := svc.Create(ctx, in)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if created == nil {
		t.Fatalf("expected repo.Create called with entity")
	}
	if p.TNBAID != "7/0515" {
		t.Fatalf("TNBAID = %q, want %q", p.TNBAID, "7/0515")
	}
	if p.Phone != "9876543210" {
		t.Fatalf("Phone = %q, want %q", p.Phone, "9876543210")
	}
	if p.DateOfBirth.Format("2006-01-02") != "1990-05-15" {
		t.Fatalf("DateOfBirth = %q", p.DateOfBirth.Format("2006-01-02"))
	}
	if p.CreatedAt.IsZero() {
		t.Fatalf("CreatedAt should be set")
	}
}

