package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Sidd231/4yaar/backend/internal/middleware"
	"github.com/Sidd231/4yaar/backend/internal/utils"
	"github.com/google/uuid"
)

// Handler exposes HTTP handlers for authentication.
type Handler struct {
	service Service
}

// NewHandler initializes a new Auth HTTP handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

type registerRequest struct {
	InviteToken string `json:"invite_token"`
	Name        string `json:"name"`
	Password    string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Register handles user registration using a valid invite token.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.InviteToken == "" || req.Name == "" || len(req.Password) < 8 {
		utils.JSONError(w, http.StatusBadRequest, "invite_token, name, and a password (min 8 chars) are required")
		return
	}

	tokenUUID, err := uuid.Parse(req.InviteToken)
	if err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid invite token format")
		return
	}

	resp, err := h.service.Register(r.Context(), tokenUUID, req.Name, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInvite):
			utils.JSONError(w, http.StatusBadRequest, "Invalid or expired invitation token")
		case errors.Is(err, ErrInviteAlreadyUsed):
			utils.JSONError(w, http.StatusConflict, "Invitation token has already been redeemed")
		case errors.Is(err, ErrUserAlreadyExists):
			utils.JSONError(w, http.StatusConflict, "A user with this email address already exists")
		default:
			utils.JSONError(w, http.StatusInternalServerError, "Failed to register user")
		}
		return
	}

	// Set httpOnly refresh token cookie
	setRefreshTokenCookie(w, resp.RefreshToken, 7*24*3600)
	utils.JSONResponse(w, http.StatusCreated, resp)
}

// Login authenticates a user and returns JWT token pair.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if req.Email == "" || req.Password == "" {
		utils.JSONError(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	resp, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			utils.JSONError(w, http.StatusUnauthorized, "Invalid email or password")
		} else {
			utils.JSONError(w, http.StatusInternalServerError, "Authentication failed")
		}
		return
	}

	// Log the auth response (excluding token values for security)
	slog.Info("Login successful", "user_id", resp.User.ID, "access_token_len", len(resp.AccessToken), "refresh_token_len", len(resp.RefreshToken))

	// Set httpOnly refresh token cookie
	setRefreshTokenCookie(w, resp.RefreshToken, 7*24*3600)
	utils.JSONResponse(w, http.StatusOK, resp)
}

// Refresh rotates the refresh token and returns a new access token.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	refreshToken := ""
	if cookie, err := r.Cookie("refresh_token"); err == nil && cookie.Value != "" {
		refreshToken = cookie.Value
	}

	if refreshToken == "" && r.Body != nil {
		var req tokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			refreshToken = req.RefreshToken
		}
	}

	if refreshToken == "" {
		utils.JSONError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	resp, err := h.service.Refresh(r.Context(), refreshToken)
	if err != nil {
		if errors.Is(err, ErrTokenRevoked) || errors.Is(err, ErrInvalidToken) {
			clearRefreshTokenCookie(w)
			utils.JSONError(w, http.StatusUnauthorized, "Invalid or revoked refresh token")
			return
		}
		utils.JSONError(w, http.StatusInternalServerError, "Failed to refresh token")
		return
	}

	// Set rotated httpOnly refresh token cookie
	setRefreshTokenCookie(w, resp.RefreshToken, 7*24*3600)
	utils.JSONResponse(w, http.StatusOK, resp)
}

// Logout invalidates a refresh token in Redis.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	refreshToken := ""
	if cookie, err := r.Cookie("refresh_token"); err == nil && cookie.Value != "" {
		refreshToken = cookie.Value
	}

	if refreshToken == "" && r.Body != nil {
		var req tokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			refreshToken = req.RefreshToken
		}
	}

	if refreshToken != "" {
		if err := h.service.Logout(r.Context(), refreshToken); err != nil && !errors.Is(err, ErrInvalidToken) {
			clearRefreshTokenCookie(w)
			utils.JSONError(w, http.StatusInternalServerError, "Failed to logout")
			return
		}
	}

	clearRefreshTokenCookie(w)
	utils.JSONResponse(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

func setRefreshTokenCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   false, // Set to false to support local dev http, or true in production
		SameSite: http.SameSiteLaxMode,
	})
}

func clearRefreshTokenCookie(w http.ResponseWriter, ) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	})
}

// GetMe returns the authenticated user's profile.
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		utils.JSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := h.service.GetMe(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			utils.JSONError(w, http.StatusNotFound, "User not found")
			return
		}
		slog.Error("Failed to fetch current user profile", "error", err, "user_id", claims.UserID)
		utils.JSONError(w, http.StatusInternalServerError, "Failed to retrieve user profile")
		return
	}

	utils.JSONResponse(w, http.StatusOK, user)
}
