package main

import (
	"net/http"

	"concert-booking/shared"

	"github.com/gin-gonic/gin"
)

func handleGetSeats(c *gin.Context) {
	seats, err := GetAllSeats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, shared.ErrorResponse{Error: "Failed to get seats"})
		return
	}
	c.JSON(http.StatusOK, seats)
}

func handleSelectSeat(c *gin.Context) {
	var req shared.SeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, shared.ErrorResponse{Error: "Invalid request"})
		return
	}

	if req.SeatID == "" || req.UserID == "" {
		c.JSON(http.StatusBadRequest, shared.ErrorResponse{Error: "seat_id and user_id are required"})
		return
	}

	err := SelectSeat(req.SeatID, req.UserID)
	if err != nil {
		c.JSON(http.StatusConflict, shared.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Seat selected successfully"})
}

func handleBookSeat(c *gin.Context) {
	var req shared.SeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, shared.ErrorResponse{Error: "Invalid request"})
		return
	}

	if req.SeatID == "" || req.UserID == "" {
		c.JSON(http.StatusBadRequest, shared.ErrorResponse{Error: "seat_id and user_id are required"})
		return
	}

	err := BookSeat(req.SeatID, req.UserID)
	if err != nil {
		c.JSON(http.StatusConflict, shared.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Seat booked successfully"})
}

func handleReleaseSeat(c *gin.Context) {
	var req shared.SeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, shared.ErrorResponse{Error: "Invalid request"})
		return
	}

	if req.SeatID == "" || req.UserID == "" {
		c.JSON(http.StatusBadRequest, shared.ErrorResponse{Error: "seat_id and user_id are required"})
		return
	}

	err := ReleaseSeat(req.SeatID, req.UserID)
	if err != nil {
		c.JSON(http.StatusConflict, shared.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Seat released successfully"})
}