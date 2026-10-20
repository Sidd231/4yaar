package videocall

import (
	"github.com/Sidd231/4yaar/backend/internal/middleware"
	"github.com/go-chi/chi/v5"
)

// RegisterHTTPRoutes registers the video call HTTP endpoints under /video.
func RegisterHTTPRoutes(r chi.Router, h *Handler, jwtSecret string) {
	r.Route("/video", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(jwtSecret))
		r.Post("/rooms", h.CreateRoom)
		r.Get("/rooms/{roomID}", h.GetRoomInfo)
	})
}

// RegisterWSRoutes registers the video call WebSocket signaling endpoint under /ws/video.
func RegisterWSRoutes(r chi.Router, h *Handler) {
	r.Get("/ws/video/{roomID}", h.ServeWS)
}
