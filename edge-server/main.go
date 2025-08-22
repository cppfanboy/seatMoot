package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"concert-booking/shared"

	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
)

var (
	natsConn *nats.Conn
	hub      *Hub
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// Allow connections from any origin for development
			// In production, this should be more restrictive
			return true
		},
	}
)

func main() {
	// Get port from environment variable
	port := os.Getenv("PORT")
	if port == "" {
		port = shared.DefaultEdgePort
	} else {
		port = ":" + port
	}

	log.Printf("Starting edge server on port %s...", port)

	// Connect to NATS
	if err := connectNATS(); err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer natsConn.Close()
	log.Println("Connected to NATS")

	// Initialize hub
	hub = newHub()
	go hub.run()
	log.Println("Hub initialized and running")

	// Subscribe to NATS events
	if err := subscribeToNATS(); err != nil {
		log.Fatalf("Failed to subscribe to NATS: %v", err)
	}
	log.Println("Subscribed to NATS seat events")

	// Setup HTTP routes
	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/stats", handleStats)

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down edge server...")
		os.Exit(0)
	}()

	// Start server
	log.Printf("Edge server started on %s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func connectNATS() error {
	var err error
	natsConn, err = nats.Connect(nats.DefaultURL)
	return err
}

func subscribeToNATS() error {
	// Subscribe to all seat events
	_, err := natsConn.Subscribe(shared.NATSTopicAllSeats, func(msg *nats.Msg) {
		// Broadcast the event to all connected WebSocket clients
		hub.broadcast <- msg.Data
		log.Printf("Received NATS event on %s, broadcasting to %d clients", 
			msg.Subject, len(hub.clients))
	})
	return err
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Create new client
	client := &Client{
		hub:          hub,
		conn:         conn,
		send:         make(chan []byte, 256),
		id:           generateClientID(),
		connectedAt:  time.Now(),
		lastActivity: time.Now(),
	}

	// Register client with hub
	client.hub.register <- client

	// Start client goroutines
	go client.writePump()
	go client.readPump()

	log.Printf("New WebSocket client connected: %s", client.id)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","service":"edge-server"}`))
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	stats := hub.GetStats()
	statsJSON, err := json.Marshal(stats)
	if err != nil {
		http.Error(w, "Failed to get stats", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(statsJSON)
}

func generateClientID() string {
	// Simple ID generation for now
	// In production, use UUID
	return "client-" + generateRandomString(8)
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[randInt(len(charset))]
	}
	return string(b)
}

func randInt(max int) int {
	// Simple random for demo
	// In production, use crypto/rand
	return int(os.Getpid()+len(hub.clients)) % max
}