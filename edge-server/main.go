package main

import (
	"encoding/json"
	"fmt"
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
	natsConn       *nats.Conn
	hub            *Hub
	bookingClient  *BookingClient
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

	// Initialize booking client
	bookingServiceURL := os.Getenv("BOOKING_SERVICE_URL")
	if bookingServiceURL == "" {
		bookingServiceURL = "http://localhost:8080"
	}
	bookingClient = NewBookingClient(bookingServiceURL)
	log.Printf("Booking client initialized with URL: %s", bookingServiceURL)

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
	
	// Connect with options for better reliability
	opts := []nats.Option{
		nats.Name("edge-server"),
		nats.MaxReconnects(-1), // Infinite reconnects
		nats.ReconnectWait(2 * time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("[NATS] Disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("[NATS] Reconnected to %s", nc.ConnectedUrl())
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			log.Printf("[NATS] Error: %v", err)
		}),
	}
	
	natsConn, err = nats.Connect(nats.DefaultURL, opts...)
	if err != nil {
		return err
	}
	
	// Verify connection
	if !natsConn.IsConnected() {
		return fmt.Errorf("NATS connection not established")
	}
	
	return nil
}

func subscribeToNATS() error {
	// Subscribe to all seat events
	subscription, err := natsConn.Subscribe(shared.NATSTopicAllSeats, func(msg *nats.Msg) {
		// Parse the NATS event
		var seatEvent shared.SeatEvent
		if err := json.Unmarshal(msg.Data, &seatEvent); err != nil {
			log.Printf("[ERROR] Failed to parse NATS event: %v", err)
			return
		}
		
		// Convert to WebSocket message format
		wsMessage := shared.ServerMessage{
			Type: shared.MessageTypeSeatUpdate,
			Data: map[string]interface{}{
				"event_type": seatEvent.Type,
				"seat_id":    seatEvent.SeatID,
				"user_id":    seatEvent.UserID,
				"status":     seatEvent.Status,
				"timestamp":  seatEvent.Timestamp,
				"expires_at": seatEvent.ExpiresAt,
				"seat":       seatEvent.Seat,
			},
		}
		
		// Marshal to JSON for WebSocket
		wsMessageJSON, err := json.Marshal(wsMessage)
		if err != nil {
			log.Printf("[ERROR] Failed to marshal WebSocket message: %v", err)
			return
		}
		
		// Broadcast to all connected clients
		hub.broadcastMessage(wsMessageJSON)
		
		log.Printf("[NATS] Received %s event for seat %s on topic %s, broadcasting to %d clients", 
			seatEvent.Type, seatEvent.SeatID, msg.Subject, hub.GetClientCount())
	})
	
	if err != nil {
		return err
	}
	
	log.Printf("[NATS] Subscribed to %s (subscription: %s)", shared.NATSTopicAllSeats, subscription.Subject)
	return nil
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