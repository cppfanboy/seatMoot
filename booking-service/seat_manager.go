package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"concert-booking/shared"

	"github.com/go-redis/redis/v8"
)

func GetAllSeats() ([]shared.Seat, error) {
	// Fetch all seats from Redis hash
	seatMap, err := redisClient.HGetAll(ctx, shared.RedisKeyVenueSeats).Result()
	if err != nil {
		return nil, err
	}

	seats := make([]shared.Seat, 0, len(seatMap))
	for _, seatJSON := range seatMap {
		var seat shared.Seat
		if err := json.Unmarshal([]byte(seatJSON), &seat); err != nil {
			log.Printf("Error unmarshaling seat: %v", err)
			continue
		}
		seats = append(seats, seat)
	}

	return seats, nil
}

func SelectSeat(seatID, userID string) error {
	// First, try to acquire atomic lock with 30 second TTL
	lockKey := fmt.Sprintf(shared.RedisKeySeatLock, seatID)
	success, err := redisClient.SetNX(ctx, lockKey, userID, shared.HoldDuration).Result()
	if err != nil {
		return err
	}

	if !success {
		// Lock already exists, check who holds it
		holder, _ := redisClient.Get(ctx, lockKey).Result()
		if holder == userID {
			return errors.New("you already hold this seat")
		}
		return errors.New("seat is already held by another user")
	}

	// Lock acquired, now update seat status
	seatJSON, err := redisClient.HGet(ctx, shared.RedisKeyVenueSeats, seatID).Result()
	if err == redis.Nil {
		// Seat doesn't exist, release lock
		redisClient.Del(ctx, lockKey)
		return errors.New("seat not found")
	}
	if err != nil {
		// Error occurred, release lock
		redisClient.Del(ctx, lockKey)
		return err
	}

	var seat shared.Seat
	if err := json.Unmarshal([]byte(seatJSON), &seat); err != nil {
		redisClient.Del(ctx, lockKey)
		return err
	}

	// Check if seat is already booked
	if seat.Status == shared.SeatBooked {
		redisClient.Del(ctx, lockKey)
		return errors.New("seat is already booked")
	}

	// Update seat status to held
	seat.Status = shared.SeatHeld
	seat.HeldBy = userID
	seat.ExpiresAt = time.Now().Add(shared.HoldDuration).Unix()

	updatedJSON, err := json.Marshal(seat)
	if err != nil {
		redisClient.Del(ctx, lockKey)
		return err
	}

	if err := redisClient.HSet(ctx, shared.RedisKeyVenueSeats, seatID, updatedJSON).Err(); err != nil {
		redisClient.Del(ctx, lockKey)
		return err
	}

	// Publish event to NATS
	publishSeatEvent("held", seatID, userID, seat.Status, seat.ExpiresAt)

	log.Printf("Seat %s selected by user %s", seatID, userID)
	return nil
}

func BookSeat(seatID, userID string) error {
	// Check if user holds the lock
	lockKey := fmt.Sprintf(shared.RedisKeySeatLock, seatID)
	holder, err := redisClient.Get(ctx, lockKey).Result()
	if err == redis.Nil {
		return errors.New("seat is not held")
	}
	if err != nil {
		return err
	}

	if holder != userID {
		return errors.New("you do not hold this seat")
	}

	// Get current seat status
	seatJSON, err := redisClient.HGet(ctx, shared.RedisKeyVenueSeats, seatID).Result()
	if err != nil {
		return err
	}

	var seat shared.Seat
	if err := json.Unmarshal([]byte(seatJSON), &seat); err != nil {
		return err
	}

	// Verify seat is held by this user
	if seat.Status != shared.SeatHeld || seat.HeldBy != userID {
		return errors.New("seat is not held by you")
	}

	// Update seat to booked status
	seat.Status = shared.SeatBooked
	seat.ExpiresAt = 0 // Remove expiration

	updatedJSON, err := json.Marshal(seat)
	if err != nil {
		return err
	}

	// Update seat in Redis
	if err := redisClient.HSet(ctx, shared.RedisKeyVenueSeats, seatID, updatedJSON).Err(); err != nil {
		return err
	}

	// Remove the lock (no longer needed for booked seats)
	redisClient.Del(ctx, lockKey)

	// Publish event to NATS
	publishSeatEvent("booked", seatID, userID, seat.Status, 0)

	log.Printf("Seat %s booked by user %s", seatID, userID)
	return nil
}

func ReleaseSeat(seatID, userID string) error {
	// Check if user holds the lock
	lockKey := fmt.Sprintf(shared.RedisKeySeatLock, seatID)
	holder, err := redisClient.Get(ctx, lockKey).Result()
	if err == redis.Nil {
		return errors.New("seat is not held")
	}
	if err != nil {
		return err
	}

	if holder != userID {
		return errors.New("you do not hold this seat")
	}

	// Get current seat status
	seatJSON, err := redisClient.HGet(ctx, shared.RedisKeyVenueSeats, seatID).Result()
	if err != nil {
		return err
	}

	var seat shared.Seat
	if err := json.Unmarshal([]byte(seatJSON), &seat); err != nil {
		return err
	}

	// Verify seat is held by this user
	if seat.Status != shared.SeatHeld || seat.HeldBy != userID {
		return errors.New("seat is not held by you")
	}

	// Reset seat to available
	seat.Status = shared.SeatAvailable
	seat.HeldBy = ""
	seat.ExpiresAt = 0

	updatedJSON, err := json.Marshal(seat)
	if err != nil {
		return err
	}

	// Update seat in Redis
	if err := redisClient.HSet(ctx, shared.RedisKeyVenueSeats, seatID, updatedJSON).Err(); err != nil {
		return err
	}

	// Remove the lock
	redisClient.Del(ctx, lockKey)

	// Publish event to NATS
	publishSeatEvent("released", seatID, userID, seat.Status, 0)

	log.Printf("Seat %s released by user %s", seatID, userID)
	return nil
}

func publishSeatEvent(eventType string, seatID string, userID string, status int, expiresAt int64) {
	event := shared.SeatEvent{
		Type:      eventType,
		SeatID:    seatID,
		UserID:    userID,
		Status:    status,
		Timestamp: time.Now(),
		ExpiresAt: expiresAt,
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshaling event: %v", err)
		return
	}

	// Determine topic based on event type
	var topic string
	switch eventType {
	case "held":
		topic = shared.NATSTopicSeatHeld
	case "released":
		topic = shared.NATSTopicSeatReleased
	case "booked":
		topic = shared.NATSTopicSeatBooked
	default:
		log.Printf("Unknown event type: %s", eventType)
		return
	}

	if err := natsConn.Publish(topic, eventJSON); err != nil {
		log.Printf("Error publishing to NATS: %v", err)
	} else {
		log.Printf("Published %s event for seat %s", eventType, seatID)
	}
}