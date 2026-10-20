package chat

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
	maxMessageSize = 65536 // 64KB
)

// Client represents an active WebSocket connection attached to a room.
type Client struct {
	Hub      *Hub
	Conn     *websocket.Conn
	Send     chan []byte
	RoomID   uuid.UUID
	UserID   uuid.UUID
	UserName string
}

// WSIncomingMessage defines the structure of payload sent from clients over WebSocket.
type WSIncomingMessage struct {
	Content string     `json:"content"`
	Type    string     `json:"type"`    // text, image, file
	FileID  *uuid.UUID `json:"file_id"` // nullable
}

// ReadPump listens for incoming messages from the WebSocket connection and forwards them to the Hub.
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
				slog.Warn("WebSocket read error", "user_id", c.UserID, "room_id", c.RoomID, "error", err)
			}
			break
		}

		var incoming WSIncomingMessage
		if err := json.Unmarshal(rawMsg, &incoming); err != nil {
			slog.Warn("Failed to unmarshal chat message", "error", err)
			continue
		}

		// Dispatch message to Hub for database persistence and Redis pub/sub
		c.Hub.HandleClientMessage(c, &incoming)
	}
}

// WritePump pumps messages from the send channel to the WebSocket connection.
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
				// Hub closed the channel
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
