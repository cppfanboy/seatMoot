# PLAN.md - Real-Time Seat Booking System Implementation

## Project Overview
- **Goal**: Build a real-time seat booking system where multiple users can see instant updates
- **Users**: 100 concurrent users max
- **Seats**: 100 seats (10x10 grid)
- **Timeline**: 7 days
- **Tech Stack**: Go, WebSocket, Redis, NATS, NGINX

---

## DAY 1: PROJECT FOUNDATION

### Step 1.1: Create Project Structure
```bash
mkdir -p edge-server booking-service shared frontend nginx
```

**Expected Output**: 
- Project folder with subdirectories created

### Step 1.2: Initialize Go Module
```bash
go mod init concert-booking
```

**Expected Output**: 
- `go.mod` file created in root

### Step 1.3: Create docker-compose.yml
**File**: `docker-compose.yml`
**Purpose**: Define all infrastructure services
**Key Points**:
- Redis on port 6379 for state management
- NATS on port 4222 for pub/sub messaging
- NGINX on port 80 for load balancing

**What to implement**:
```yaml
version: '3.8'
services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    command: redis-server --appendonly yes
    volumes:
      - redis-data:/data
  
  nats:
    image: nats:latest
    ports:
      - "4222:4222"  # Client connections
      - "8222:8222"  # HTTP monitoring
    command: ["-js"]  # Enable JetStream
  
  # NGINX will be added on Day 7

volumes:
  redis-data:
```

### Step 1.4: Install Go Dependencies
```bash
go get github.com/gorilla/websocket
go get github.com/nats-io/nats.go
go get github.com/go-redis/redis/v8
go get github.com/gin-gonic/gin
go get github.com/google/uuid
```

**Verify Installation**:
```bash
go mod tidy
cat go.mod  # Should show all dependencies
```

### Step 1.5: Create Shared Models
**File**: `shared/models.go`
**Purpose**: Define data structures used across services

**What to implement**:
1. Seat struct with fields:
   - ID (string): "A1", "B2", etc.
   - Row (int): 0-9
   - Col (int): 0-9  
   - Status (int): 0=Available, 1=Held, 2=Booked
   - HeldBy (string): User ID who holds it
   - ExpiresAt (int64): Unix timestamp when hold expires

2. Message types for WebSocket:
   - ClientMessage: What browser sends
   - ServerMessage: What server sends back
   - Types: CONNECT, SELECT_SEAT, BOOK_SEAT, RELEASE_SEAT, SEAT_UPDATE, VENUE_STATE

3. Event types for NATS:
   - SeatEvent with Type, SeatID, UserID, Timestamp

**Code Structure**:
```go
package shared

// Seat statuses
const (
    SeatAvailable = 0
    SeatHeld      = 1
    SeatBooked    = 2
)

// Seat represents a single seat
type Seat struct {
    ID        string `json:"id"`
    Row       int    `json:"row"`
    Col       int    `json:"col"`
    Status    int    `json:"status"`
    HeldBy    string `json:"held_by,omitempty"`
    ExpiresAt int64  `json:"expires_at,omitempty"`
}

// Add more structures...
```

### Step 1.6: Create Constants
**File**: `shared/constants.go`
**Purpose**: Centralize configuration values

**What to implement**:
- Redis key patterns: "venue:seats", "seat:{id}:lock"
- NATS topics: "seats.held", "seats.released", "seats.booked"
- Timeouts: HoldDuration = 30 seconds
- Venue config: 10 rows, 10 cols

### Step 1.7: Create Makefile
**File**: `Makefile`
**Purpose**: Simplify running commands

**What to implement**:
```makefile
.PHONY: run-infra run-booking run-edge-1 run-edge-2 stop clean

run-infra:
	docker-compose up -d redis nats

stop-infra:
	docker-compose down

run-booking:
	go run booking-service/main.go

run-edge-1:
	PORT=3000 go run edge-server/*.go

run-edge-2:
	PORT=3001 go run edge-server/*.go

clean:
	docker-compose down -v
	rm -f go.sum
```

### Step 1.8: Verify Day 1 Setup
**Commands to run**:
```bash
# Start infrastructure
make run-infra

# Check Redis is running
docker exec -it concert-booking-prototype-redis-1 redis-cli ping
# Should return: PONG

# Check NATS is running  
curl http://localhost:8222/varz
# Should return JSON with server info

# Stop everything
make stop-infra
```

