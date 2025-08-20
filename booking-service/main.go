package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"concert-booking/shared"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/nats-io/nats.go"
)

var (
	redisClient *redis.Client
	natsConn    *nats.Conn
	ctx         = context.Background()
)

func main() {
	log.Println("Starting booking service...")

	// Connect to Redis
	if err := connectRedis(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()
	log.Println("Connected to Redis")

	// Connect to NATS
	if err := connectNATS(); err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer natsConn.Close()
	log.Println("Connected to NATS")

	// Initialize venue with 100 seats
	if err := initializeVenue(); err != nil {
		log.Fatalf("Failed to initialize venue: %v", err)
	}
	log.Println("Venue initialized with", shared.TotalSeats, "seats")

	// Setup Gin router
	router := setupRoutes()

	// Start timer service for auto-releasing held seats
	StartTimerService(redisClient, natsConn)
	log.Println("Timer service started")

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down booking service...")
		os.Exit(0)
	}()

	// Start server
	log.Printf("Booking service started on %s\n", shared.BookingServicePort)
	if err := router.Run(shared.BookingServicePort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func connectRedis() error {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	// Test connection
	_, err := redisClient.Ping(ctx).Result()
	return err
}

func connectNATS() error {
	var err error
	natsConn, err = nats.Connect(nats.DefaultURL)
	return err
}

func initializeVenue() error {
	// Check if venue already initialized
	exists, err := redisClient.Exists(ctx, shared.RedisKeyVenueSeats).Result()
	if err != nil {
		return err
	}

	if exists > 0 {
		log.Println("Venue already initialized, skipping...")
		return nil
	}

	// Create all seats
	for row := 0; row < shared.VenueRows; row++ {
		for col := 0; col < shared.VenueCols; col++ {
			seatID := shared.GetSeatID(row, col)
			seat := shared.Seat{
				ID:     seatID,
				Row:    row,
				Col:    col,
				Status: shared.SeatAvailable,
			}

			seatJSON, err := json.Marshal(seat)
			if err != nil {
				return err
			}

			if err := redisClient.HSet(ctx, shared.RedisKeyVenueSeats, seatID, seatJSON).Err(); err != nil {
				return err
			}
		}
	}

	log.Printf("Initialized %d seats (A1 to J10)\n", shared.TotalSeats)
	return nil
}

func setupRoutes() *gin.Engine {
	router := gin.Default()

	// API routes
	api := router.Group("/api")
	{
		api.GET("/seats", handleGetSeats)
		api.POST("/seats/select", handleSelectSeat)
		api.POST("/seats/book", handleBookSeat)
		api.POST("/seats/release", handleReleaseSeat)
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	return router
}