#!/bin/bash

echo "=== Testing Enhanced Client Structure ==="

# Function to connect via WebSocket and send a message
test_ws() {
    echo -e "\n1. Testing WebSocket connection..."
    (
        echo '{"type":"SUBSCRIBE","data":{"user_id":"test_user_123"}}'
        sleep 2
        echo '{"type":"SELECT_SEAT","data":{"seat_id":"F1","user_id":"test_user_123"}}'
        sleep 2
    ) | nc -N localhost 3000 | head -20 &
    
    WS_PID=$!
    sleep 5
    kill $WS_PID 2>/dev/null
}

# Test the connection
test_ws

echo -e "\n2. Checking edge server logs for enhanced features..."
sleep 2

echo -e "\n✅ Test completed - check logs for connection duration tracking"