**End of Day 1 Checklist**:
- [ ] Project structure created
- [ ] Go module initialized with dependencies
- [ ] docker-compose.yml with Redis and NATS
- [ ] Shared models defined
- [ ] Constants file created
- [ ] Makefile ready
- [ ] Infrastructure starts successfully

---

## DAY 2: BOOKING SERVICE

### Step 2.1: Create Booking Service Main
**File**: `booking-service/main.go`
**Purpose**: HTTP API server for seat management

**What to implement**:
1. Initialize Gin router
2. Connect to Redis
3. Connect to NATS
4. Initialize venue (create 100 seats)
5. Setup API routes
6. Start server on port 8080

**Key Functions Needed**:
- `connectRedis()` - Create Redis client
- `connectNATS()` - Create NATS connection
- `initializeVenue()` - Create 100 seats in Redis
- `setupRoutes()` - Define API endpoints

### Step 2.2: Create Seat Manager
**File**: `booking-service/seat_manager.go`
**Purpose**: Core business logic for seat operations

**What to implement**:
1. `InitializeVenue()`:
   - Create 100 seats (A1-J10)
   - Store in Redis as hash
   - All seats start as available

2. `GetAllSeats()`:
   - Fetch all seats from Redis
   - Return as array

3. `SelectSeat(seatID, userID)`:
   - Check if seat available
   - Use Redis SET NX EX for atomic lock
   - Set status to held
   - Set 30 second TTL
   - Publish to NATS

4. `BookSeat(seatID, userID)`:
   - Verify user holds the seat
   - Change status to booked
   - Remove TTL
   - Publish to NATS

5. `ReleaseSeat(seatID, userID)`:
   - Verify user holds the seat
   - Set status back to available
   - Remove lock
   - Publish to NATS

**Redis Commands to Use**:
```redis
# Store seat data
HSET venue:seats A1 "{json data}"

# Atomic lock
SET seat:A1:lock userID NX EX 30

# Get all seats
HGETALL venue:seats

# Delete lock
DEL seat:A1:lock
```

### Step 2.3: Create API Handlers
**File**: `booking-service/handlers.go`
**Purpose**: HTTP request handlers

**What to implement**:
1. `GET /api/seats`:
   - Call GetAllSeats()
   - Return JSON array

2. `POST /api/seats/select`:
   - Parse request body: {seat_id, user_id}
   - Call SelectSeat()
   - Return success/error

3. `POST /api/seats/book`:
   - Parse request body
   - Call BookSeat()
   - Return confirmation

4. `POST /api/seats/release`:
   - Parse request body
   - Call ReleaseSeat()
   - Return success

5. `GET /health`:
   - Return {"status": "ok"}

### Step 2.4: NATS Publisher Functions
**Add to**: `booking-service/seat_manager.go`

**What to implement**:
```go
func publishSeatEvent(eventType string, seatID string, userID string) {
    // Create event struct
    // Marshal to JSON
    // Publish to NATS topic
    // Log any errors
}
```

### Step 2.5: Test Booking Service
**Commands to run**:
```bash
# Terminal 1: Start infrastructure
make run-infra

# Terminal 2: Start booking service
go run booking-service/main.go
# Should see: "Booking service started on :8080"

# Terminal 3: Test endpoints
# Get all seats
curl http://localhost:8080/api/seats

# Select a seat
curl -X POST http://localhost:8080/api/seats/select \
  -H "Content-Type: application/json" \
  -d '{"seat_id":"A1","user_id":"user123"}'

# Check Redis
docker exec -it concert-booking-prototype-redis-1 redis-cli
> HGET venue:seats A1
> TTL seat:A1:lock
```

**End of Day 2 Checklist**:
- [ ] Booking service starts on port 8080
- [ ] Can GET all 100 seats
- [ ] Can select a seat (changes to held)
- [ ] Redis shows seat data and lock
- [ ] NATS connection established

---

## DAY 3: TIMER AND EVENTS

### Step 3.1: Add Timer Manager
**File**: `booking-service/timer.go`
**Purpose**: Auto-release held seats after timeout

