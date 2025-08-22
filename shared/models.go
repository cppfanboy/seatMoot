package shared

import "time"

// Seat statuses
const (
	SeatAvailable = 0
	SeatHeld      = 1
	SeatBooked    = 2
)

// Seat represents a single seat in the venue
type Seat struct {
	ID        string `json:"id"`
	Row       int    `json:"row"`
	Col       int    `json:"col"`
	Status    int    `json:"status"`
	HeldBy    string `json:"held_by,omitempty"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
}

// Message types for WebSocket communication
const (
	MessageTypeConnect     = "CONNECT"
	MessageTypeSelectSeat  = "SELECT_SEAT"
	MessageTypeBookSeat    = "BOOK_SEAT"
	MessageTypeReleaseSeat = "RELEASE_SEAT"
	MessageTypeSeatUpdate  = "SEAT_UPDATE"
	MessageTypeVenueState  = "VENUE_STATE"
	MessageTypeSubscribe   = "SUBSCRIBE"
	MessageTypeError       = "ERROR"
)

// ClientMessage represents a message from the browser to the server
type ClientMessage struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

// ServerMessage represents a message from the server to the browser
type ServerMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// SeatRequest represents a request to select, book, or release a seat
type SeatRequest struct {
	SeatID string `json:"seat_id"`
	UserID string `json:"user_id"`
}

// SeatEvent represents an event for NATS pub/sub
type SeatEvent struct {
	Type      string    `json:"type"`      // held, released, booked, auto_released
	SeatID    string    `json:"seat_id"`
	UserID    string    `json:"user_id"`
	Status    int       `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	ExpiresAt int64     `json:"expires_at,omitempty"`
	Seat      *Seat     `json:"seat,omitempty"` // Full seat data for venue state updates
}

// VenueState represents the complete state of all seats
type VenueState struct {
	Seats []Seat `json:"seats"`
}

// ErrorResponse represents an error message
type ErrorResponse struct {
	Error string `json:"error"`
}