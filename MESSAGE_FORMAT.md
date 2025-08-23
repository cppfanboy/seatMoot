# WebSocket Message Format Documentation

## Client to Server Messages

All client messages follow this structure:
```json
{
  "type": "MESSAGE_TYPE",
  "data": {
    // Message-specific data
  }
}
```

### 1. SUBSCRIBE
Establishes user identity for the WebSocket connection.

```json
{
  "type": "SUBSCRIBE",
  "data": {
    "user_id": "user123"
  }
}
```

**Response:**
```json
{
  "type": "SUBSCRIBE_ACK",
  "data": {
    "success": true,
    "message": "Subscribed successfully",
    "data": {
      "client_id": "client-abc123",
      "user_id": "user123"
    }
  }
}
```

### 2. SELECT_SEAT
Attempts to select (hold) a seat for 30 seconds.

```json
{
  "type": "SELECT_SEAT",
  "data": {
    "seat_id": "A1",
    "user_id": "user123"
  }
}
```

**Response:**
```json
{
  "type": "SELECT_SEAT_RESPONSE",
  "data": {
    "success": true,
    "message": "Seat A1 selected successfully",
    "data": {
      "seat_id": "A1",
      "user_id": "user123"
    }
  }
}
```

### 3. BOOK_SEAT
Permanently books a held seat.

```json
{
  "type": "BOOK_SEAT",
  "data": {
    "seat_id": "A1",
    "user_id": "user123"
  }
}
```

**Response:**
```json
{
  "type": "BOOK_SEAT_RESPONSE",
  "data": {
    "success": true,
    "message": "Seat A1 booked successfully",
    "data": {
      "seat_id": "A1",
      "user_id": "user123"
    }
  }
}
```

### 4. RELEASE_SEAT
Manually releases a held seat.

```json
{
  "type": "RELEASE_SEAT",
  "data": {
    "seat_id": "A1",
    "user_id": "user123"
  }
}
```

**Response:**
```json
{
  "type": "RELEASE_SEAT_RESPONSE",
  "data": {
    "success": true,
    "message": "Seat A1 released successfully",
    "data": {
      "seat_id": "A1",
      "user_id": "user123"
    }
  }
}
```

## Server to Client Messages

### 1. WELCOME
Sent immediately upon connection.

```json
{
  "type": "WELCOME",
  "data": {
    "client_id": "client-abc123",
    "total_clients": 5,
    "server_time": 1699123456
  }
}
```

### 2. VENUE_STATE
Complete venue state sent after subscription.

```json
{
  "type": "VENUE_STATE",
  "data": {
    "seats": [
      {
        "id": "A1",
        "row": 0,
        "col": 0,
        "status": 0,  // 0=available, 1=held, 2=booked
        "held_by": "",
        "expires_at": 0
      },
      // ... more seats
    ]
  }
}
```

### 3. SEAT_UPDATE
Real-time seat status changes broadcast to all clients.

```json
{
  "type": "SEAT_UPDATE",
  "data": {
    "event_type": "held",  // held, released, booked, auto_released
    "seat_id": "A1",
    "user_id": "user123",
    "status": 1,
    "timestamp": "2024-01-01T12:00:00Z",
    "expires_at": 1699123486,
    "seat": {
      "id": "A1",
      "row": 0,
      "col": 0,
      "status": 1,
      "held_by": "user123",
      "expires_at": 1699123486
    }
  }
}
```

### 4. ERROR
Error messages for failed operations.

```json
{
  "type": "ERROR",
  "data": {
    "error": "Seat already booked"
  }
}
```

## Seat Status Codes

- `0` - Available: Seat is free and can be selected
- `1` - Held: Seat is temporarily held (30 seconds)
- `2` - Booked: Seat is permanently booked

## Event Types for SEAT_UPDATE

- `held` - Seat was selected by a user
- `released` - Seat was manually released by a user
- `booked` - Seat was permanently booked
- `auto_released` - Seat was automatically released after hold expiry

## NATS Event Structure

Internal events published to NATS for inter-service communication:

```json
{
  "type": "held",
  "seat_id": "A1",
  "user_id": "user123",
  "status": 1,
  "timestamp": "2024-01-01T12:00:00Z",
  "expires_at": 1699123486,
  "seat": {
    "id": "A1",
    "row": 0,
    "col": 0,
    "status": 1,
    "held_by": "user123",
    "expires_at": 1699123486
  }
}
```

NATS Topics:
- `seats.held` - Seat selection events
- `seats.released` - Seat release events
- `seats.booked` - Seat booking events
- `seats.>` - Wildcard subscription for all seat events