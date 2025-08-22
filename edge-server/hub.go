package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

// HubStats tracks statistics for the hub
type HubStats struct {
	TotalClients      int       `json:"total_clients"`
	TotalMessages     int64     `json:"total_messages"`
	ConnectedAt       time.Time `json:"connected_at"`
	LastBroadcastTime time.Time `json:"last_broadcast_time"`
}

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from the clients
	broadcast chan []byte

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Statistics
	stats HubStats
	
	// Mutex for thread-safe operations
	mu sync.RWMutex
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		stats: HubStats{
			ConnectedAt: time.Now(),
		},
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.stats.TotalClients = len(h.clients)
			h.mu.Unlock()
			
			log.Printf("Client registered: %s (total clients: %d)", client.id, h.stats.TotalClients)
			
			// Send welcome message to the new client
			h.sendWelcomeMessage(client)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.stats.TotalClients = len(h.clients)
			}
			h.mu.Unlock()
			
			log.Printf("Client unregistered: %s (total clients: %d)", client.id, h.stats.TotalClients)

		case message := <-h.broadcast:
			h.mu.RLock()
			clientCount := len(h.clients)
			h.mu.RUnlock()
			
			// Update statistics
			h.mu.Lock()
			h.stats.TotalMessages++
			h.stats.LastBroadcastTime = time.Now()
			h.mu.Unlock()
			
			// Send message to all connected clients
			h.broadcastToClients(message)
			
			log.Printf("Broadcasted message to %d clients (total broadcasts: %d)", 
				clientCount, h.stats.TotalMessages)
		}
	}
}

func (h *Hub) broadcastMessage(message []byte) {
	select {
	case h.broadcast <- message:
		// Message queued successfully
	default:
		// Broadcast channel is full
		log.Printf("Warning: Broadcast channel full, dropping message")
	}
}

// broadcastToClients sends a message to all connected clients
func (h *Hub) broadcastToClients(message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	for client := range h.clients {
		select {
		case client.send <- message:
			// Message sent successfully
		default:
			// Client's send channel is full, close it
			log.Printf("Client %s send buffer full, disconnecting", client.id)
			go func(c *Client) {
				h.unregister <- c
			}(client)
		}
	}
}

// sendWelcomeMessage sends a welcome message to a newly connected client
func (h *Hub) sendWelcomeMessage(client *Client) {
	welcome := map[string]interface{}{
		"type": "WELCOME",
		"data": map[string]interface{}{
			"client_id":     client.id,
			"total_clients": h.stats.TotalClients,
			"server_time":   time.Now().Unix(),
		},
	}
	
	if welcomeJSON, err := json.Marshal(welcome); err == nil {
		select {
		case client.send <- welcomeJSON:
			log.Printf("Sent welcome message to client %s", client.id)
		default:
			log.Printf("Failed to send welcome message to client %s", client.id)
		}
	}
}

// GetStats returns current hub statistics
func (h *Hub) GetStats() HubStats {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.stats
}

// GetClientCount returns the current number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// BroadcastToUser sends a message to clients with a specific user ID
func (h *Hub) BroadcastToUser(userID string, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	sent := 0
	for client := range h.clients {
		if client.userID == userID {
			select {
			case client.send <- message:
				sent++
			default:
				log.Printf("Failed to send message to client %s (user %s)", client.id, userID)
			}
		}
	}
	
	if sent > 0 {
		log.Printf("Sent message to %d clients for user %s", sent, userID)
	}
}