**What to implement**:
1. `StartTimerService()`:
   - Run in goroutine
   - Check every 2 seconds
   - Find expired holds
   - Release them

2. `checkExpiredHolds()`:
   - Get all seats from Redis
   - Check ExpiresAt field
   - If past current time, release
   - Publish release event

**Code Structure**:
```go
func StartTimerService(redisClient *redis.Client, natsConn *nats.Conn) {
    ticker := time.NewTicker(2 * time.Second)
    go func() {
        for range ticker.C {
            checkExpiredHolds(redisClient, natsConn)
        }
    }()
}
```

### Step 3.2: Improve NATS Event Publishing
**Update**: `booking-service/seat_manager.go`

**What to implement**:
1. Structured event format
2. Different topics for different events
3. Include timestamp
4. Error handling

**Event Structure**:
```go
type SeatEvent struct {
    Type      string    `json:"type"`      // held, released, booked
    SeatID    string    `json:"seat_id"`
    UserID    string    `json:"user_id"`
    Status    int       `json:"status"`
    Timestamp time.Time `json:"timestamp"`
}
```

### Step 3.3: Test Timer Functionality
**Commands to run**:
```bash
# Select a seat
curl -X POST http://localhost:8080/api/seats/select \
  -d '{"seat_id":"B2","user_id":"user456"}'

# Watch in Redis (should see TTL counting down)
docker exec -it concert-booking-prototype-redis-1 redis-cli
> TTL seat:B2:lock
# Wait 30 seconds
> HGET venue:seats B2
# Should show status back to available

# Monitor NATS events
# In another terminal:
docker exec -it concert-booking-prototype-nats-1 nats sub "seats.>"
```

**End of Day 3 Checklist**:
- [ ] Timer service runs in background
- [ ] Held seats auto-release after 30 seconds
- [ ] NATS events published for all state changes
- [ ] Can monitor events with NATS CLI

---

## DAY 4: EDGE SERVER - WEBSOCKET

### Step 4.1: Create Edge Server Main
**File**: `edge-server/main.go`
**Purpose**: WebSocket server for real-time communication

**What to implement**:
1. HTTP server with configurable port
2. WebSocket upgrade endpoint
3. Hub initialization
4. NATS subscriber
5. Graceful shutdown

**Key Components**:
- Port from environment variable
- `/ws` endpoint for WebSocket
- Health check endpoint
- Logging

### Step 4.2: Create Client Structure
**File**: `edge-server/client.go`
**Purpose**: Manage individual WebSocket connections

**What to implement**:
```go
type Client struct {
    hub      *Hub
    conn     *websocket.Conn
    send     chan []byte
    id       string
    userID   string
}
```

**Methods needed**:
1. `readPump()`:
   - Read messages from WebSocket
   - Parse JSON
   - Handle different message types
   - Forward to hub

2. `writePump()`:
   - Write messages to WebSocket
   - Handle ping/pong
   - Send from channel

3. `close()`:
   - Clean up connection
   - Unregister from hub

### Step 4.3: Create Hub Structure
**File**: `edge-server/hub.go`
**Purpose**: Manage all WebSocket clients

**What to implement**:
```go
type Hub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
}
```

**Methods needed**:
1. `run()`:
   - Main event loop
   - Handle registrations
   - Handle unregistrations
   - Broadcast messages

2. `broadcastMessage(message []byte)`:
   - Send to all clients
   - Skip failed sends

### Step 4.4: Create Message Handlers
**File**: `edge-server/handlers.go`
**Purpose**: Process incoming WebSocket messages

**What to implement**:
1. `handleMessage(client *Client, message []byte)`:
   - Parse message type
   - Route to appropriate handler

2. `handleSubscribe(client *Client, data map[string]interface{})`:
   - Set client's venue
   - Send current venue state

3. `handleSelectSeat(client *Client, data map[string]interface{})`:
   - Extract seat_id
   - Call booking service API
   - Handle response

4. `handleBookSeat(client *Client, data map[string]interface{})`:
   - Similar to select
   - Call book endpoint

### Step 4.5: Test WebSocket Connection
**Commands to run**:
```bash
# Start edge server
PORT=3000 go run edge-server/*.go

# Test with websocat (install: brew install websocat)
websocat ws://localhost:3000/ws

# Or test with browser console:
# Open browser console and run:
const ws = new WebSocket('ws://localhost:3000/ws');
ws.onopen = () => console.log('Connected');
ws.onmessage = (e) => console.log('Message:', e.data);
ws.send(JSON.stringify({type: 'subscribe', venue: 'main'}));
```

