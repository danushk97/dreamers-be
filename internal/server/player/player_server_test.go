package playerserver

import (
	"context"
	"testing"
	"time"

	"github.com/dreamers-be/internal/domain/player"
	"github.com/dreamers-be/internal/mocks"
	playersvc "github.com/dreamers-be/internal/service/player"
	"github.com/golang/mock/gomock"
)

func TestPlayerServer_Create_SanitizesAndCallsService(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockRepository(ctrl)
	svc := playersvc.NewPlayerService(repo)
	srv := NewPlayerServer(svc)

	repo.EXPECT().ExistsByTNBAID(gomock.Any(), "7/0515").Return(false, nil).Times(1)

	var created *player.Entity
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, p *player.Entity) error {
			created = p
			return nil
		},
	).Times(1)

	req := &CreateRequest{
		Name:               "  Test Player  ",
		ImageURL:           " profile_photo/abc.jpg ",
		AadharCardImageURL: " aadhar/xyz.jpg ",
		Gender:             "MALE",
		DateOfBirth:        "1990-05-15",
		TNBAID:             "TNBA/7/0515",
		District:           "Chennai",
		Phone:              "+91 9876543210",
		RecentAchievements: "  Achv  ",
		TshirtSize:         "M",
	}

	p, err := srv.Create(ctx, req)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if created == nil || p == nil {
		t.Fatalf("expected entity to be created")
	}

	if p.Name != "Test Player" {
		t.Fatalf("Name = %q, want %q", p.Name, "Test Player")
	}
	if p.ImageURL != "profile_photo/abc.jpg" {
		t.Fatalf("ImageURL = %q", p.ImageURL)
	}
	if p.AadharCardImageURL != "aadhar/xyz.jpg" {
		t.Fatalf("AadharCardImageURL = %q", p.AadharCardImageURL)
	}
	if p.Phone != "9876543210" {
		t.Fatalf("Phone = %q, want %q", p.Phone, "9876543210")
	}
	if p.TNBAID != "7/0515" {
		t.Fatalf("TNBAID = %q, want %q", p.TNBAID, "7/0515")
	}
	if p.District != "Chennai" {
		t.Fatalf("District = %q, want %q", p.District, "Chennai")
	}
	if p.RecentAchievements != "Achv" {
		t.Fatalf("RecentAchievements = %q, want %q", p.RecentAchievements, "Achv")
	}
	if p.TshirtSize != "M" {
		t.Fatalf("TshirtSize = %q, want %q", p.TshirtSize, "M")
	}
	if p.DateOfBirth.Format("2006-01-02") != (time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC)).Format("2006-01-02") {
		t.Fatalf("DateOfBirth = %q", p.DateOfBirth.Format("2006-01-02"))
	}
}

func TestPlayerServer_Create_InvalidDOB_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockRepository(ctrl)
	svc := playersvc.NewPlayerService(repo)
	srv := NewPlayerServer(svc)

	req := &CreateRequest{
		Name:               "Test Player",
		ImageURL:           "profile_photo/abc.jpg",
		AadharCardImageURL: "aadhar/xyz.jpg",
		Gender:             "MALE",
		DateOfBirth:        "not-a-date",
		TNBAID:             "7/0515",
		District:           "Chennai",
		Phone:              "9876543210",
		RecentAchievements: "Achv",
		TshirtSize:         "M",
	}

	_, err := srv.Create(ctx, req)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !playersvc.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestPlayerServer_Get_EmptyID_ReturnsValidationError(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mocks.NewMockRepository(ctrl)
	svc := playersvc.NewPlayerService(repo)
	srv := NewPlayerServer(svc)

	_, err := srv.Get(ctx, "   ")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !playersvc.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}
