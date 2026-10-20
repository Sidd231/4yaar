package invite

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"

	"github.com/Sidd231/4yaar/backend/internal/middleware"
	"github.com/Sidd231/4yaar/backend/internal/utils"
	"github.com/google/uuid"
)

// Handler manages invite endpoints.
type Handler struct {
	service Service
}

// NewHandler creates a new invite handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

type createInviteRequest struct {
	Email string `json:"email"`
}

// Create generates and emails an invitation. Accessible only to admins.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		utils.JSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req createInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if _, err := mail.ParseAddress(req.Email); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "A valid email address is required")
		return
	}

	inviterName := claims.Email // Fallback identifier
	invite, err := h.service.CreateInvite(r.Context(), claims.UserID, inviterName, req.Email)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to create invitation")
		return
	}

	utils.JSONResponse(w, http.StatusCreated, invite)
}

// Verify checks the validity of an invitation token provided in query params.
func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		utils.JSONError(w, http.StatusBadRequest, "Query parameter 'token' is required")
		return
	}

	tokenUUID, err := uuid.Parse(tokenStr)
	if err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid token format")
		return
	}

	res, err := h.service.VerifyInvite(r.Context(), tokenUUID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInviteNotFound):
			utils.JSONError(w, http.StatusNotFound, "Invitation token not found")
		case errors.Is(err, ErrInviteExpired):
			utils.JSONError(w, http.StatusGone, "Invitation token has expired")
		case errors.Is(err, ErrInviteUsed):
			utils.JSONError(w, http.StatusConflict, "Invitation token has already been redeemed")
		default:
			utils.JSONError(w, http.StatusInternalServerError, "Failed to verify invitation")
		}
		return
	}

	utils.JSONResponse(w, http.StatusOK, res)
}
