package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"concert-booking/shared"
)

const (
	bookingServiceURL = "http://localhost:8080"
	maxRetries        = 3
	retryDelay        = 100 * time.Millisecond
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

	// Call booking service API with retry
	req := shared.SeatRequest{
		SeatID: seatID,
		UserID: userID,
	}

	resp, err := callBookingServiceWithRetry("/api/seats/select", req)
	if err != nil {
		log.Printf("[ERROR] Failed to select seat %s for user %s: %v", seatID, userID, err)
		c.sendOperationResponse("SELECT_SEAT_RESPONSE", false, 
			fmt.Sprintf("Service unavailable: %v", err), nil)
		return
	}
	defer resp.Body.Close()

	// Parse response
	if resp.StatusCode != http.StatusOK {
		var errResp shared.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			c.sendOperationResponse("SELECT_SEAT_RESPONSE", false, errResp.Error, nil)
		} else {
			c.sendOperationResponse("SELECT_SEAT_RESPONSE", false, "Failed to select seat", nil)
		}
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

	// Call booking service API with retry
	req := shared.SeatRequest{
		SeatID: seatID,
		UserID: userID,
	}

	resp, err := callBookingServiceWithRetry("/api/seats/book", req)
	if err != nil {
		log.Printf("[ERROR] Failed to book seat %s for user %s: %v", seatID, userID, err)
		c.sendOperationResponse("BOOK_SEAT_RESPONSE", false, 
			fmt.Sprintf("Service unavailable: %v", err), nil)
		return
	}
	defer resp.Body.Close()

	// Parse response
	if resp.StatusCode != http.StatusOK {
		var errResp shared.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			c.sendOperationResponse("BOOK_SEAT_RESPONSE", false, errResp.Error, nil)
		} else {
			c.sendOperationResponse("BOOK_SEAT_RESPONSE", false, "Failed to book seat", nil)
		}
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

	// Call booking service API with retry
	req := shared.SeatRequest{
		SeatID: seatID,
		UserID: userID,
	}

	resp, err := callBookingServiceWithRetry("/api/seats/release", req)
	if err != nil {
		log.Printf("[ERROR] Failed to release seat %s for user %s: %v", seatID, userID, err)
		c.sendOperationResponse("RELEASE_SEAT_RESPONSE", false, 
			fmt.Sprintf("Service unavailable: %v", err), nil)
		return
	}
	defer resp.Body.Close()

	// Parse response
	if resp.StatusCode != http.StatusOK {
		var errResp shared.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			c.sendOperationResponse("RELEASE_SEAT_RESPONSE", false, errResp.Error, nil)
		} else {
			c.sendOperationResponse("RELEASE_SEAT_RESPONSE", false, "Failed to release seat", nil)
		}
		return
	}

	// Success - send immediate confirmation
	c.sendOperationResponse("RELEASE_SEAT_RESPONSE", true, 
		fmt.Sprintf("Seat %s released successfully", seatID), 
		map[string]string{"seat_id": seatID, "user_id": userID})
	
	log.Printf("[RELEASE] Client %s (user %s) released seat %s", c.id, userID, seatID)
}

func (c *Client) sendVenueState() {
	// Get all seats from booking service with timeout
	client := &http.Client{
		Timeout: 3 * time.Second,
	}
	
	resp, err := client.Get(bookingServiceURL + "/api/seats")
	if err != nil {
		log.Printf("[ERROR] Failed to get venue state for client %s: %v", c.id, err)
		c.sendOperationResponse("VENUE_STATE_ERROR", false, "Failed to load venue state", nil)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[ERROR] Venue state request failed with status %d", resp.StatusCode)
		c.sendOperationResponse("VENUE_STATE_ERROR", false, "Failed to load venue state", nil)
		return
	}

	var seats []shared.Seat
	if err := json.NewDecoder(resp.Body).Decode(&seats); err != nil {
		log.Printf("[ERROR] Failed to decode venue state: %v", err)
		c.sendOperationResponse("VENUE_STATE_ERROR", false, "Invalid venue data", nil)
		return
	}

	// Send venue state to client
	c.sendMessage(shared.MessageTypeVenueState, shared.VenueState{Seats: seats})
	log.Printf("[VENUE] Sent venue state to client %s (%d seats)", c.id, len(seats))
}

func callBookingService(endpoint string, data interface{}) (*http.Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("POST", bookingServiceURL+endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	return client.Do(req)
}

// callBookingServiceWithRetry attempts to call the booking service with retries
func callBookingServiceWithRetry(endpoint string, data interface{}) (*http.Response, error) {
	var lastErr error
	
	for i := 0; i < maxRetries; i++ {
		resp, err := callBookingService(endpoint, data)
		if err == nil {
			return resp, nil
		}
		
		lastErr = err
		if i < maxRetries-1 {
			log.Printf("[RETRY %d/%d] Failed to call %s: %v", i+1, maxRetries, endpoint, err)
			time.Sleep(retryDelay)
		}
	}
	
	return nil, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

// sendOperationResponse sends a structured response to the client
func (c *Client) sendOperationResponse(msgType string, success bool, message string, data interface{}) {
	c.sendMessage(msgType, OperationResponse{
		Success: success,
		Message: message,
		Data:    data,
	})
}