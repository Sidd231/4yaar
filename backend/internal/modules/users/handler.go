package users

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Sidd231/4yaar/backend/internal/models"
	"github.com/Sidd231/4yaar/backend/internal/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserResponse represents a user item in the response list.
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// Handler handles user management HTTP endpoints.
type Handler struct {
	db *gorm.DB
}

// NewHandler creates a new users Handler with the given database connection.
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// ListUsers handles GET /api/v1/users with optional ?search= filter.
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	query := h.db.WithContext(r.Context()).Model(&models.User{})

	if search != "" {
		searchTerm := "%" + search + "%"
		query = query.Where("name ILIKE ? OR email ILIKE ?", searchTerm, searchTerm)
	}

	var dbUsers []models.User
	if err := query.Order("created_at ASC").Find(&dbUsers).Error; err != nil {
		slog.Error("Failed to list users", "error", err, "search", search)
		utils.JSONError(w, http.StatusInternalServerError, "Failed to retrieve users")
		return
	}

	users := make([]UserResponse, len(dbUsers))
	for i, u := range dbUsers {
		users[i] = UserResponse{
			ID:        u.ID,
			Name:      u.Name,
			Email:     u.Email,
			Role:      u.Role,
			CreatedAt: u.CreatedAt,
		}
	}

	utils.JSONResponse(w, http.StatusOK, users)
}
