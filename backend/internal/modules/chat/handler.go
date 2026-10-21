package chat

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/Sidd231/4yaar/backend/internal/middleware"
	"github.com/Sidd231/4yaar/backend/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Handler manages chat HTTP endpoints and WebSocket connections.
type Handler struct {
	service   Service
	hub       *Hub
	jwtSecret string
	upgrader  websocket.Upgrader
}

// NewHandler creates a new chat handler with configurable origin allowlist.
func NewHandler(service Service, hub *Hub, jwtSecret string, allowedOrigins []string) *Handler {
	return &Handler{
		service:   service,
		hub:       hub,
		jwtSecret: jwtSecret,
		upgrader:  utils.NewWebSocketUpgrader(allowedOrigins),
	}
}

type createRoomRequest struct {
	Name string `json:"name"`
	Type string `json:"type"` // direct or group
}

// CreateRoom handles room creation.
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

	if req.Name == "" {
		utils.JSONError(w, http.StatusBadRequest, "Room name is required")
		return
	}

	room, err := h.service.CreateRoom(r.Context(), req.Name, req.Type, claims.UserID)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to create room")
		return
	}

	utils.JSONResponse(w, http.StatusCreated, room)
}

// ListRooms lists all chat rooms the authenticated user is a member of.
func (h *Handler) ListRooms(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		utils.JSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	rooms, err := h.service.GetUserRooms(r.Context(), claims.UserID)
	if err != nil {
		utils.JSONError(w, http.StatusInternalServerError, "Failed to fetch rooms")
		return
	}

	utils.JSONResponse(w, http.StatusOK, rooms)
}

type addMemberRequest struct {
	UserID string `json:"user_id"`
}

// AddMember adds a user to a chat room.
func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
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

	var req addMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	targetUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		utils.JSONError(w, http.StatusBadRequest, "Invalid user_id UUID")
		return
	}

	member, err := h.service.AddMember(r.Context(), roomID, targetUserID, claims.UserID, claims.Role)
	if err != nil {
		switch {
		case errors.Is(err, ErrRoomNotFound):
			utils.JSONError(w, http.StatusNotFound, "Room not found")
		case errors.Is(err, ErrNotAuthorized):
			utils.JSONError(w, http.StatusForbidden, "Only the room creator or an admin can add members")
		case errors.Is(err, ErrAlreadyMember):
			utils.JSONError(w, http.StatusConflict, "User is already a member of this room")
		default:
			slog.Error("Failed to add member to room", "error", err, "room_id", roomID, "user_id", targetUserID)
			utils.JSONError(w, http.StatusInternalServerError, "Failed to add member to room")
		}
		return
	}

	utils.JSONResponse(w, http.StatusCreated, map[string]any{
		"room_id":   member.RoomID,
		"user_id":   member.UserID,
		"joined_at": member.JoinedAt,
	})
}

// GetMessages retrieves cursor-based paginated message history.
func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
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

	var cursorTime *time.Time
	cursorStr := r.URL.Query().Get("cursor")
	if cursorStr != "" {
		parsed, err := time.Parse(time.RFC3339Nano, cursorStr)
		if err == nil {
			cursorTime = &parsed
		} else {
			// Try RFC3339 without subseconds
			parsed, err := time.Parse(time.RFC3339, cursorStr)
			if err == nil {
				cursorTime = &parsed
			}
		}
	}

	limit := 50
	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	messages, err := h.service.GetRoomMessages(r.Context(), roomID, claims.UserID, cursorTime, limit)
	if err != nil {
		if errors.Is(err, ErrAccessDenied) {
			utils.JSONError(w, http.StatusForbidden, "You do not have access to this room")
			return
		}
		utils.JSONError(w, http.StatusInternalServerError, "Failed to retrieve messages")
		return
	}

	utils.JSONResponse(w, http.StatusOK, messages)
}

// ServeWS handles WebSocket connection upgrades for the chat room.
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

	// Verify that only access tokens can authenticate WebSocket connections
	if claims.TokenType != middleware.TokenTypeAccess {
		http.Error(w, "Unauthorized: access token required", http.StatusUnauthorized)
		return
	}

	// Verify user is a member of the room
	allowed, err := h.service.CanAccessRoom(r.Context(), roomID, claims.UserID)
	if err != nil || !allowed {
		http.Error(w, "Forbidden: not a member of this room", http.StatusForbidden)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Failed to upgrade chat websocket", "error", err)
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
