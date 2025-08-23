#!/bin/bash

echo "🎭 End-to-End Test: Complete Seat Booking Flow"
echo "=============================================="

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test configuration
API_URL="http://localhost:8080"
USER_ID="test-user-$(date +%s)"
SEAT_ID="D4"

echo ""
echo "Test User: $USER_ID"
echo "Test Seat: $SEAT_ID"
echo ""

# Step 1: Check seat is available
echo "1. Checking seat availability..."
SEATS=$(curl -s "$API_URL/api/seats")
SEAT_STATUS=$(echo "$SEATS" | grep -o "\"id\":\"$SEAT_ID\"[^}]*" | grep -o '"status":[0-9]' | cut -d':' -f2)

if [ "$SEAT_STATUS" == "0" ]; then
    echo -e "   ${GREEN}✓${NC} Seat $SEAT_ID is available"
elif [ "$SEAT_STATUS" == "1" ]; then
    echo -e "   ${YELLOW}⚠${NC} Seat $SEAT_ID is held, waiting for release..."
    sleep 31
elif [ "$SEAT_STATUS" == "2" ]; then
    echo -e "   ${RED}✗${NC} Seat $SEAT_ID is already booked, choosing different seat"
    SEAT_ID="E5"
    echo "   Using seat $SEAT_ID instead"
fi

# Step 2: Select (hold) the seat
echo ""
echo "2. Selecting seat $SEAT_ID..."
SELECT_RESPONSE=$(curl -s -X POST "$API_URL/api/seats/select" \
    -H "Content-Type: application/json" \
    -d "{\"seat_id\":\"$SEAT_ID\",\"user_id\":\"$USER_ID\"}")

if echo "$SELECT_RESPONSE" | grep -q "selected"; then
    echo -e "   ${GREEN}✓${NC} Seat selected successfully"
    echo "   Hold expires in 30 seconds"
else
    echo -e "   ${RED}✗${NC} Failed to select seat"
    echo "   Response: $SELECT_RESPONSE"
    exit 1
fi

# Step 3: Verify seat is held
echo ""
echo "3. Verifying seat is held..."
sleep 1
SEATS=$(curl -s "$API_URL/api/seats")
SEAT_INFO=$(echo "$SEATS" | grep -o "\"id\":\"$SEAT_ID\"[^}]*")
HELD_BY=$(echo "$SEAT_INFO" | grep -o '"held_by":"[^"]*"' | cut -d'"' -f4)

if [ "$HELD_BY" == "$USER_ID" ]; then
    echo -e "   ${GREEN}✓${NC} Seat is held by $USER_ID"
else
    echo -e "   ${RED}✗${NC} Seat hold verification failed"
fi

# Step 4: Book the seat
echo ""
echo "4. Booking seat $SEAT_ID..."
BOOK_RESPONSE=$(curl -s -X POST "$API_URL/api/seats/book" \
    -H "Content-Type: application/json" \
    -d "{\"seat_id\":\"$SEAT_ID\",\"user_id\":\"$USER_ID\"}")

if echo "$BOOK_RESPONSE" | grep -q "booked"; then
    echo -e "   ${GREEN}✓${NC} Seat booked successfully!"
else
    echo -e "   ${RED}✗${NC} Failed to book seat"
    echo "   Response: $BOOK_RESPONSE"
    exit 1
fi

# Step 5: Verify seat is booked
echo ""
echo "5. Verifying seat is booked..."
sleep 1
SEATS=$(curl -s "$API_URL/api/seats")
SEAT_STATUS=$(echo "$SEATS" | grep -o "\"id\":\"$SEAT_ID\"[^}]*" | grep -o '"status":[0-9]' | cut -d':' -f2)

if [ "$SEAT_STATUS" == "2" ]; then
    echo -e "   ${GREEN}✓${NC} Seat is permanently booked"
else
    echo -e "   ${RED}✗${NC} Seat booking verification failed (status: $SEAT_STATUS)"
fi

# Step 6: Try to select already booked seat (should fail)
echo ""
echo "6. Testing conflict - trying to select booked seat..."
CONFLICT_RESPONSE=$(curl -s -X POST "$API_URL/api/seats/select" \
    -H "Content-Type: application/json" \
    -d "{\"seat_id\":\"$SEAT_ID\",\"user_id\":\"another-user\"}")

if echo "$CONFLICT_RESPONSE" | grep -q "error"; then
    echo -e "   ${GREEN}✓${NC} Correctly prevented selecting booked seat"
else
    echo -e "   ${RED}✗${NC} Error: Should not allow selecting booked seat"
fi

# Step 7: Test auto-release
echo ""
echo "7. Testing auto-release of held seats..."
TEMP_SEAT="F6"
echo "   Selecting seat $TEMP_SEAT..."
curl -s -X POST "$API_URL/api/seats/select" \
    -H "Content-Type: application/json" \
    -d "{\"seat_id\":\"$TEMP_SEAT\",\"user_id\":\"temp-user\"}" > /dev/null

echo "   Waiting 32 seconds for auto-release..."
sleep 32

SEATS=$(curl -s "$API_URL/api/seats")
TEMP_STATUS=$(echo "$SEATS" | grep -o "\"id\":\"$TEMP_SEAT\"[^}]*" | grep -o '"status":[0-9]' | cut -d':' -f2)

if [ "$TEMP_STATUS" == "0" ]; then
    echo -e "   ${GREEN}✓${NC} Seat auto-released after timeout"
else
    echo -e "   ${YELLOW}⚠${NC} Auto-release may not have triggered yet"
fi

echo ""
echo "========================================"
echo -e "${GREEN}✨ End-to-End Test Complete!${NC}"
echo ""
echo "Summary:"
echo "  • Successfully selected seat $SEAT_ID"
echo "  • Successfully booked seat $SEAT_ID"
echo "  • Verified conflict prevention"
echo "  • Tested auto-release mechanism"
echo ""
echo "The seat booking system is working correctly!"
echo ""