**End of Day 4 Checklist**:
- [ ] Edge server starts on port 3000
- [ ] Can connect via WebSocket
- [ ] Can send/receive messages
- [ ] Hub manages multiple clients
- [ ] Basic message handling works

---

## DAY 5: EDGE SERVER - INTEGRATION

### Step 5.1: NATS Subscriber
**Add to**: `edge-server/main.go`

**What to implement**:
1. `subscribeToNATS()`:
   - Subscribe to "seats.*" topics
   - Parse incoming events
   - Broadcast to all clients

```go
func subscribeToNATS(nc *nats.Conn, hub *Hub) {
    nc.Subscribe("seats.>", func(msg *nats.Msg) {
        // Parse event
        // Convert to WebSocket message
        // Broadcast via hub
    })
}
```

### Step 5.2: Booking Service Client
**File**: `edge-server/booking_client.go`
**Purpose**: Call booking service APIs

**What to implement**:
1. HTTP client with timeout
2. Functions for each API endpoint:
   - `selectSeat(seatID, userID)`
   - `bookSeat(seatID, userID)`
   - `releaseSeat(seatID, userID)`
   - `getVenueState()`

### Step 5.3: Connect Everything
**Update**: `edge-server/handlers.go`

**What to implement**:
1. When client selects seat:
   - Call booking service
   - Don't wait for NATS (it will broadcast)

2. When NATS event received:
   - Create SEAT_UPDATE message
   - Broadcast to all clients

3. On new connection:
   - Send VENUE_STATE with all seats

### Step 5.4: Message Format
**Define clear message format**:

**Client to Server**:
```json
{
  "type": "SELECT_SEAT",
  "data": {
    "seat_id": "A1",
    "user_id": "user123"
  }
}
```

**Server to Client**:
```json
{
  "type": "SEAT_UPDATE",
  "data": {
    "seat_id": "A1",
    "status": 1,
    "held_by": "user123",
    "expires_at": 1699564920
  }
}
```

### Step 5.5: Test Full Flow
**Commands to run**:
```bash
# Terminal 1: Infrastructure
make run-infra

# Terminal 2: Booking service
make run-booking

# Terminal 3: Edge server 1
PORT=3000 go run edge-server/*.go

# Terminal 4: Edge server 2
PORT=3001 go run edge-server/*.go

# Terminal 5: Test WebSocket
# Connect to edge 1
websocat ws://localhost:3000/ws
> {"type":"subscribe","data":{"venue":"main"}}
> {"type":"SELECT_SEAT","data":{"seat_id":"C3","user_id":"test1"}}

# Terminal 6: Connect to edge 2
websocat ws://localhost:3001/ws
> {"type":"subscribe","data":{"venue":"main"}}
# Should see seat update from edge 1's action!
```

**End of Day 5 Checklist**:
- [ ] Edge server connects to NATS
- [ ] Edge server calls booking service
- [ ] NATS events trigger broadcasts
- [ ] Multiple edge servers share updates
- [ ] Full flow works: Select → API → NATS → Broadcast

---

## DAY 6: FRONTEND

### Step 6.1: Create HTML Structure
**File**: `frontend/index.html`
**Purpose**: Main UI structure

**What to implement**:
```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Concert Seat Booking</title>
    <link rel="stylesheet" href="styles.css">
</head>
<body>
    <div class="container">
        <header>
            <h1>Concert Seat Booking</h1>
            <div class="status-bar">
                <span id="connection-status" class="disconnected">● Disconnected</span>
                <span id="user-info">User: <span id="user-id"></span></span>
                <span id="available-count">Available: <span id="count">100</span>/100</span>
            </div>
        </header>
        
        <main>
            <div class="stage">STAGE</div>
            <div id="seat-map" class="seat-map">
                <!-- Seats will be generated here -->
            </div>
            
            <div class="legend">
                <div><span class="seat available"></span> Available</div>
                <div><span class="seat held"></span> Held (30s)</div>
                <div><span class="seat booked"></span> Booked</div>
                <div><span class="seat mine"></span> Your Selection</div>
            </div>
        </main>
        
        <aside class="actions">
            <div id="selected-info" style="display:none;">
                <h3>Selected Seat: <span id="selected-seat"></span></h3>
                <button id="book-btn">Book Seat</button>
                <button id="release-btn">Release</button>
                <div id="timer"></div>
            </div>
            <div id="messages"></div>
        </aside>
    </div>
    
    <script src="app.js"></script>
</body>
</html>
```

