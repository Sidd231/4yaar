package auth

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registers the authentication endpoints on the provided Chi router.
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/refresh", h.Refresh)
		r.Post("/logout", h.Logout)
	})
}
