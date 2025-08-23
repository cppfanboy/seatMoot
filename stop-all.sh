#!/bin/bash

echo "🛑 Stopping Concert Seat Booking System"
echo "======================================"
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Stop Go services
echo "1. Stopping Go services..."

# Stop using saved PIDs if available
if [ -f .pids/booking.pid ]; then
    kill -9 $(cat .pids/booking.pid) 2>/dev/null && echo "  • Stopped booking service"
fi

if [ -f .pids/edge1.pid ]; then
    kill -9 $(cat .pids/edge1.pid) 2>/dev/null && echo "  • Stopped edge server 1"
fi

if [ -f .pids/edge2.pid ]; then
    kill -9 $(cat .pids/edge2.pid) 2>/dev/null && echo "  • Stopped edge server 2"
fi

# Also kill by port in case PIDs are stale
lsof -ti:8080 2>/dev/null | xargs kill -9 2>/dev/null
lsof -ti:3000 2>/dev/null | xargs kill -9 2>/dev/null
lsof -ti:3001 2>/dev/null | xargs kill -9 2>/dev/null

# Stop Python frontend server if running
lsof -ti:8000 2>/dev/null | xargs kill -9 2>/dev/null && echo "  • Stopped frontend server"

echo -e "${GREEN}✅ Go services stopped${NC}"

# Stop Docker containers
echo ""
echo "2. Stopping Docker containers..."
docker-compose down

# Clean up PID files
rm -rf .pids/*.pid 2>/dev/null

echo ""
echo -e "${GREEN}✅ All services stopped${NC}"
echo ""

# Check if anything is still running
echo "Checking for remaining processes..."
REMAINING=0

if lsof -Pi :80 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  Something still running on port 80${NC}"
    REMAINING=1
fi

if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  Something still running on port 8080${NC}"
    REMAINING=1
fi

if lsof -Pi :3000 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  Something still running on port 3000${NC}"
    REMAINING=1
fi

if lsof -Pi :3001 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  Something still running on port 3001${NC}"
    REMAINING=1
fi

if [ $REMAINING -eq 0 ]; then
    echo -e "${GREEN}✅ All ports are clear${NC}"
fi

echo ""
echo "System shutdown complete."
echo ""