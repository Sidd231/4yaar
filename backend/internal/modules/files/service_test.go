package files

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Sidd231/4yaar/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type mockFilesRepo struct {
	files       map[uuid.UUID]*models.File
	roomMembers map[string]bool // key: roomID:userID
}

func newMockFilesRepo() *mockFilesRepo {
	return &mockFilesRepo{
		files:       make(map[uuid.UUID]*models.File),
		roomMembers: make(map[string]bool),
	}
}

func (m *mockFilesRepo) CreateFile(ctx context.Context, file *models.File) error {
	m.files[file.ID] = file
	return nil
}

func (m *mockFilesRepo) GetFileByID(ctx context.Context, id uuid.UUID) (*models.File, error) {
	f, ok := m.files[id]
	if !ok || f.DeletedAt.Valid {
		return nil, nil
	}
	return f, nil
}

func (m *mockFilesRepo) SoftDeleteFile(ctx context.Context, id uuid.UUID) error {
	f, ok := m.files[id]
	if !ok {
		return errors.New("file not found")
	}
	f.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
	return nil
}

func (m *mockFilesRepo) IsUserInRoom(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	key := roomID.String() + ":" + userID.String()
	return m.roomMembers[key], nil
}

func (m *mockFilesRepo) addMember(roomID, userID uuid.UUID) {
	key := roomID.String() + ":" + userID.String()
	m.roomMembers[key] = true
}

func TestFilesService_UploadFile_Authorization(t *testing.T) {
	repo := newMockFilesRepo()
	svc := &service{repo: repo}

	roomID := uuid.New()
	memberID := uuid.New()
	nonMemberID := uuid.New()

	repo.addMember(roomID, memberID)

	// Negative test: non-member cannot upload to room
	data := []byte("dummy file content")
	_, err := svc.UploadFile(
		context.Background(),
		nonMemberID,
		roomID,
		"test.png",
		"image/png",
		int64(len(data)),
		bytes.NewReader(data),
	)
	if err == nil {
		t.Fatal("expected ErrUnauthorized when non-member uploads file, got nil")
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got: %v", err)
	}
}

func TestFilesService_GetPresignedURL_Authorization(t *testing.T) {
	repo := newMockFilesRepo()
	svc := &service{repo: repo}

	roomID := uuid.New()
	uploaderID := uuid.New()
	otherMemberID := uuid.New()
	nonMemberID := uuid.New()

	repo.addMember(roomID, uploaderID)
	repo.addMember(roomID, otherMemberID)

	fileID := uuid.New()
	file := &models.File{
		ID:         fileID,
		RoomID:     roomID,
		UploaderID: uploaderID,
		FileName:   "document.pdf",
		S3Key:      "files/" + fileID.String() + ".pdf",
	}
	repo.files[fileID] = file

	// Negative test: non-member cannot get download URL for room file
	_, err := svc.GetPresignedURL(context.Background(), fileID, nonMemberID)
	if err == nil {
		t.Fatal("expected ErrUnauthorized when non-member attempts to get file URL")
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got: %v", err)
	}

	// Negative test: non-existent file
	_, err = svc.GetPresignedURL(context.Background(), uuid.New(), memberIDOr(otherMemberID))
	if err == nil || !errors.Is(err, ErrFileNotFound) {
		t.Fatalf("expected ErrFileNotFound, got: %v", err)
	}
}

func TestFilesService_DeleteFile_Authorization(t *testing.T) {
	repo := newMockFilesRepo()
	svc := &service{repo: repo}

	roomID := uuid.New()
	uploaderID := uuid.New()
	otherMemberID := uuid.New()
	adminID := uuid.New()

	fileID := uuid.New()
	file := &models.File{
		ID:         fileID,
		RoomID:     roomID,
		UploaderID: uploaderID,
		FileName:   "photo.jpg",
		S3Key:      "files/" + fileID.String() + ".jpg",
	}
	repo.files[fileID] = file

	// Non-uploader, non-admin member cannot delete
	err := svc.DeleteFile(context.Background(), fileID, otherMemberID, models.RoleMember)
	if err == nil || !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for non-uploader non-admin delete, got: %v", err)
	}

	// Note: deletion by uploader/admin requires s3Client.DeleteObject, which is tested in handler tests
	_ = adminID
}

func memberIDOr(id uuid.UUID) uuid.UUID {
	return id
}
