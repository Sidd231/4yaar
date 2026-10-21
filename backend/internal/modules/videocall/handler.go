package videocall

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Sidd231/4yaar/backend/internal/middleware"
	"github.com/Sidd231/4yaar/backend/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Handler handles video call HTTP and WebSocket endpoints.
type Handler struct {
	service   Service
	hub       *Hub
	jwtSecret string
	upgrader  websocket.Upgrader
}

// NewHandler initializes a new video call handler with configurable origin allowlist.
func NewHandler(service Service, hub *Hub, jwtSecret string, allowedOrigins []string) *Handler {
	return &Handler{
		service:   service,
		hub:       hub,
		jwtSecret: jwtSecret,
		upgrader:  utils.NewWebSocketUpgrader(allowedOrigins),
	}
}

type createRoomRequest struct {
	Name   string  `json:"name"`
	RoomID *string `json:"room_id,omitempty"`
}

// CreateRoom handles creating a new video conference room or reusing an existing room if room_id is provided.
func (h *Handler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		utils.JSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req createRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// If room_id is provided, validate existence and membership, and return existing room info
	if req.RoomID != nil && strings.TrimSpace(*req.RoomID) != "" {
		existingID, err := uuid.Parse(strings.TrimSpace(*req.RoomID))
		if err != nil {
			utils.JSONError(w, http.StatusBadRequest, "Invalid room_id UUID")
			return
		}

		info, err := h.service.GetRoomInfo(r.Context(), existingID, claims.UserID)
		if err != nil {
			switch {
			case errors.Is(err, ErrVideoRoomNotFound):
				utils.JSONError(w, http.StatusNotFound, "Room not found")
			case errors.Is(err, ErrVideoRoomUnauthorized):
				utils.JSONError(w, http.StatusForbidden, "You are not a member of this room")
			default:
				utils.JSONError(w, http.StatusInternalServerError, "Failed to get video room details")
			}
			return
		}

		utils.JSONResponse(w, http.StatusOK, map[string]any{
			"room_id":    info.ID,
			"name":       info.Name,
			"created_at": info.CreatedAt,
		})
		return
	}

	if req.Name == "" {
		utils.JSONError(w, http.StatusBadRequest, "Room name is required")
		return
	}

	room, err := h.service.CreateRoom(r.Context(), req.Name, claims.UserID)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to create video room")
		return
	}

	utils.JSONResponse(w, http.StatusCreated, map[string]any{
		"room_id":    room.ID,
		"name":       room.Name,
		"created_at": room.CreatedAt,
	})
}

// GetRoomInfo returns the metadata and real-time active peer count for a video room.
// Enforces that the requesting user is an active member of the room.
func (h *Handler) GetRoomInfo(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		utils.JSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	roomIDStr := chi.URLParam(r, "roomID")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid roomID UUID")
		return
	}

	info, err := h.service.GetRoomInfo(r.Context(), roomID, claims.UserID)
	if err != nil {
		switch {
		case errors.Is(err, ErrVideoRoomNotFound):
			utils.JSONError(w, http.StatusNotFound, "Video room not found")
		case errors.Is(err, ErrVideoRoomUnauthorized):
			utils.JSONError(w, http.StatusForbidden, "You are not a member of this video room")
		default:
			utils.JSONError(w, http.StatusInternalServerError, "Failed to get video room details")
		}
		return
	}

	utils.JSONResponse(w, http.StatusOK, info)
}

// ServeWS upgrades the HTTP connection to a WebRTC signaling WebSocket.
// Enforces origin check, access token validation, and room membership BEFORE upgrading connection.
func (h *Handler) ServeWS(w http.ResponseWriter, r *http.Request) {
	roomIDStr := chi.URLParam(r, "roomID")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		http.Error(w, "Invalid roomID UUID", http.StatusBadRequest)
		return
	}

	tokenStr := utils.ExtractWebSocketToken(r)
	if tokenStr == "" {
		http.Error(w, "Unauthorized: missing token", http.StatusUnauthorized)
		return
	}

	claims, err := middleware.ParseAndValidateToken(tokenStr, h.jwtSecret)
	if err != nil {
		http.Error(w, "Unauthorized: invalid or expired token", http.StatusUnauthorized)
		return
	}

	// Protected WebSocket connections require access tokens
	if claims.TokenType != middleware.TokenTypeAccess {
		http.Error(w, "Unauthorized: access token required", http.StatusUnauthorized)
		return
	}

	// Verify room membership before upgrading the connection
	isMember, err := h.service.IsUserInRoom(r.Context(), roomID, claims.UserID)
	if err != nil || !isMember {
		http.Error(w, "Forbidden: you are not a member of this video room", http.StatusForbidden)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Failed to upgrade video signaling websocket", "error", err)
		return
	}

	client := &Client{
		Hub:      h.hub,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		RoomID:   roomID,
		UserID:   claims.UserID,
		UserName: claims.Email,
	}

	h.hub.Register <- client

	go client.WritePump()
	go client.ReadPump()
}
