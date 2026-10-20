package router

import (
	"net/http"

	"github.com/Sidd231/4yaar/backend/internal/config"
	appmiddleware "github.com/Sidd231/4yaar/backend/internal/middleware"
	"github.com/Sidd231/4yaar/backend/internal/modules/auth"
	"github.com/Sidd231/4yaar/backend/internal/modules/chat"
	"github.com/Sidd231/4yaar/backend/internal/modules/files"
	"github.com/Sidd231/4yaar/backend/internal/modules/invite"
	"github.com/Sidd231/4yaar/backend/internal/modules/videocall"
	"github.com/Sidd231/4yaar/backend/internal/utils"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// Handlers encapsulates all HTTP module handlers.
type Handlers struct {
	Auth      *auth.Handler
	Invite    *invite.Handler
	Chat      *chat.Handler
	Files     *files.Handler
	VideoCall *videocall.Handler
}

// NewRouter initializes Chi router with global middlewares and module endpoints.
func NewRouter(cfg *config.Config, handlers *Handlers) *chi.Mux {
	r := chi.NewRouter()

	// Global Middlewares
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(appmiddleware.Logger())
	r.Use(chimiddleware.Recoverer)
	r.Use(appmiddleware.CORS())

	// Health Check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		utils.JSONResponse(w, http.StatusOK, map[string]string{
			"status":  "healthy",
			"app":     "4yaar",
			"version": "1.0.0",
		})
	})

	// API v1 Endpoints
	r.Route("/api/v1", func(api chi.Router) {
		auth.RegisterRoutes(api, handlers.Auth)
		invite.RegisterRoutes(api, handlers.Invite, cfg.JWTSecret)
		chat.RegisterHTTPRoutes(api, handlers.Chat, cfg.JWTSecret)
		files.RegisterRoutes(api, handlers.Files, cfg.JWTSecret)
		videocall.RegisterHTTPRoutes(api, handlers.VideoCall, cfg.JWTSecret)
	})

	// WebSocket Endpoints
	chat.RegisterWSRoutes(r, handlers.Chat)
	videocall.RegisterWSRoutes(r, handlers.VideoCall)

	return r
}
