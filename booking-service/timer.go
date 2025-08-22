package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"concert-booking/shared"

	"github.com/go-redis/redis/v8"
	"github.com/nats-io/nats.go"
)

func StartTimerService(redisClient *redis.Client, natsConn *nats.Conn) {
	ticker := time.NewTicker(shared.TimerCheckInterval)
	go func() {
		for range ticker.C {
			checkExpiredHolds(redisClient, natsConn)
		}
	}()
	log.Println("Timer service started - checking every", shared.TimerCheckInterval)
}

func checkExpiredHolds(redisClient *redis.Client, natsConn *nats.Conn) {
	ctx := context.Background()
	currentTime := time.Now().Unix()
	expiredCount := 0
	
	// Get all seats from Redis
	seatMap, err := redisClient.HGetAll(ctx, shared.RedisKeyVenueSeats).Result()
	if err != nil {
		log.Printf("Error fetching seats for timer check: %v", err)
		return
	}

	// Check each seat for expiration
	for seatID, seatJSON := range seatMap {
		var seat shared.Seat
		if err := json.Unmarshal([]byte(seatJSON), &seat); err != nil {
			log.Printf("Error unmarshaling seat %s: %v", seatID, err)
			continue
		}

		// Only check held seats with expiration times
		if seat.Status == shared.SeatHeld && seat.ExpiresAt > 0 && seat.ExpiresAt < currentTime {
			// This seat has expired, release it
			if err := autoReleaseSeat(redisClient, natsConn, &seat); err != nil {
				log.Printf("Error auto-releasing seat %s: %v", seatID, err)
				continue
			}
			expiredCount++
			log.Printf("Auto-released expired seat %s (was held by %s)", seat.ID, seat.HeldBy)
		}
	}
	
	if expiredCount > 0 {
		log.Printf("Timer: Released %d expired holds", expiredCount)
	}
}

func autoReleaseSeat(redisClient *redis.Client, natsConn *nats.Conn, seat *shared.Seat) error {
	ctx := context.Background()
	
	// Check if lock still exists (it should have expired naturally)
	lockKey := fmt.Sprintf(shared.RedisKeySeatLock, seat.ID)
	lockExists, _ := redisClient.Exists(ctx, lockKey).Result()
	
	// If lock still exists somehow, delete it
	if lockExists > 0 {
		redisClient.Del(ctx, lockKey)
	}
	
	// Reset seat to available status
	previousHolder := seat.HeldBy
	seat.Status = shared.SeatAvailable
	seat.HeldBy = ""
	seat.ExpiresAt = 0
	
	// Update seat in Redis
	updatedJSON, err := json.Marshal(seat)
	if err != nil {
		return err
	}
	
	if err := redisClient.HSet(ctx, shared.RedisKeyVenueSeats, seat.ID, updatedJSON).Err(); err != nil {
		return err
	}
	
	// Publish release event to NATS with full seat data
	event := shared.SeatEvent{
		Type:      "auto_released",
		SeatID:    seat.ID,
		UserID:    previousHolder,
		Status:    seat.Status,
		Timestamp: time.Now(),
		ExpiresAt: 0,
		Seat:      seat,
	}
	
	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Printf("[ERROR] Failed to marshal auto-release event for seat %s: %v", seat.ID, err)
		return nil // Don't fail the release just because of event publishing
	}
	
	// Publish with retry
	maxRetries := 3
	published := false
	for i := 0; i < maxRetries; i++ {
		if err := natsConn.Publish(shared.NATSTopicSeatReleased, eventJSON); err != nil {
			if i == maxRetries-1 {
				log.Printf("[ERROR] Failed to publish auto-release event for seat %s after %d attempts: %v", 
					seat.ID, maxRetries, err)
			} else {
				log.Printf("[WARN] Retry %d/%d: Failed to publish auto-release event: %v", 
					i+1, maxRetries, err)
				time.Sleep(100 * time.Millisecond)
			}
		} else {
			log.Printf("[INFO] Published auto-release event for seat %s (was held by %s)", 
				seat.ID, previousHolder)
			published = true
			break
		}
	}
	
	if !published {
		// Log failure but don't fail the operation
		log.Printf("[WARN] Seat %s was released but event notification failed", seat.ID)
	}
	
	return nil
}