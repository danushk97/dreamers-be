package uploadserver

import (
	"bytes"
	"context"
	"testing"

	"github.com/dreamers-be/internal/domain/storage"
	"github.com/dreamers-be/internal/mocks"
	uploadsvc "github.com/dreamers-be/internal/service/upload"
	"github.com/golang/mock/gomock"
)

func TestUploadServer_NormalizesProfilePhotoType(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uploader := mocks.NewMockFileUploader(ctrl)
	svc := uploadsvc.NewUploadService(uploader, 1)
	srv := NewUploadServer(svc)

	uploader.EXPECT().Upload(
		gomock.Any(),
		"file.jpg",
		gomock.Any(),
		"text/plain",
		storage.FolderProfilePhoto,
	).Return("uploaded-key", nil).Times(1)

	key, err := srv.Upload(ctx, "file.jpg", bytes.NewReader([]byte("hi")), "text/plain", "Profile-Photo", "", "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if key != "uploaded-key" {
		t.Fatalf("key = %q", key)
	}
}

func TestUploadServer_NormalizesAadharFromFolderField(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uploader := mocks.NewMockFileUploader(ctrl)
	svc := uploadsvc.NewUploadService(uploader, 1)
	srv := NewUploadServer(svc)

	uploader.EXPECT().Upload(
		gomock.Any(),
		"file.jpg",
		gomock.Any(),
		"text/plain",
		storage.FolderAadhar,
	).Return("uploaded-key", nil).Times(1)

	_, err := srv.Upload(ctx, "file.jpg", bytes.NewReader([]byte("hi")), "text/plain", "", "aadhar_card", "")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestUploadServer_DefaultsToUploads(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uploader := mocks.NewMockFileUploader(ctrl)
	svc := uploadsvc.NewUploadService(uploader, 1)
	srv := NewUploadServer(svc)

	uploader.EXPECT().Upload(
		gomock.Any(),
		"file.jpg",
		gomock.Any(),
		"text/plain",
		"uploads",
	).Return("uploaded-key", nil).Times(1)

	_, err := srv.Upload(ctx, "file.jpg", bytes.NewReader([]byte("hi")), "text/plain", "", "", "unknown")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
}

