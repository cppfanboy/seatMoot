#!/bin/bash

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo "🎭 Starting Concert Seat Booking System"
echo "======================================"
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running${NC}"
    echo "Please start Docker Desktop first"
    exit 1
fi

# Clean up any existing processes
echo "Cleaning up existing processes..."
lsof -ti:8080,3000,3001 2>/dev/null | xargs kill -9 2>/dev/null

# Start infrastructure
echo ""
echo "1. Starting infrastructure (Redis, NATS, NGINX)..."
docker-compose up -d
sleep 3

# Check if containers are running
if ! docker ps | grep -q "redis"; then
    echo -e "${RED}❌ Redis failed to start${NC}"
    exit 1
fi

if ! docker ps | grep -q "nats"; then
    echo -e "${RED}❌ NATS failed to start${NC}"
    exit 1
fi

if ! docker ps | grep -q "nginx"; then
    echo -e "${RED}❌ NGINX failed to start${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Infrastructure started${NC}"

# Start booking service
echo ""
echo "2. Starting booking service on port 8080..."
(cd "$SCRIPT_DIR/booking-service" && go run . > "$SCRIPT_DIR/logs/booking-service.log" 2>&1) &
BOOKING_PID=$!
sleep 2

# Check if booking service started
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Booking service started (PID: $BOOKING_PID)${NC}"
else
    echo -e "${RED}❌ Booking service failed to start${NC}"
    echo "Check logs/booking-service.log for errors"
    exit 1
fi

# Start edge servers
echo ""
echo "3. Starting edge servers..."

# Edge server 1 on port 3000
(cd "$SCRIPT_DIR/edge-server" && PORT=3000 BOOKING_SERVICE_URL=http://localhost:8080 go run . > "$SCRIPT_DIR/logs/edge-server-3000.log" 2>&1) &
EDGE1_PID=$!
sleep 2

# Edge server 2 on port 3001  
(cd "$SCRIPT_DIR/edge-server" && PORT=3001 BOOKING_SERVICE_URL=http://localhost:8080 go run . > "$SCRIPT_DIR/logs/edge-server-3001.log" 2>&1) &
EDGE2_PID=$!
sleep 2

# Check if edge servers started
if lsof -Pi :3000 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Edge server 1 started on port 3000 (PID: $EDGE1_PID)${NC}"
else
    echo -e "${RED}❌ Edge server 1 failed to start${NC}"
    echo "Check logs/edge-server-3000.log for errors"
fi

if lsof -Pi :3001 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Edge server 2 started on port 3001 (PID: $EDGE2_PID)${NC}"
else
    echo -e "${RED}❌ Edge server 2 failed to start${NC}"
    echo "Check logs/edge-server-3001.log for errors"
fi

# Save PIDs for stop script
mkdir -p .pids
echo "$BOOKING_PID" > .pids/booking.pid
echo "$EDGE1_PID" > .pids/edge1.pid
echo "$EDGE2_PID" > .pids/edge2.pid

# System status
echo ""
echo "======================================"
echo -e "${GREEN}✨ System is running!${NC}"
echo ""
echo "📌 Access Points:"
echo "  • Frontend: http://localhost (via NGINX)"
echo "  • Direct Frontend: http://localhost:8000 (Python server)"
echo "  • API: http://localhost/api/seats"
echo "  • Health: http://localhost/health"
echo "  • Stats: http://localhost/stats"
echo "  • NGINX Health: http://localhost/nginx-health"
echo ""
echo "📊 Services:"
echo "  • NGINX: Port 80 (Load Balancer)"
echo "  • Booking Service: Port 8080"
echo "  • Edge Server 1: Port 3000"
echo "  • Edge Server 2: Port 3001"
echo "  • Redis: Port 6379"
echo "  • NATS: Port 4222"
echo ""
echo "📁 Logs:"
echo "  • logs/booking-service.log"
echo "  • logs/edge-server-3000.log"
echo "  • logs/edge-server-3001.log"
echo ""
echo "To stop all services, run: ./stop-all.sh"
echo ""