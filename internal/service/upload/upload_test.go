package uploadservice

import (
	"bytes"
	"context"
	"testing"

	"github.com/dreamers-be/internal/mocks"
	"github.com/golang/mock/gomock"
)

func TestUploadService_Upload_Success(t *testing.T) {
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uploader := mocks.NewMockFileUploader(ctrl)
	svc := NewUploadService(uploader, 1)

	data := []byte("hello")

	var gotFilename, gotContentType, gotFolder string
	var gotData []byte
	uploader.EXPECT().Upload(
		gomock.Any(),
		"file.jpg",
		gomock.Any(),
		"text/plain",
		"profile_photo",
	).DoAndReturn(func(ctx context.Context, filename string, data []byte, contentType, folder string) (string, error) {
		gotFilename = filename
		gotContentType = contentType
		gotFolder = folder
		gotData = append([]byte(nil), data...)
		return "profile_photo/file.jpg", nil
	}).Times(1)

	key, err := svc.Upload(ctx, "file.jpg", bytes.NewReader(data), "text/plain", "profile_photo")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if key != "profile_photo/file.jpg" {
		t.Fatalf("key = %q, want %q", key, "profile_photo/file.jpg")
	}
	if gotFolder != "profile_photo" {
		t.Fatalf("folder = %q, want %q", gotFolder, "profile_photo")
	}
	if gotFilename != "file.jpg" {
		t.Fatalf("filename = %q, want %q", gotFilename, "file.jpg")
	}
	if gotContentType != "text/plain" {
		t.Fatalf("contentType = %q, want %q", gotContentType, "text/plain")
	}
	if string(gotData) != "hello" {
		t.Fatalf("data = %q, want %q", string(gotData), "hello")
	}
}

func TestUploadService_Upload_SizeLimitExceeded(t *testing.T) {
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	uploader := mocks.NewMockFileUploader(ctrl)
	svc := NewUploadService(uploader, 1) // 1 MB

	maxBytes := int64(1) * 1024 * 1024
	data := bytes.Repeat([]byte{'a'}, int(maxBytes)+1)

	_, err := svc.Upload(ctx, "file.jpg", bytes.NewReader(data), "text/plain", "uploads")
	if err == nil {
		t.Fatalf("expected error")
	}
	// If uploader.Upload() was called, gomock would fail the test due to missing EXPECT().
}