### Step 6.2: Create CSS Styles
**File**: `frontend/styles.css`
**Purpose**: Visual styling

**What to implement**:
1. Grid layout for seats (10x10)
2. Seat colors:
   - Green: available
   - Yellow: held
   - Red: booked
   - Blue border: your selection
3. Hover effects
4. Responsive design
5. Status indicators

**Key Styles**:
```css
.seat-map {
    display: grid;
    grid-template-columns: repeat(10, 40px);
    grid-template-rows: repeat(10, 40px);
    gap: 5px;
}

.seat {
    width: 40px;
    height: 40px;
    border-radius: 4px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 10px;
    transition: all 0.3s;
}

.seat.available { background: #27ae60; }
.seat.held { background: #f39c12; }
.seat.booked { background: #e74c3c; cursor: not-allowed; }
.seat.mine { border: 3px solid #3498db; }
```

### Step 6.3: Create JavaScript Client
**File**: `frontend/app.js`
**Purpose**: WebSocket client and UI logic

**What to implement**:
```javascript
class SeatBookingApp {
    constructor() {
        this.ws = null;
        this.seats = {};
        this.selectedSeat = null;
        this.userId = this.generateUserId();
        this.wsUrl = 'ws://localhost:3000/ws';
        this.reconnectAttempts = 0;
    }
    
    init() {
        this.renderSeatMap();
        this.connect();
        this.setupEventListeners();
    }
    
    connect() {
        // Create WebSocket connection
        // Setup event handlers
        // Handle reconnection
    }
    
    renderSeatMap() {
        // Create 100 seat elements
        // Add click handlers
    }
    
    handleSeatClick(seatId) {
        // Send SELECT_SEAT message
    }
    
    updateSeat(seatData) {
        // Update UI for single seat
    }
    
    handleMessage(event) {
        // Parse and route messages
    }
}
```

### Step 6.4: Implement Core Functions
**Complete implementation needed**:

1. `generateUserId()`:
   - Create random user ID
   - Store in localStorage

2. `connect()`:
   - Create WebSocket
   - Auto-reconnect on disconnect
   - Update connection status

3. `renderSeatMap()`:
   - Create 100 divs
   - Set IDs like "seat-A1"
   - Add to DOM

4. `handleSeatClick()`:
   - Check if available
   - Send SELECT_SEAT
   - Update UI optimistically

5. `handleMessage()`:
   - Parse JSON
   - Handle VENUE_STATE
   - Handle SEAT_UPDATE
   - Handle ERROR

6. `updateSeat()`:
   - Find seat element
   - Update class
   - Update counters

### Step 6.5: Add Timer Display
**Enhancement**: Show countdown for held seats

**What to implement**:
```javascript
startTimer(seatId, expiresAt) {
    const timerInterval = setInterval(() => {
        const remaining = expiresAt - Date.now()/1000;
        if (remaining <= 0) {
            clearInterval(timerInterval);
            // Seat expired
        } else {
            // Update timer display
        }
    }, 1000);
}
```

### Step 6.6: Test Frontend
**Steps**:
1. Open index.html in browser
2. Check seat grid renders
3. Open console for errors
4. Click a seat
5. Check WebSocket messages in Network tab
6. Open second browser window
7. Select seat in one, see update in other

**End of Day 6 Checklist**:
- [ ] Seat grid displays 10x10
- [ ] Can click seats
- [ ] WebSocket connects
- [ ] Seats change colors
- [ ] Multiple browsers see updates
- [ ] Timer shows for held seats

---

## DAY 7: INTEGRATION AND DEPLOYMENT

### Step 7.1: NGINX Configuration
**File**: `nginx/nginx.conf`
**Purpose**: Load balance WebSocket connections

