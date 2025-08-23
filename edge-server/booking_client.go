package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"concert-booking/shared"
)

// BookingClient handles communication with the booking service
type BookingClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewBookingClient creates a new booking service client
func NewBookingClient(baseURL string) *BookingClient {
	return &BookingClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetAllSeats fetches all seats from the booking service
func (bc *BookingClient) GetAllSeats() ([]shared.Seat, error) {
	resp, err := bc.httpClient.Get(bc.baseURL + "/api/seats")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch seats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var seats []shared.Seat
	if err := json.NewDecoder(resp.Body).Decode(&seats); err != nil {
		return nil, fmt.Errorf("failed to decode seats: %w", err)
	}

	return seats, nil
}

// SelectSeat attempts to select a seat for a user
func (bc *BookingClient) SelectSeat(seatID, userID string) error {
	req := shared.SeatRequest{
		SeatID: seatID,
		UserID: userID,
	}

	return bc.postRequest("/api/seats/select", req)
}

// BookSeat attempts to book a seat for a user
func (bc *BookingClient) BookSeat(seatID, userID string) error {
	req := shared.SeatRequest{
		SeatID: seatID,
		UserID: userID,
	}

	return bc.postRequest("/api/seats/book", req)
}

// ReleaseSeat releases a seat held by a user
func (bc *BookingClient) ReleaseSeat(seatID, userID string) error {
	req := shared.SeatRequest{
		SeatID: seatID,
		UserID: userID,
	}

	return bc.postRequest("/api/seats/release", req)
}

// postRequest makes a POST request to the booking service
func (bc *BookingClient) postRequest(endpoint string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", bc.baseURL+endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := bc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp shared.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			return fmt.Errorf(errResp.Error)
		}
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetSeat fetches a single seat by ID
func (bc *BookingClient) GetSeat(seatID string) (*shared.Seat, error) {
	resp, err := bc.httpClient.Get(bc.baseURL + "/api/seats/" + seatID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch seat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("seat not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var seat shared.Seat
	if err := json.NewDecoder(resp.Body).Decode(&seat); err != nil {
		return nil, fmt.Errorf("failed to decode seat: %w", err)
	}

	return &seat, nil
}

// HealthCheck verifies the booking service is available
func (bc *BookingClient) HealthCheck() error {
	resp, err := bc.httpClient.Get(bc.baseURL + "/health")
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unhealthy status: %d", resp.StatusCode)
	}

	return nil
}