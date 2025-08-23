# 🎭 Real-Time Concert Seat Booking System

A scalable, real-time seat booking system built with Go, WebSockets, Redis, and NATS. Features instant seat updates across all connected browsers, automatic seat release, and load-balanced architecture.

## 🌟 Features

- **Real-time Updates**: Instant seat status synchronization across all browsers
- **Atomic Operations**: Redis-based locking prevents double-booking
- **Auto-release**: Abandoned seats automatically released after 30 seconds
- **Load Balancing**: NGINX distributes WebSocket connections across multiple edge servers
- **Scalable Architecture**: Horizontally scalable with separate API and WebSocket layers
- **Beautiful UI**: Modern, responsive interface with smooth animations

## 🏗️ Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Browser 1  │     │   Browser 2  │     │   Browser N  │
└──────┬───────┘     └──────┬───────┘     └──────┬───────┘
       │                    │                     │
       └────────────────────┼─────────────────────┘
                           │
                    ┌──────▼───────┐
                    │    NGINX     │
                    │  Port 80     │
                    └──────┬───────┘
                           │
          ┌────────────────┼────────────────┐
          │                                 │
    ┌─────▼──────┐                 ┌───────▼──────┐
    │Edge Server 1│                 │Edge Server 2 │
    │  Port 3000  │                 │  Port 3001   │
    └─────┬──────┘                 └───────┬──────┘
          │                                 │
          └────────────┬───────────────────┘
                       │
              ┌────────▼────────┐
              │ Booking Service │
              │   Port 8080     │
              └────────┬────────┘
                       │
          ┌────────────┼────────────┐
          │                         │
    ┌─────▼──────┐          ┌──────▼──────┐
    │   Redis    │          │    NATS     │
    │ Port 6379  │          │  Port 4222  │
    └────────────┘          └─────────────┘
```

## 🚀 Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.21+
- Modern web browser

### 1. Clone the repository
```bash
git clone <repository-url>
cd seatBookingSystem
```

### 2. Start all services
```bash
./start-all.sh
```

This will start:
- Redis (state management)
- NATS (pub/sub messaging)
- NGINX (load balancer)
- Booking Service (business logic)
- 2 Edge Servers (WebSocket handlers)

### 3. Access the application
Open http://localhost in your browser

### 4. Stop all services
```bash
./stop-all.sh
```

## 🧪 Testing

### Run all tests
```bash
# Integration tests
./test_integration.sh

# End-to-end booking flow
./test_e2e.sh

# Real-time messaging
./test_realtime.sh

# NGINX load balancing
./test_nginx.sh

# Frontend multi-browser test
./test_frontend.sh
```

### Manual Testing
1. Open http://localhost in multiple browser windows
2. Select a seat in one browser
3. Watch it update instantly in all browsers
4. Book the seat and see it turn red everywhere
5. Wait 30 seconds without booking to see auto-release

## 📁 Project Structure

```
seatBookingSystem/
├── booking-service/      # Go REST API service
│   ├── main.go          # Service entry point
│   ├── seat_manager.go  # Core business logic
│   ├── timer.go         # Auto-release timer
│   ├── handlers.go      # HTTP handlers
│   └── Dockerfile       # Container definition
├── edge-server/         # WebSocket server
│   ├── main.go         # Server entry point
│   ├── hub.go          # Client management
│   ├── client.go       # WebSocket client handler
│   ├── handlers.go     # Message handlers
│   ├── booking_client.go # API client
│   └── Dockerfile      # Container definition
├── frontend/           # Web interface
│   ├── index.html     # HTML structure
│   ├── styles.css     # Styling
│   └── app.js         # WebSocket client
├── nginx/             # Load balancer config
│   └── nginx.conf     # NGINX configuration
├── shared/            # Shared Go packages
│   ├── models.go      # Data structures
│   └── constants.go   # Constants
└── docker-compose.yml # Container orchestration
```

## 🔧 Configuration

### Environment Variables

**Edge Server:**
- `PORT`: Server port (default: 3000)
- `BOOKING_SERVICE_URL`: Booking API URL (default: http://localhost:8080)

**Booking Service:**
- `REDIS_URL`: Redis connection (default: localhost:6379)
- `NATS_URL`: NATS connection (default: nats://localhost:4222)

### Scaling

To add more edge servers:
1. Start additional instances on different ports
2. Add them to NGINX upstream in `nginx/nginx.conf`
3. Restart NGINX

## 📊 API Endpoints

### REST API (Port 8080)
- `GET /api/seats` - Get all seats
- `POST /api/seats/select` - Select a seat
- `POST /api/seats/book` - Book a seat
- `POST /api/seats/release` - Release a seat
- `GET /health` - Health check

### WebSocket (Port 3000/3001)
- `/ws` - WebSocket connection endpoint

### NGINX (Port 80)
- `/` - Frontend files
- `/api/*` - Proxied to booking service
- `/ws` - Load-balanced WebSocket
- `/stats` - Edge server statistics
- `/nginx-health` - NGINX health check

## 📝 Message Format

### Client → Server
```json
{
  "type": "SELECT_SEAT",
  "data": {
    "seat_id": "A1",
    "user_id": "user123"
  }
}
```

### Server → Client
```json
{
  "type": "SEAT_UPDATE",
  "data": {
    "event_type": "held",
    "seat_id": "A1",
    "user_id": "user123",
    "status": 1,
    "timestamp": "2024-01-01T12:00:00Z",
    "expires_at": 1699123486
  }
}
```

## 🐳 Docker Deployment

### Development
```bash
docker-compose up -d
```

### Production
```bash
docker-compose -f docker-compose.prod.yml up -d
```

## 🔍 Monitoring

### View logs
```bash
# Booking service
tail -f logs/booking-service.log

# Edge servers
tail -f logs/edge-server-3000.log
tail -f logs/edge-server-3001.log

# Docker logs
docker-compose logs -f
```

### Check statistics
```bash
# Edge server stats
curl http://localhost/stats | jq

# Redis monitoring
redis-cli monitor

# NATS monitoring
curl http://localhost:8222/varz
```

## 🚦 Health Checks

```bash
# All health endpoints
curl http://localhost/health         # Booking service
curl http://localhost/nginx-health   # NGINX
curl http://localhost/stats          # Edge servers
```

## 🔐 Security Considerations

- No authentication implemented (demo purposes)
- Use HTTPS in production
- Implement rate limiting
- Add input validation
- Use secure WebSocket (WSS)
- Implement CORS properly

## 📈 Performance

- Supports hundreds of concurrent users
- Sub-50ms message latency (local)
- Horizontal scaling via edge servers
- Redis handles 100k+ ops/sec
- NATS handles millions of messages/sec

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Open a pull request

## 📄 License

MIT License - feel free to use this project for learning and development.

## 🙏 Acknowledgments

- Built with Go, Redis, NATS, and NGINX
- WebSocket handling via Gorilla WebSocket
- UI inspired by modern booking systems

## 📞 Support

For issues and questions:
- Open an issue on GitHub
- Check existing documentation
- Review test scripts for examples

---

Built with ❤️ for real-time applications