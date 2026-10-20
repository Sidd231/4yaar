package videocall

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

// Hub coordinates WebRTC and mediasoup signaling between active peers in video rooms.
type Hub struct {
	// Active peers indexed by roomID -> userID -> Client
	rooms      map[uuid.UUID]map[uuid.UUID]*Client
	roomsMu    sync.RWMutex
	Register   chan *Client
	Unregister chan *Client
}

// NewHub initializes a new video call signaling hub.
func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[uuid.UUID]map[uuid.UUID]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

// Run starts the signaling hub's lifecycle management loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.roomsMu.Lock()
			if _, exists := h.rooms[client.RoomID]; !exists {
				h.rooms[client.RoomID] = make(map[uuid.UUID]*Client)
			}
			h.rooms[client.RoomID][client.UserID] = client
			h.roomsMu.Unlock()

			slog.Info("Video peer connected", "room_id", client.RoomID, "user_id", client.UserID)

			// Notify other peers in this room that a new peer joined
			h.broadcastExcept(client.RoomID, client.UserID, &SignalMessage{
				Type:     "peer-joined",
				RoomID:   client.RoomID,
				SenderID: client.UserID,
			})

			// Inform the joined client about existing peers in the room
			h.sendExistingPeers(client)

		case client := <-h.Unregister:
			h.roomsMu.Lock()
			if peers, exists := h.rooms[client.RoomID]; exists {
				if _, ok := peers[client.UserID]; ok {
					delete(peers, client.UserID)
					close(client.Send)
					if len(peers) == 0 {
						delete(h.rooms, client.RoomID)
					}
				}
			}
			h.roomsMu.Unlock()

			slog.Info("Video peer disconnected", "room_id", client.RoomID, "user_id", client.UserID)

			// Broadcast peer-left to remaining peers
			h.broadcastExcept(client.RoomID, client.UserID, &SignalMessage{
				Type:     "peer-left",
				RoomID:   client.RoomID,
				SenderID: client.UserID,
			})
		}
	}
}

// GetPeerCount returns the number of active peers currently connected to a video call room.
func (h *Hub) GetPeerCount(roomID uuid.UUID) int {
	h.roomsMu.RLock()
	defer h.roomsMu.RUnlock()

	peers, exists := h.rooms[roomID]
	if !exists {
		return 0
	}
	return len(peers)
}

// RouteSignal relays signaling messages (offer, answer, ice-candidate, leave) to the target peer or broadcasts to room.
func (h *Hub) RouteSignal(sender *Client, signal *SignalMessage) {
	if signal.Type == "leave" {
		go func() {
			h.Unregister <- sender
		}()
		return
	}

	payload, err := json.Marshal(signal)
	if err != nil {
		slog.Error("Failed to marshal signaling message", "error", err)
		return
	}

	h.roomsMu.RLock()
	defer h.roomsMu.RUnlock()

	peers, exists := h.rooms[sender.RoomID]
	if !exists {
		return
	}

	// Targeted peer message (SDP offer / answer / targeted ICE candidate)
	if signal.TargetID != nil && *signal.TargetID != uuid.Nil {
		if targetClient, ok := peers[*signal.TargetID]; ok {
			select {
			case targetClient.Send <- payload:
			default:
				slog.Warn("Target peer send buffer full, dropping signal packet", "target_id", signal.TargetID)
			}
		}
		return
	}

	// Broadcast signal to all other peers in the room
	for peerID, peerClient := range peers {
		if peerID != sender.UserID {
			select {
			case peerClient.Send <- payload:
			default:
				slog.Warn("Peer send buffer full, dropping signal packet", "peer_id", peerID)
			}
		}
	}
}

// broadcastExcept sends a signal to all peers in room except the excluded user.
func (h *Hub) broadcastExcept(roomID uuid.UUID, excludeUserID uuid.UUID, signal *SignalMessage) {
	payload, err := json.Marshal(signal)
	if err != nil {
		return
	}

	h.roomsMu.RLock()
	defer h.roomsMu.RUnlock()

	peers, exists := h.rooms[roomID]
	if !exists {
		return
	}

	for peerID, peerClient := range peers {
		if peerID != excludeUserID {
			select {
			case peerClient.Send <- payload:
			default:
			}
		}
	}
}

// sendExistingPeers provides the joined peer with a list of currently active peer IDs.
func (h *Hub) sendExistingPeers(joiner *Client) {
	h.roomsMu.RLock()
	defer h.roomsMu.RUnlock()

	peers, exists := h.rooms[joiner.RoomID]
	if !exists {
		return
	}

	var peerIDs []uuid.UUID
	for peerID := range peers {
		if peerID != joiner.UserID {
			peerIDs = append(peerIDs, peerID)
		}
	}

	rawPeerList, err := json.Marshal(peerIDs)
	if err != nil {
		return
	}

	msg := &SignalMessage{
		Type:     "existing-peers",
		RoomID:   joiner.RoomID,
		SenderID: uuid.Nil,
		Data:     rawPeerList,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return
	}

	select {
	case joiner.Send <- payload:
	default:
	}
}
