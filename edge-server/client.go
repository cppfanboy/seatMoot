package main

import (
	"encoding/json"
	"log"
	"time"

	"concert-booking/shared"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024
)

// Client is a middleman between the websocket connection and the hub
type Client struct {
	hub *Hub

	// The websocket connection
	conn *websocket.Conn

	// Buffered channel of outbound messages
	send chan []byte

	// Client ID
	id string

	// User ID (set when client subscribes)
	userID string

	// Connection timestamp
	connectedAt time.Time

	// Last activity timestamp
	lastActivity time.Time
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
		log.Printf("Client %s disconnected", c.id)
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for client %s: %v", c.id, err)
			}
			break
		}

		// Update last activity
		c.lastActivity = time.Now()

		// Parse the message
		var clientMsg shared.ClientMessage
		if err := json.Unmarshal(message, &clientMsg); err != nil {
			log.Printf("Error parsing message from client %s: %v", c.id, err)
			c.sendError("Invalid message format")
			continue
		}

		// Handle the message
		c.handleMessage(&clientMsg)
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleMessage(msg *shared.ClientMessage) {
	log.Printf("Client %s sent message type: %s", c.id, msg.Type)

	switch msg.Type {
	case shared.MessageTypeSubscribe:
		c.handleSubscribe(msg.Data)
	case shared.MessageTypeSelectSeat:
		c.handleSelectSeat(msg.Data)
	case shared.MessageTypeBookSeat:
		c.handleBookSeat(msg.Data)
	case shared.MessageTypeReleaseSeat:
		c.handleReleaseSeat(msg.Data)
	default:
		c.sendError("Unknown message type: " + msg.Type)
	}
}

func (c *Client) sendMessage(msgType string, data interface{}) {
	msg := shared.ServerMessage{
		Type: msgType,
		Data: data,
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	select {
	case c.send <- jsonData:
		// Message queued successfully
	default:
		// Client send buffer is full
		log.Printf("Failed to send message to client %s: buffer full", c.id)
	}
}

func (c *Client) sendError(errorMsg string) {
	c.sendMessage(shared.MessageTypeError, shared.ErrorResponse{Error: errorMsg})
}

// close cleanly shuts down the client connection
func (c *Client) close() {
	// Send close message to client
	c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	
	// Close the connection
	c.conn.Close()
	
	// Log connection duration
	duration := time.Since(c.connectedAt)
	log.Printf("Client %s (user: %s) disconnected after %v", c.id, c.userID, duration)
}