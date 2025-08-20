package shared

import "time"

// Redis key patterns
const (
	RedisKeyVenueSeats = "venue:seats"
	RedisKeySeatLock   = "seat:%s:lock" // formatted with seat ID
)

// NATS topics
const (
	NATSTopicSeatHeld     = "seats.held"
	NATSTopicSeatReleased = "seats.released"
	NATSTopicSeatBooked   = "seats.booked"
	NATSTopicAllSeats     = "seats.>"
)

// Timeouts and durations
const (
	HoldDuration        = 30 * time.Second
	TimerCheckInterval  = 2 * time.Second
	WebSocketReadTimeout = 60 * time.Second
	WebSocketWriteTimeout = 10 * time.Second
	WebSocketPongWait   = 60 * time.Second
	WebSocketPingPeriod = (WebSocketPongWait * 9) / 10
)

// Venue configuration
const (
	VenueRows = 10
	VenueCols = 10
	TotalSeats = VenueRows * VenueCols
)

// Server configuration
const (
	BookingServicePort = ":8080"
	DefaultEdgePort    = ":3000"
)

// API endpoints
const (
	APIEndpointSeats       = "/api/seats"
	APIEndpointSelectSeat  = "/api/seats/select"
	APIEndpointBookSeat    = "/api/seats/book"
	APIEndpointReleaseSeat = "/api/seats/release"
	APIEndpointHealth      = "/health"
	WebSocketEndpoint      = "/ws"
)

// GetSeatID generates a seat ID from row and column
func GetSeatID(row, col int) string {
	rowLetter := string(rune('A' + row))
	return rowLetter + string(rune('1' + col))
}