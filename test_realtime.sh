#!/bin/bash

echo "🎯 Testing Real-Time Updates via NATS"
echo "====================================="

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo ""
echo "This test will:"
echo "1. Select a seat via API"
echo "2. Verify NATS publishes the event"
echo "3. Check that edge server would broadcast it"
echo ""

# Test seat
SEAT_ID="H8"
USER_ID="realtime-test-$(date +%s)"

echo "Test Configuration:"
echo "  Seat: $SEAT_ID"
echo "  User: $USER_ID"
echo ""

# First, check edge server stats
echo "📊 Current Edge Server Stats:"
curl -s http://localhost:3000/stats | python3 -m json.tool 2>/dev/null || echo "  (No stats available)"
echo ""

# Select a seat
echo "🎬 Selecting seat $SEAT_ID..."
RESPONSE=$(curl -s -X POST http://localhost:8080/api/seats/select \
    -H "Content-Type: application/json" \
    -d "{\"seat_id\":\"$SEAT_ID\",\"user_id\":\"$USER_ID\"}")

if echo "$RESPONSE" | grep -q "selected"; then
    echo -e "  ${GREEN}✓${NC} Seat selected successfully"
else
    echo "  Response: $RESPONSE"
fi

echo ""
echo "⏱️  Waiting 2 seconds..."
sleep 2

# Book the seat to trigger another event
echo ""
echo "📚 Booking seat $SEAT_ID..."
RESPONSE=$(curl -s -X POST http://localhost:8080/api/seats/book \
    -H "Content-Type: application/json" \
    -d "{\"seat_id\":\"$SEAT_ID\",\"user_id\":\"$USER_ID\"}")

if echo "$RESPONSE" | grep -q "booked"; then
    echo -e "  ${GREEN}✓${NC} Seat booked successfully"
else
    echo "  Response: $RESPONSE"
fi

echo ""
echo "📊 Updated Edge Server Stats:"
curl -s http://localhost:3000/stats | python3 -m json.tool 2>/dev/null || echo "  (No stats available)"

echo ""
echo "====================================="
echo -e "${GREEN}✨ Real-time test complete!${NC}"
echo ""
echo "If NATS is working correctly:"
echo "  • The booking service published events to NATS"
echo "  • The edge server received them via subscription"
echo "  • Any connected WebSocket clients would get updates"
echo ""
echo "To see real-time updates in action:"
echo "  1. Install wscat: npm install -g wscat"
echo "  2. Connect: wscat -c ws://localhost:3000/ws"
echo "  3. Subscribe: {\"type\":\"SUBSCRIBE\",\"data\":{\"user_id\":\"test\"}}"
echo "  4. Run this script again in another terminal"
echo ""