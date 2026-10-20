package videocall

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 131072 // 128KB (accommodates large SDP blobs and ICE candidates)
)

// SignalMessage defines the standard signaling message envelope exchanged over WebSockets.
type SignalMessage struct {
	Type     string          `json:"type"`                // join, offer, answer, ice-candidate, leave, peer-joined, peer-left
	RoomID   uuid.UUID       `json:"room_id,omitempty"`   // Video call room UUID
	SenderID uuid.UUID       `json:"sender_id,omitempty"` // User UUID sending this signal
	TargetID *uuid.UUID      `json:"target_id,omitempty"` // Specific peer target, if direct
	Data     json.RawMessage `json:"data,omitempty"`      // SDP object, ICE candidate, or extra payload
}

// Client represents an active WebRTC signaling peer connection.
type Client struct {
	Hub      *Hub
	Conn     *websocket.Conn
	Send     chan []byte
	RoomID   uuid.UUID
	UserID   uuid.UUID
	UserName string
}

// ReadPump handles incoming WebRTC signaling messages from the WebSocket peer.
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, rawMsg, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Warn("Video signaling WebSocket read error", "user_id", c.UserID, "room_id", c.RoomID, "error", err)
			}
			break
		}

		var signal SignalMessage
		if err := json.Unmarshal(rawMsg, &signal); err != nil {
			slog.Warn("Invalid signaling payload format", "error", err)
			continue
		}

		// Ensure room_id and sender_id are stamped accurately
		signal.RoomID = c.RoomID
		signal.SenderID = c.UserID

		c.Hub.RouteSignal(c, &signal)
	}
}

// WritePump pushes signaling packets down to the connected peer client.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(message); err != nil {
				return
			}
			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
