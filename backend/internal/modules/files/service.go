package files

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/Sidd231/4yaar/backend/internal/config"
	"github.com/Sidd231/4yaar/backend/internal/models"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/uuid"
)

const (
	MaxFileSize int64 = 50 * 1024 * 1024 // 50MB
)

var (
	ErrFileTooLarge    = errors.New("file exceeds maximum allowed size of 50MB")
	ErrUnsupportedType = errors.New("unsupported file MIME type")
	ErrFileNotFound    = errors.New("file not found")
	ErrUnauthorized    = errors.New("unauthorized to delete this file")
)

var allowedMimeTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/gif":       true,
	"image/webp":      true,
	"application/pdf": true,
	"text/plain":      true,
	"video/mp4":       true,
}

// UploadResponse returns the persisted file record along with its presigned download URL.
type UploadResponse struct {
	File         *models.File `json:"file"`
	PresignedURL string       `json:"presigned_url"`
}

// Service defines S3 file operations and metadata management.
type Service interface {
	UploadFile(ctx context.Context, uploaderID, roomID uuid.UUID, fileName, mimeType string, fileSize int64, reader io.Reader) (*UploadResponse, error)
	GetPresignedURL(ctx context.Context, fileID, requesterID uuid.UUID) (string, error)
	DeleteFile(ctx context.Context, fileID, requesterID uuid.UUID, requesterRole string) error
}

type service struct {
	repo          Repository
	s3Client      *s3.Client
	presignClient *s3.PresignClient
	bucketName    string
}

// NewService initializes AWS S3 client and the files service.
func NewService(repo Repository, cfg *config.Config) (Service, error) {
	var optFns []func(*awsconfig.LoadOptions) error
	optFns = append(optFns, awsconfig.WithRegion(cfg.AWSRegion))

	if cfg.AWSAccessKeyID != "" && cfg.AWSSecretAccessKey != "" {
		optFns = append(optFns, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AWSAccessKeyID, cfg.AWSSecretAccessKey, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), optFns...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for S3: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)
	presignClient := s3.NewPresignClient(client)

	return &service{
		repo:          repo,
		s3Client:      client,
		presignClient: presignClient,
		bucketName:    cfg.S3Bucket,
	}, nil
}

// UploadFile validates the file, streams it to AWS S3, saves metadata in PostgreSQL, and creates a presigned GET URL.
func (s *service) UploadFile(ctx context.Context, uploaderID, roomID uuid.UUID, fileName, mimeType string, fileSize int64, reader io.Reader) (*UploadResponse, error) {
	// Enforce room membership before processing file
	isMember, err := s.repo.IsUserInRoom(ctx, roomID, uploaderID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify room membership: %w", err)
	}
	if !isMember {
		return nil, ErrUnauthorized
	}

	if fileSize > MaxFileSize {
		return nil, ErrFileTooLarge
	}

	cleanMime := strings.ToLower(strings.TrimSpace(mimeType))
	if !allowedMimeTypes[cleanMime] {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedType, cleanMime)
	}

	fileID := uuid.New()
	ext := filepath.Ext(fileName)
	s3Key := fmt.Sprintf("rooms/%s/files/%s%s", roomID.String(), fileID.String(), ext)

	// Read file into buffer or stream to S3
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	_, err = s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucketName),
		Key:           aws.String(s3Key),
		Body:          bytes.NewReader(data),
		ContentType:   aws.String(cleanMime),
		ContentLength: aws.Int64(int64(len(data))),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload object to S3: %w", err)
	}

	fileRecord := &models.File{
		ID:         fileID,
		UploaderID: uploaderID,
		RoomID:     roomID,
		S3Key:      s3Key,
		FileName:   fileName,
		FileSize:   int64(len(data)),
		MimeType:   cleanMime,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.CreateFile(ctx, fileRecord); err != nil {
		// Attempt rollback on S3 if metadata creation fails
		_, _ = s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(s.bucketName),
			Key:    aws.String(s3Key),
		})
		return nil, fmt.Errorf("failed to save file record: %w", err)
	}

	// Generate 15-minute presigned GET URL
	presignedURL, err := s.generatePresignedURL(ctx, s3Key, 15*time.Minute)
	if err != nil {
		slog.Warn("Could not generate presigned URL after upload", "file_id", fileID, "error", err)
	}

	return &UploadResponse{
		File:         fileRecord,
		PresignedURL: presignedURL,
	}, nil
}

// GetPresignedURL fetches an active file by ID, verifies room membership of the requester, and returns a fresh 15-minute presigned GET URL.
func (s *service) GetPresignedURL(ctx context.Context, fileID, requesterID uuid.UUID) (string, error) {
	file, err := s.repo.GetFileByID(ctx, fileID)
	if err != nil {
		return "", fmt.Errorf("database query failed: %w", err)
	}
	if file == nil {
		return "", ErrFileNotFound
	}

	// Any room member can access files in that room
	isMember, err := s.repo.IsUserInRoom(ctx, file.RoomID, requesterID)
	if err != nil {
		return "", fmt.Errorf("failed to verify room membership: %w", err)
	}
	if !isMember {
		return "", ErrUnauthorized
	}

	return s.generatePresignedURL(ctx, file.S3Key, 15*time.Minute)
}

// DeleteFile soft deletes the file from DB and deletes the underlying S3 object.
func (s *service) DeleteFile(ctx context.Context, fileID, requesterID uuid.UUID, requesterRole string) error {
	file, err := s.repo.GetFileByID(ctx, fileID)
	if err != nil {
		return fmt.Errorf("failed finding file: %w", err)
	}
	if file == nil {
		return ErrFileNotFound
	}

	// Allow deletion only if requester is the uploader or an admin
	if file.UploaderID != requesterID && requesterRole != models.RoleAdmin {
		return ErrUnauthorized
	}

	// Soft delete in DB
	if err := s.repo.SoftDeleteFile(ctx, fileID); err != nil {
		return fmt.Errorf("failed to soft delete file record: %w", err)
	}

	// Remove from S3 asynchronously or synchronously
	_, err = s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(file.S3Key),
	})
	if err != nil {
		slog.Error("Failed to delete object from S3 during file delete", "s3_key", file.S3Key, "error", err)
	}

	return nil
}

// generatePresignedURL signs an S3 GetObject request for the specified lifetime.
func (s *service) generatePresignedURL(ctx context.Context, s3Key string, expiry time.Duration) (string, error) {
	req, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(s3Key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned GET url: %w", err)
	}
	return req.URL, nil
}