**What to implement**:
```nginx
upstream edge_servers {
    ip_hash;  # Sticky sessions
    server host.docker.internal:3000;
    server host.docker.internal:3001;
}

upstream booking_api {
    server host.docker.internal:8080;
}

server {
    listen 80;
    
    # WebSocket endpoint
    location /ws {
        proxy_pass http://edge_servers;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }
    
    # Booking API
    location /api {
        proxy_pass http://booking_api;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
    
    # Frontend files
    location / {
        root /usr/share/nginx/html;
        index index.html;
        try_files $uri $uri/ /index.html;
    }
}
```

### Step 7.2: Update Docker Compose
**File**: `docker-compose.yml`
**Purpose**: Add all services

**What to add**:
```yaml
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./frontend:/usr/share/nginx/html:ro
    depends_on:
      - redis
      - nats
    extra_hosts:
      - "host.docker.internal:host-gateway"
```

### Step 7.3: Update Frontend WebSocket URL
**File**: `frontend/app.js`
**Change**: WebSocket URL to use NGINX

```javascript
// Change from:
this.wsUrl = 'ws://localhost:3000/ws';
// To:
this.wsUrl = 'ws://localhost/ws';
```

### Step 7.4: Create Start Scripts
**File**: `start.sh`
**Purpose**: Start everything in correct order

```bash
#!/bin/bash
echo "Starting infrastructure..."
docker-compose up -d redis nats

echo "Waiting for Redis and NATS..."
sleep 3

echo "Starting booking service..."
go run booking-service/main.go &
BOOKING_PID=$!

echo "Starting edge servers..."
PORT=3000 go run edge-server/*.go &
EDGE1_PID=$!

PORT=3001 go run edge-server/*.go &
EDGE2_PID=$!

echo "Starting NGINX..."
docker-compose up -d nginx

echo "All services started!"
echo "Open http://localhost in your browser"
echo "Press Ctrl+C to stop all services"

# Wait for interrupt
trap "echo 'Stopping...'; kill $BOOKING_PID $EDGE1_PID $EDGE2_PID; docker-compose down; exit" INT
wait
```

### Step 7.5: Full System Test
**Test Scenario**:
1. Run `./start.sh`
2. Open http://localhost in Browser 1
3. Open http://localhost in Browser 2 (incognito)
4. In Browser 1: Click seat "D5"
5. Verify Browser 2 shows D5 as yellow
6. Wait 30 seconds
7. Verify both browsers show D5 as green again
8. In Browser 1: Click D5, then click "Book"
9. Verify Browser 2 shows D5 as red

### Step 7.6: Debugging Checklist
**If things don't work**:

**WebSocket won't connect**:
- Check browser console
- Check NGINX is running: `docker ps`
- Check edge servers: `ps aux | grep edge-server`
- Try direct connection: `ws://localhost:3000/ws`

**Seats not updating**:
- Check NATS is running: `docker logs concert-booking-prototype-nats-1`
- Check booking service logs
- Monitor NATS: `docker exec -it concert-booking-prototype-nats-1 nats sub "seats.>"`

**Seats not auto-releasing**:
- Check timer in booking service
- Check Redis TTL: `redis-cli TTL seat:D5:lock`
- Check system time sync

### Step 7.7: Create Demo Script
**File**: `DEMO.md`
**Purpose**: Script for demonstrating the system

```markdown
# Demo Script

## Setup
1. Start all services: `./start.sh`
2. Open 2 browser windows side by side
3. Both go to http://localhost

## Demo Flow

### 1. Real-time Updates
- User 1: Click seat B3
- User 2: See B3 turn yellow immediately
- Explain: "WebSocket broadcast via NATS"

### 2. Auto-release
- Wait 30 seconds
- Both: See B3 turn green
- Explain: "Timer service with Redis TTL"

### 3. Booking
- User 1: Click C4, click "Book"
- User 2: See C4 turn red
- Explain: "Permanent booking"

### 4. Conflict Prevention
- User 1: Click E5
- User 2: Try to click E5
- User 2: Gets error
- Explain: "Redis atomic locks"

### 5. Scale Demo
- Show 2 edge servers running
- Kill one: `kill <PID>`
- Still works (via other edge)
- Explain: "Horizontal scaling"
```

