package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Hub maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	// Registered clients partitioned by RoomID
	rooms      map[uuid.UUID]map[*Client]bool
	roomsMu    sync.RWMutex
	Register   chan *Client
	Unregister chan *Client

	redisClient *redis.Client
	service     Service
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewHub initializes a new Chat Hub.
func NewHub(redisClient *redis.Client, service Service) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		rooms:       make(map[uuid.UUID]map[*Client]bool),
		Register:    make(chan *Client),
		Unregister:  make(chan *Client),
		redisClient: redisClient,
		service:     service,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Run begins the event loop for client registration, unregistration, and Redis pub/sub subscription.
func (h *Hub) Run() {
	// Start background subscriber listening on "chat:*" Redis pub/sub channels
	go h.listenToRedisPubSub()

	for {
		select {
		case <-h.ctx.Done():
			return

		case client := <-h.Register:
			h.roomsMu.Lock()
			if _, exists := h.rooms[client.RoomID]; !exists {
				h.rooms[client.RoomID] = make(map[*Client]bool)
			}
			h.rooms[client.RoomID][client] = true
			h.roomsMu.Unlock()
			slog.Debug("Client registered to chat room", "user_id", client.UserID, "room_id", client.RoomID)

		case client := <-h.Unregister:
			h.roomsMu.Lock()
			if clients, exists := h.rooms[client.RoomID]; exists {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.Send)
					if len(clients) == 0 {
						delete(h.rooms, client.RoomID)
					}
				}
			}
			h.roomsMu.Unlock()
			slog.Debug("Client unregistered from chat room", "user_id", client.UserID, "room_id", client.RoomID)
		}
	}
}

// Shutdown stops the hub and its background goroutines.
func (h *Hub) Shutdown() {
	h.cancel()
}

// HandleClientMessage persists incoming messages to PostgreSQL and publishes to Redis.
func (h *Hub) HandleClientMessage(client *Client, incoming *WSIncomingMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Persist message first
	msg, err := h.service.SaveMessage(ctx, client.RoomID, client.UserID, incoming.Content, incoming.Type, incoming.FileID)
	if err != nil {
		slog.Error("Failed to persist chat message", "room_id", client.RoomID, "user_id", client.UserID, "error", err)
		return
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		slog.Error("Failed to serialize chat message for Redis", "error", err)
		return
	}

	// Publish to Redis channel "chat:<roomID>"
	channel := fmt.Sprintf("chat:%s", client.RoomID.String())
	if err := h.redisClient.Publish(ctx, channel, payload).Err(); err != nil {
		slog.Error("Failed to publish message to Redis pub/sub", "channel", channel, "error", err)
	}
}

// listenToRedisPubSub subscribes to "chat:*" channels to fan out across multiple backend instances.
func (h *Hub) listenToRedisPubSub() {
	pubsub := h.redisClient.PSubscribe(h.ctx, "chat:*")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-h.ctx.Done():
			return

		case msg, ok := <-ch:
			if !ok {
				return
			}

			// msg.Channel has format "chat:<roomID>"
			parts := strings.Split(msg.Channel, ":")
			if len(parts) != 2 {
				continue
			}

			roomID, err := uuid.Parse(parts[1])
			if err != nil {
				continue
			}

			// Broadcast to local room clients
			h.broadcastToRoom(roomID, []byte(msg.Payload))
		}
	}
}

// broadcastToRoom pushes serialized bytes to all clients connected locally to roomID.
func (h *Hub) broadcastToRoom(roomID uuid.UUID, data []byte) {
	h.roomsMu.RLock()
	defer h.roomsMu.RUnlock()

	clients, exists := h.rooms[roomID]
	if !exists {
		return
	}

	for client := range clients {
		select {
		case client.Send <- data:
		default:
			// If buffer is full, remove client to avoid blocking other connections
			go func(c *Client) {
				h.Unregister <- c
			}(client)
		}
	}
}
