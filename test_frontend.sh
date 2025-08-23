#!/bin/bash

echo "🎭 Frontend Multi-Browser Test"
echo "=============================="
echo ""

# Check services
echo "Checking services..."
echo "-------------------"

check_service() {
    local service=$1
    local port=$2
    
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo "✅ $service is running on port $port"
        return 0
    else
        echo "❌ $service is not running on port $port"
        return 1
    fi
}

check_service "Frontend Server" 8000
FRONTEND=$?

check_service "Edge Server" 3000
EDGE=$?

check_service "Booking Service" 8080
BOOKING=$?

check_service "Redis" 6379
REDIS=$?

check_service "NATS" 4222
NATS=$?

echo ""

if [ $FRONTEND -eq 0 ] && [ $EDGE -eq 0 ] && [ $BOOKING -eq 0 ] && [ $REDIS -eq 0 ] && [ $NATS -eq 0 ]; then
    echo "✨ All services are running!"
    echo ""
    echo "📱 Open these URLs in different browsers/tabs to test:"
    echo "------------------------------------------------------"
    echo ""
    echo "Browser 1: http://localhost:8000"
    echo "Browser 2: http://localhost:8000 (Private/Incognito mode)"
    echo ""
    echo "🧪 Test Instructions:"
    echo "--------------------"
    echo "1. Open both browser windows side by side"
    echo "2. In Browser 1: Click on any available (green) seat"
    echo "3. Observe: The seat turns yellow in BOTH browsers"
    echo "4. In Browser 1: Click 'Book Seat' button"
    echo "5. Observe: The seat turns red in BOTH browsers"
    echo "6. In Browser 2: Try to select the same seat"
    echo "7. Observe: You should see an error message"
    echo "8. Wait 30 seconds after selecting a seat (don't book)"
    echo "9. Observe: The seat auto-releases and turns green in BOTH browsers"
    echo ""
    echo "🎯 Expected Results:"
    echo "-------------------"
    echo "• Real-time updates across all browsers"
    echo "• Seat colors sync instantly"
    echo "• Cannot select seats held by others"
    echo "• Auto-release after 30 seconds"
    echo "• Unique user IDs per browser session"
    echo ""
else
    echo "⚠️  Some services are not running!"
    echo ""
    echo "To start all services, run:"
    echo "  docker-compose up -d                    # Start Redis & NATS"
    echo "  cd booking-service && go run . &        # Start Booking Service"
    echo "  cd edge-server && go run . &            # Start Edge Server"
    echo "  cd frontend && python3 server.py &      # Start Frontend"
fi

echo ""
echo "Press Ctrl+C to exit"