### Step 7.8: Final Verification
**Complete System Checklist**:
- [ ] All services start with one command
- [ ] Frontend loads at http://localhost
- [ ] Can select seats (turn yellow)
- [ ] Other browsers see updates immediately
- [ ] Seats auto-release after 30 seconds
- [ ] Can book seats (turn red)
- [ ] No double-booking possible
- [ ] System survives edge server restart
- [ ] 10+ users can use simultaneously

**End of Day 7 Checklist**:
- [ ] NGINX load balancing works
- [ ] Full system runs via docker-compose
- [ ] Demo script ready
- [ ] All core features working
- [ ] Ready for demonstration!

---

## TROUBLESHOOTING GUIDE

### Issue: WebSocket Connection Fails
**Symptoms**: "WebSocket connection failed" in browser console
**Solutions**:
1. Check edge servers are running: `ps aux | grep edge-server`
2. Check NGINX is running: `docker ps | grep nginx`
3. Try direct connection: `ws://localhost:3000/ws`
4. Check NGINX logs: `docker logs concert-booking-prototype-nginx-1`

### Issue: Seats Not Updating
**Symptoms**: Click seat in one browser, doesn't update in another
**Solutions**:
1. Check NATS connection in edge server logs
2. Monitor NATS: `docker exec -it concert-booking-prototype-nats-1 nats sub "seats.>"`
3. Check booking service is publishing events
4. Verify WebSocket message format matches

### Issue: Seats Not Auto-Releasing
**Symptoms**: Yellow seats stay yellow after 30 seconds
**Solutions**:
1. Check timer goroutine is running (add log)
2. Check Redis TTL: `redis-cli TTL seat:A1:lock`
3. Check ExpiresAt timestamp is correct
4. Verify system time is synchronized

### Issue: Double Booking Occurs
**Symptoms**: Two users can book same seat
**Solutions**:
1. Check Redis SET NX is used (atomic operation)
2. Verify lock checking before booking
3. Add logging to see race condition
4. Check Redis connection pool settings

### Issue: High Latency
**Symptoms**: Updates take >1 second
**Solutions**:
1. Check network latency: `ping localhost`
2. Monitor Redis: `redis-cli --latency`
3. Check NATS performance: `nats bench seats.test`
4. Reduce logging in production

---

## QUICK REFERENCE

### Start Everything
```bash
# Method 1: Using script
./start.sh

# Method 2: Manual
docker-compose up -d redis nats
go run booking-service/main.go &
PORT=3000 go run edge-server/*.go &
PORT=3001 go run edge-server/*.go &
docker-compose up -d nginx
```

### Test Commands
```bash
# Test booking API
curl http://localhost:8080/api/seats

# Test WebSocket
websocat ws://localhost/ws

# Monitor NATS
docker exec -it concert-booking-prototype-nats-1 nats sub "seats.>"

# Check Redis
docker exec -it concert-booking-prototype-redis-1 redis-cli
> HGETALL venue:seats
> KEYS seat:*:lock
```

### Stop Everything
```bash
# Stop all services
docker-compose down
pkill -f "go run"

# Clean everything
docker-compose down -v
rm -rf redis-data/
```

---

## SUCCESS CRITERIA

Your prototype is complete when:
1. ✅ Multiple users can view the same 10x10 seat grid
2. ✅ Clicking a seat updates all browsers in <200ms
3. ✅ Held seats auto-release after 30 seconds
4. ✅ Booked seats are permanent
5. ✅ No double-booking is possible
6. ✅ System works with 10+ concurrent users
7. ✅ Can start everything with one command
8. ✅ Can demo the core features

---

## NEXT STEPS (After Prototype)

Once prototype is working:
1. Add authentication (JWT tokens)
2. Add payment integration
3. Add database persistence
4. Implement proper error handling
5. Add monitoring and metrics
6. Deploy to cloud (AWS/GCP/DigitalOcean)
7. Scale to support 1000+ users
8. Add more features (seat types, pricing, etc.)

---

## NOTES SECTION

Use this space to track your progress and issues:

### Day 1 Notes:
- 

### Day 2 Notes:
- 

### Day 3 Notes:
- 

### Day 4 Notes:
- 

### Day 5 Notes:
- 

### Day 6 Notes:
- 

### Day 7 Notes:
- 

### Bugs Found:
- 

### Ideas for Improvement:
- 

---

END OF PLAN.md