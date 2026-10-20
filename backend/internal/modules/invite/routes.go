package invite

import (
	"github.com/Sidd231/4yaar/backend/internal/middleware"
	"github.com/Sidd231/4yaar/backend/internal/models"
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registers the invite endpoints.
func RegisterRoutes(r chi.Router, h *Handler, jwtSecret string) {
	r.Route("/invites", func(r chi.Router) {
		// Public verification endpoint
		r.Get("/verify", h.Verify)

		// Protected admin invitation endpoint
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(jwtSecret))
			r.Use(middleware.RequireRole(models.RoleAdmin))
			r.Post("/", h.Create)
		})
	})
}
