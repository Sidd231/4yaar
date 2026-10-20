package files

import (
	"errors"
	"net/http"

	"github.com/Sidd231/4yaar/backend/internal/middleware"
	"github.com/Sidd231/4yaar/backend/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler processes file uploads, presigned URL requests, and deletions.
type Handler struct {
	service Service
}

// NewHandler creates a new files handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Upload handles multipart file upload to S3 and database registration.
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		utils.JSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Limit request body to MaxFileSize + 1MB overhead
	r.Body = http.MaxBytesReader(w, r.Body, MaxFileSize+1024*1024)
	if err := r.ParseMultipartForm(MaxFileSize); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "File too large or malformed multipart form")
		return
	}

	roomIDStr := r.FormValue("room_id")
	if roomIDStr == "" {
		utils.JSONError(w, http.StatusBadRequest, "room_id form field is required")
		return
	}

	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid room_id UUID")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Missing 'file' field in multipart form")
		return
	}
	defer file.Close()

	// Detect or parse mime type
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	resp, err := h.service.UploadFile(
		r.Context(),
		claims.UserID,
		roomID,
		header.Filename,
		mimeType,
		header.Size,
		file,
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrFileTooLarge):
			utils.JSONError(w, http.StatusBadRequest, "File exceeds maximum size of 50MB")
		case errors.Is(err, ErrUnsupportedType):
			utils.JSONError(w, http.StatusBadRequest, err.Error())
		default:
			utils.JSONError(w, http.StatusInternalServerError, "Failed to upload file")
		}
		return
	}

	utils.JSONResponse(w, http.StatusCreated, resp)
}

// GetURL generates a fresh 15-minute presigned GET URL for a file.
func (h *Handler) GetURL(w http.ResponseWriter, r *http.Request) {
	fileIDStr := chi.URLParam(r, "fileID")
	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid fileID UUID")
		return
	}

	url, err := h.service.GetPresignedURL(r.Context(), fileID)
	if err != nil {
		if errors.Is(err, ErrFileNotFound) {
			utils.JSONError(w, http.StatusNotFound, "File not found")
			return
		}
		utils.JSONError(w, http.StatusInternalServerError, "Failed to generate presigned download URL")
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]string{
		"file_id":       fileID.String(),
		"presigned_url": url,
	})
}

// Delete marks a file as deleted in the database and removes it from S3.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		utils.JSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	fileIDStr := chi.URLParam(r, "fileID")
	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid fileID UUID")
		return
	}

	err = h.service.DeleteFile(r.Context(), fileID, claims.UserID, claims.Role)
	if err != nil {
		switch {
		case errors.Is(err, ErrFileNotFound):
			utils.JSONError(w, http.StatusNotFound, "File not found")
		case errors.Is(err, ErrUnauthorized):
			utils.JSONError(w, http.StatusForbidden, "You do not have permission to delete this file")
		default:
			utils.JSONError(w, http.StatusInternalServerError, "Failed to delete file")
		}
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]string{"message": "File deleted successfully"})
}
