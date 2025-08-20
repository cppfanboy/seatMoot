package main

import (
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
}

func checkExpiredHolds(redisClient *redis.Client, natsConn *nats.Conn) {
	// This will be implemented in Day 3
	// For now, just a placeholder
	log.Println("Checking for expired holds...")
}