package s3

import (
	"bytes"
	"context"
	"fmt"
	"mime"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"

	"github.com/dreamers-be/internal/domain/storage"
)

var (
	_ storage.FileUploader = (*FileUploader)(nil)
	_ storage.Presigner    = (*FileUploader)(nil)
)

// FileUploader uploads files to S3 and returns public URLs.
type FileUploader struct {
	client  *s3.Client
	bucket  string
	region  string
	baseURL string // optional custom base URL (e.g. CloudFront)
	maxSize int64
}

// Config holds S3 uploader configuration.
type Config struct {
	Bucket    string
	Region    string
	BaseURL   string // optional, e.g. https://cdn.example.com
	MaxSizeMB int64
	AccessKey string // optional, uses env/default creds if empty
	SecretKey string // required if AccessKey is set
}

// NewFileUploader creates a new S3 file uploader.
func NewFileUploader(ctx context.Context, cfg Config) (*FileUploader, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("s3 bucket is required")
	}
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	if cfg.MaxSizeMB <= 0 {
		cfg.MaxSizeMB = 2
	}

	optFns := []func(*config.LoadOptions) error{
		config.WithRetryer(func() aws.Retryer {
			return retry.AddWithMaxAttempts(retry.NewStandard(), 3)
		}),
	}
	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		optFns = append(optFns, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKey, cfg.SecretKey, "",
		)))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.Region = cfg.Region
	})

	return &FileUploader{
		client:  client,
		bucket:  cfg.Bucket,
		region:  cfg.Region,
		baseURL: cfg.BaseURL,
		maxSize: cfg.MaxSizeMB * 1024 * 1024,
	}, nil
}

// Upload uploads data to S3 as private and returns the object key.
// folder: "profile_photo" or "aadhar"; empty defaults to "uploads".
func (u *FileUploader) Upload(ctx context.Context, filename string, data []byte, contentType string, folder string) (string, error) {
	if int64(len(data)) > u.maxSize {
		return "", fmt.Errorf("file exceeds max size of %d MB", u.maxSize/1024/1024)
	}

	if contentType == "" {
		contentType = mime.TypeByExtension(path.Ext(filename))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	if folder == "" {
		folder = "uploads"
	}

	ext := path.Ext(filename)
	safe := sanitizeKey(filename)
	key := fmt.Sprintf("%s/%s/%s-%s%s", folder, time.Now().Format("2006/01/02"), strings.TrimSuffix(safe, ext), uuid.New().String()[:8], ext)

	input := &s3.PutObjectInput{
		Bucket:      aws.String(u.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
		ACL:         types.ObjectCannedACLPrivate,
	}

	_, err := u.client.PutObject(ctx, input)
	if err != nil {
		return "", fmt.Errorf("s3 put object: %w", err)
	}

	return key, nil
}

// Presign generates a presigned GET URL for the object.
func (u *FileUploader) Presign(ctx context.Context, key string, expiry time.Duration) (string, error) {
	presigner := s3.NewPresignClient(u.client)
	req, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("presign: %w", err)
	}
	return req.URL, nil
}

func sanitizeKey(filename string) string {
	filename = path.Base(filename)
	var b strings.Builder
	for _, r := range filename {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	s := b.String()
	if s == "" {
		return "file"
	}
	return s
}
