package files

import (
	"github.com/Sidd231/4yaar/backend/internal/middleware"
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registers file management endpoints on the router.
func RegisterRoutes(r chi.Router, h *Handler, jwtSecret string) {
	r.Route("/files", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(jwtSecret))
		r.Post("/upload", h.Upload)
		r.Get("/{fileID}", h.GetURL)
		r.Delete("/{fileID}", h.Delete)
	})
}
