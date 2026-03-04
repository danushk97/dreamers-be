package player

import (
	"context"
	"testing"
	"time"

	"github.com/dreamers-be/internal/domain/player"
	"github.com/dreamers-be/internal/domain/storage"
)

type mockRepo struct {
	createErr      error
	created       *player.Entity
	tnbaIDExists  bool
}

func (m *mockRepo) Create(ctx context.Context, p *player.Entity) error {
	m.created = p
	return m.createErr
}

func (m *mockRepo) List(ctx context.Context, f *player.ListFilter) (*player.ListResult, error) {
	return nil, nil
}

func (m *mockRepo) ExistsByTNBAID(ctx context.Context, tnbaID string) (bool, error) {
	return m.tnbaIDExists, nil
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (*player.Entity, error) {
	return nil, nil
}

type mockUploader struct {
	url string
}

func (m *mockUploader) Upload(ctx context.Context, filename string, data []byte, contentType string, folder string) (string, error) {
	return m.url, nil
}

var _ storage.FileUploader = (*mockUploader)(nil)

func TestCreateUseCase_Create(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	uploader := &mockUploader{url: "profile_photo/2024/01/02/photo-abc123.jpg"}
	uc := NewCreateUseCase(repo, uploader)

	dob := time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC)
	in := &CreateInput{
		Name:               "Test Player",
		ImageURL:           "profile_photo/2024/01/02/photo-abc123.jpg",
		AadharCardImageURL: "aadhar/2024/01/02/aadhar-xyz456.jpg",
		Gender:             "MALE",
		DateOfBirth:        dob,
		TNBAID:             "TNBA/7/0515",
		District:           "Chennai",
		Phone:              "9876543210",
		TshirtSize:         "M",
	}

	p, err := uc.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create() err = %v", err)
	}
	if p.ID == "" {
		t.Error("ID should be set")
	}
	if p.Name != "Test Player" {
		t.Errorf("Name = %q", p.Name)
	}
	if p.TNBAID != "7/0515" {
		t.Errorf("TNBAID should be stored as 7/0515, got %q", p.TNBAID)
	}
	if repo.created == nil {
		t.Fatal("repo.Create was not called")
	}
}

func TestCreateUseCase_Validation(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{}
	uploader := &mockUploader{}
	uc := NewCreateUseCase(repo, uploader)

	tests := []struct {
		name string
		in   *CreateInput
	}{
		{"missing name", &CreateInput{ImageURL: "profile_photo/x.jpg", AadharCardImageURL: "aadhar/x.jpg", Gender: "MALE", DateOfBirth: time.Now(), TNBAID: "7/0515", District: "Chennai", Phone: "9876543210", TshirtSize: "M"}},
		{"invalid phone", &CreateInput{Name: "X", ImageURL: "profile_photo/x.jpg", AadharCardImageURL: "aadhar/x.jpg", Gender: "MALE", DateOfBirth: time.Now(), TNBAID: "7/0515", District: "Chennai", Phone: "123", TshirtSize: "M"}},
		{"invalid district", &CreateInput{Name: "X", ImageURL: "profile_photo/x.jpg", AadharCardImageURL: "aadhar/x.jpg", Gender: "MALE", DateOfBirth: time.Now(), TNBAID: "7/0515", District: "InvalidDistrict", Phone: "9876543210", TshirtSize: "M"}},
		{"invalid gender", &CreateInput{Name: "X", ImageURL: "profile_photo/x.jpg", AadharCardImageURL: "aadhar/x.jpg", Gender: "X", DateOfBirth: time.Now(), TNBAID: "7/0515", District: "Chennai", Phone: "9876543210", TshirtSize: "M"}},
		{"invalid tnba id", &CreateInput{Name: "X", ImageURL: "profile_photo/x.jpg", AadharCardImageURL: "aadhar/x.jpg", Gender: "MALE", DateOfBirth: time.Now(), TNBAID: "TNBA123", District: "Chennai", Phone: "9876543210", TshirtSize: "M"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uc.Create(ctx, tt.in)
			if err == nil {
				t.Error("expected validation error")
		}
	})
	}
}

func TestCreateUseCase_DuplicateTNBAID(t *testing.T) {
	ctx := context.Background()
	repo := &mockRepo{tnbaIDExists: true}
	uploader := &mockUploader{url: "profile_photo/x.jpg"}
	uc := NewCreateUseCase(repo, uploader)

	in := &CreateInput{
		Name:               "Test Player",
		ImageURL:           "profile_photo/x.jpg",
		AadharCardImageURL: "aadhar/x.jpg",
		Gender:             "MALE",
		DateOfBirth:        time.Now(),
		TNBAID:             "7/0515",
		District:           "Chennai",
		Phone:              "9876543210",
		TshirtSize:         "M",
	}

	_, err := uc.Create(ctx, in)
	if err == nil {
		t.Fatal("expected validation error for duplicate tnba id")
	}
	if !IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
	if repo.created != nil {
		t.Error("Create should not have been called")
	}
}
