package users

import (
	"github.com/Sidd231/4yaar/backend/internal/middleware"
	"github.com/Sidd231/4yaar/backend/internal/models"
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registers the user management endpoints.
func RegisterRoutes(r chi.Router, h *Handler, jwtSecret string) {
	r.Route("/users", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(jwtSecret))
		r.Use(middleware.RequireRole(models.RoleAdmin))
		r.Get("/", h.ListUsers)
	})
}
