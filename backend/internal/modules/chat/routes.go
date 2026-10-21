package chat

import (
	"github.com/Sidd231/4yaar/backend/internal/middleware"
	"github.com/go-chi/chi/v5"
)

// RegisterHTTPRoutes mounts chat REST endpoints under /chat.
func RegisterHTTPRoutes(r chi.Router, h *Handler, jwtSecret string) {
	r.Route("/chat", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(jwtSecret))
		r.Get("/rooms", h.ListRooms)
		r.Post("/rooms", h.CreateRoom)
		r.Post("/rooms/{roomID}/members", h.AddMember)
		r.Get("/rooms/{roomID}/messages", h.GetMessages)
	})
}

// RegisterWSRoutes mounts the chat WebSocket endpoint under /ws/chat.
func RegisterWSRoutes(r chi.Router, h *Handler) {
	r.Get("/ws/chat/{roomID}", h.ServeWS)
}
