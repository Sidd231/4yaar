package auth

import (
	"github.com/Sidd231/4yaar/backend/internal/middleware"
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registers the authentication endpoints on the provided Chi router.
func RegisterRoutes(r chi.Router, h *Handler, jwtSecret string) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/refresh", h.Refresh)
		r.Post("/logout", h.Logout)

		// Protected endpoints
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(jwtSecret))
			r.Get("/me", h.GetMe)
		})
	})
}
