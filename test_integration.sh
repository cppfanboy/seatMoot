#!/bin/bash

echo "🔧 Integration Test Script for Seat Booking System"
echo "================================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if services are running
check_service() {
    local service=$1
    local port=$2
    
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null; then
        echo -e "${GREEN}✓${NC} $service is running on port $port"
        return 0
    else
        echo -e "${RED}✗${NC} $service is not running on port $port"
        return 1
    fi
}

echo ""
echo "Checking services..."
echo "-------------------"

# Check Redis
check_service "Redis" 6379
REDIS_RUNNING=$?

# Check NATS
check_service "NATS" 4222
NATS_RUNNING=$?

# Check Booking Service
check_service "Booking Service" 8080
BOOKING_RUNNING=$?

# Check Edge Server
check_service "Edge Server" 3000
EDGE_RUNNING=$?

echo ""

# If any service is not running, offer to start them
if [ $REDIS_RUNNING -ne 0 ] || [ $NATS_RUNNING -ne 0 ] || [ $BOOKING_RUNNING -ne 0 ] || [ $EDGE_RUNNING -ne 0 ]; then
    echo -e "${YELLOW}Some services are not running. Would you like to start them? (y/n)${NC}"
    read -r response
    
    if [[ "$response" =~ ^[Yy]$ ]]; then
        echo ""
        echo "Starting services..."
        echo "-------------------"
        
        if [ $REDIS_RUNNING -ne 0 ] || [ $NATS_RUNNING -ne 0 ]; then
            echo "Starting Docker containers..."
            docker-compose up -d
            sleep 2
        fi
        
        if [ $BOOKING_RUNNING -ne 0 ]; then
            echo "Starting Booking Service..."
            (cd booking-service && go run . > ../logs/booking-service.log 2>&1 &)
            sleep 2
        fi
        
        if [ $EDGE_RUNNING -ne 0 ]; then
            echo "Starting Edge Server..."
            (cd edge-server && go run . > ../logs/edge-server.log 2>&1 &)
            sleep 2
        fi
        
        echo ""
        echo "Re-checking services..."
        echo "----------------------"
        check_service "Redis" 6379
        check_service "NATS" 4222
        check_service "Booking Service" 8080
        check_service "Edge Server" 3000
    fi
fi

echo ""
echo "Running integration tests..."
echo "----------------------------"

# Test 1: Health check endpoints
echo -n "Testing Booking Service health endpoint... "
if curl -s http://localhost:8080/health | grep -q "ok"; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
fi

echo -n "Testing Edge Server health endpoint... "
if curl -s http://localhost:3000/health | grep -q "ok"; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
fi

# Test 2: Get all seats
echo -n "Testing GET /api/seats... "
SEATS_RESPONSE=$(curl -s http://localhost:8080/api/seats)
if echo "$SEATS_RESPONSE" | grep -q "A1"; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
fi

# Test 3: Select a seat
echo -n "Testing seat selection... "
SELECT_RESPONSE=$(curl -s -X POST http://localhost:8080/api/seats/select \
    -H "Content-Type: application/json" \
    -d '{"seat_id":"A1","user_id":"test-user-1"}')
    
if echo "$SELECT_RESPONSE" | grep -q "error"; then
    # Might already be selected, try another seat
    SELECT_RESPONSE=$(curl -s -X POST http://localhost:8080/api/seats/select \
        -H "Content-Type: application/json" \
        -d '{"seat_id":"B2","user_id":"test-user-1"}')
fi

if echo "$SELECT_RESPONSE" | grep -q "selected"; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    echo "  Response: $SELECT_RESPONSE"
fi

# Test 4: Book a seat
echo -n "Testing seat booking... "
BOOK_RESPONSE=$(curl -s -X POST http://localhost:8080/api/seats/book \
    -H "Content-Type: application/json" \
    -d '{"seat_id":"C3","user_id":"test-user-2"}')
    
if echo "$BOOK_RESPONSE" | grep -q "booked\|already"; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    echo "  Response: $BOOK_RESPONSE"
fi

# Test 5: WebSocket connection
echo -n "Testing WebSocket connection... "
if command -v wscat &> /dev/null; then
    # If wscat is installed, use it for a quick test
    timeout 2 wscat -c ws://localhost:3000/ws 2>&1 | grep -q "Connected" && echo -e "${GREEN}✓${NC}" || echo -e "${RED}✗${NC}"
else
    # Otherwise, just check if the endpoint responds
    if curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/ws | grep -q "426\|400"; then
        echo -e "${GREEN}✓${NC} (WebSocket upgrade required)"
    else
        echo -e "${RED}✗${NC}"
    fi
fi

echo ""
echo "Integration test complete!"
echo ""

# Show logs location
echo "📋 Service logs are available at:"
echo "   - Booking Service: logs/booking-service.log"
echo "   - Edge Server: logs/edge-server.log"
echo ""

# Instructions for manual testing
echo "🧪 For manual WebSocket testing:"
echo "   1. Install wscat: npm install -g wscat"
echo "   2. Connect: wscat -c ws://localhost:3000/ws"
echo "   3. Subscribe: {\"type\":\"SUBSCRIBE\",\"data\":{\"user_id\":\"test-user\"}}"
echo "   4. Select seat: {\"type\":\"SELECT_SEAT\",\"data\":{\"seat_id\":\"A1\",\"user_id\":\"test-user\"}}"
echo ""