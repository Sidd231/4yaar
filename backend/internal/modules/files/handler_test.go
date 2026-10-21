package files

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sidd231/4yaar/backend/internal/middleware"
	"github.com/Sidd231/4yaar/backend/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type mockFilesService struct {
	uploadFunc     func(ctx context.Context, uploaderID, roomID uuid.UUID, fileName, mimeType string, fileSize int64, reader io.Reader) (*UploadResponse, error)
	getURLFunc     func(ctx context.Context, fileID, requesterID uuid.UUID) (string, error)
	deleteFileFunc func(ctx context.Context, fileID, requesterID uuid.UUID, requesterRole string) error
}

func (m *mockFilesService) UploadFile(ctx context.Context, uploaderID, roomID uuid.UUID, fileName, mimeType string, fileSize int64, reader io.Reader) (*UploadResponse, error) {
	if m.uploadFunc != nil {
		return m.uploadFunc(ctx, uploaderID, roomID, fileName, mimeType, fileSize, reader)
	}
	return nil, errors.New("not implemented")
}

func (m *mockFilesService) GetPresignedURL(ctx context.Context, fileID, requesterID uuid.UUID) (string, error) {
	if m.getURLFunc != nil {
		return m.getURLFunc(ctx, fileID, requesterID)
	}
	return "", errors.New("not implemented")
}

func (m *mockFilesService) DeleteFile(ctx context.Context, fileID, requesterID uuid.UUID, requesterRole string) error {
	if m.deleteFileFunc != nil {
		return m.deleteFileFunc(ctx, fileID, requesterID, requesterRole)
	}
	return errors.New("not implemented")
}

func withAuthClaims(r *http.Request, userID uuid.UUID, role string) *http.Request {
	claims := &middleware.UserClaims{
		UserID:    userID,
		Email:     "user@example.com",
		Role:      role,
		TokenType: middleware.TokenTypeAccess,
	}
	ctx := context.WithValue(r.Context(), middleware.UserContextKey, claims)
	return r.WithContext(ctx)
}

func TestFilesHandler_Upload_RoomMembershipAuthorization(t *testing.T) {
	memberID := uuid.New()
	nonMemberID := uuid.New()
	roomID := uuid.New()

	svc := &mockFilesService{
		uploadFunc: func(ctx context.Context, uploaderID, rID uuid.UUID, fileName, mimeType string, fileSize int64, reader io.Reader) (*UploadResponse, error) {
			if uploaderID != memberID {
				return nil, ErrUnauthorized
			}
			return &UploadResponse{
				File: &models.File{
					ID:       uuid.New(),
					RoomID:   rID,
					FileName: fileName,
				},
				PresignedURL: "https://s3.example.com/test",
			}, nil
		},
	}

	handler := NewHandler(svc)

	createUploadRequest := func(userID uuid.UUID) *http.Request {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.WriteField("room_id", roomID.String())
		part, _ := writer.CreateFormFile("file", "test.png")
		_, _ = part.Write([]byte("fake png image data"))
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/files/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		return withAuthClaims(req, userID, models.RoleMember)
	}

	t.Run("member upload succeeds with 201", func(t *testing.T) {
		req := createUploadRequest(memberID)
		rec := httptest.NewRecorder()
		handler.Upload(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("non-member upload rejected with 403 Forbidden", func(t *testing.T) {
		req := createUploadRequest(nonMemberID)
		rec := httptest.NewRecorder()
		handler.Upload(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected status 403 Forbidden, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestFilesHandler_GetURL_RoomMembershipAuthorization(t *testing.T) {
	memberID := uuid.New()
	nonMemberID := uuid.New()
	fileID := uuid.New()

	svc := &mockFilesService{
		getURLFunc: func(ctx context.Context, fID, reqID uuid.UUID) (string, error) {
			if fID != fileID {
				return "", ErrFileNotFound
			}
			if reqID != memberID {
				return "", ErrUnauthorized
			}
			return "https://s3.example.com/signed-download-url", nil
		},
	}

	handler := NewHandler(svc)

	r := chi.NewRouter()
	r.Get("/files/{fileID}/url", handler.GetURL)

	t.Run("member gets 200 with presigned URL", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/files/"+fileID.String()+"/url", nil)
		req = withAuthClaims(req, memberID, models.RoleMember)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Data struct {
				FileID       string `json:"file_id"`
				PresignedURL string `json:"presigned_url"`
			} `json:"data"`
		}
		_ = json.NewDecoder(rec.Body).Decode(&resp)
		if resp.Data.PresignedURL != "https://s3.example.com/signed-download-url" {
			t.Fatalf("unexpected presigned_url in response: %+v", resp.Data)
		}
	})

	t.Run("non-member is rejected with 403 Forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/files/"+fileID.String()+"/url", nil)
		req = withAuthClaims(req, nonMemberID, models.RoleMember)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403 Forbidden, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("non-existent file returns 404 Not Found", func(t *testing.T) {
		unknownID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/files/"+unknownID.String()+"/url", nil)
		req = withAuthClaims(req, memberID, models.RoleMember)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}
