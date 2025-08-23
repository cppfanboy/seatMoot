package main

import (
	"fmt"
	"log"
	"time"

	"concert-booking/shared"
)


// Response types for client feedback
type OperationResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (c *Client) handleSubscribe(data map[string]interface{}) {
	// Extract user ID if provided
	if userID, ok := data["user_id"].(string); ok && userID != "" {
		c.userID = userID
		c.lastActivity = time.Now()
		log.Printf("[SUBSCRIBE] Client %s subscribed as user %s", c.id, c.userID)
	} else {
		log.Printf("[SUBSCRIBE] Client %s subscribed without user ID", c.id)
	}

	// Send acknowledgment
	c.sendMessage("SUBSCRIBE_ACK", OperationResponse{
		Success: true,
		Message: "Subscribed successfully",
		Data: map[string]interface{}{
			"client_id": c.id,
			"user_id":   c.userID,
		},
	})

	// Send current venue state
	c.sendVenueState()
}

func (c *Client) handleSelectSeat(data map[string]interface{}) {
	seatID, ok := data["seat_id"].(string)
	if !ok || seatID == "" {
		c.sendOperationResponse("SELECT_SEAT_RESPONSE", false, "seat_id is required", nil)
		return
	}

	userID := c.userID
	if uid, ok := data["user_id"].(string); ok && uid != "" {
		userID = uid
	}
	if userID == "" {
		c.sendOperationResponse("SELECT_SEAT_RESPONSE", false, "user_id is required", nil)
		return
	}

	// Update activity
	c.lastActivity = time.Now()

	// Call booking service API
	err := bookingClient.SelectSeat(seatID, userID)
	if err != nil {
		log.Printf("[ERROR] Failed to select seat %s for user %s: %v", seatID, userID, err)
		c.sendOperationResponse("SELECT_SEAT_RESPONSE", false, err.Error(), nil)
		return
	}

	// Success - send immediate confirmation
	c.sendOperationResponse("SELECT_SEAT_RESPONSE", true, 
		fmt.Sprintf("Seat %s selected successfully", seatID), 
		map[string]string{"seat_id": seatID, "user_id": userID})
	
	log.Printf("[SELECT] Client %s (user %s) selected seat %s", c.id, userID, seatID)
}

func (c *Client) handleBookSeat(data map[string]interface{}) {
	seatID, ok := data["seat_id"].(string)
	if !ok || seatID == "" {
		c.sendOperationResponse("BOOK_SEAT_RESPONSE", false, "seat_id is required", nil)
		return
	}

	userID := c.userID
	if uid, ok := data["user_id"].(string); ok && uid != "" {
		userID = uid
	}
	if userID == "" {
		c.sendOperationResponse("BOOK_SEAT_RESPONSE", false, "user_id is required", nil)
		return
	}

	// Update activity
	c.lastActivity = time.Now()

	// Call booking service API
	err := bookingClient.BookSeat(seatID, userID)
	if err != nil {
		log.Printf("[ERROR] Failed to book seat %s for user %s: %v", seatID, userID, err)
		c.sendOperationResponse("BOOK_SEAT_RESPONSE", false, err.Error(), nil)
		return
	}

	// Success - send immediate confirmation
	c.sendOperationResponse("BOOK_SEAT_RESPONSE", true, 
		fmt.Sprintf("Seat %s booked successfully", seatID), 
		map[string]string{"seat_id": seatID, "user_id": userID})
	
	log.Printf("[BOOK] Client %s (user %s) booked seat %s", c.id, userID, seatID)
}

func (c *Client) handleReleaseSeat(data map[string]interface{}) {
	seatID, ok := data["seat_id"].(string)
	if !ok || seatID == "" {
		c.sendOperationResponse("RELEASE_SEAT_RESPONSE", false, "seat_id is required", nil)
		return
	}

	userID := c.userID
	if uid, ok := data["user_id"].(string); ok && uid != "" {
		userID = uid
	}
	if userID == "" {
		c.sendOperationResponse("RELEASE_SEAT_RESPONSE", false, "user_id is required", nil)
		return
	}

	// Update activity
	c.lastActivity = time.Now()

	// Call booking service API
	err := bookingClient.ReleaseSeat(seatID, userID)
	if err != nil {
		log.Printf("[ERROR] Failed to release seat %s for user %s: %v", seatID, userID, err)
		c.sendOperationResponse("RELEASE_SEAT_RESPONSE", false, err.Error(), nil)
		return
	}

	// Success - send immediate confirmation
	c.sendOperationResponse("RELEASE_SEAT_RESPONSE", true, 
		fmt.Sprintf("Seat %s released successfully", seatID), 
		map[string]string{"seat_id": seatID, "user_id": userID})
	
	log.Printf("[RELEASE] Client %s (user %s) released seat %s", c.id, userID, seatID)
}

func (c *Client) sendVenueState() {
	// Get all seats from booking service
	seats, err := bookingClient.GetAllSeats()
	if err != nil {
		log.Printf("[ERROR] Failed to get venue state for client %s: %v", c.id, err)
		c.sendOperationResponse("VENUE_STATE_ERROR", false, "Failed to load venue state", nil)
		return
	}

	// Send venue state to client
	c.sendMessage(shared.MessageTypeVenueState, shared.VenueState{Seats: seats})
	log.Printf("[VENUE] Sent venue state to client %s (%d seats)", c.id, len(seats))
}


// sendOperationResponse sends a structured response to the client
func (c *Client) sendOperationResponse(msgType string, success bool, message string, data interface{}) {
	c.sendMessage(msgType, OperationResponse{
		Success: success,
		Message: message,
		Data:    data,
	})
}