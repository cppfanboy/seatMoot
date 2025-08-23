class SeatBookingApp {
    constructor() {
        this.ws = null;
        this.seats = {};
        this.selectedSeat = null;
        this.userId = this.generateUserId();
        this.wsUrl = 'ws://localhost:3000/ws';
        this.reconnectAttempts = 0;
        this.reconnectDelay = 1000;
        this.maxReconnectAttempts = 10;
        this.timers = {};
    }
    
    init() {
        this.updateUserDisplay();
        this.renderSeatMap();
        this.connect();
        this.setupEventListeners();
    }
    
    generateUserId() {
        // Check if user ID exists in localStorage
        let userId = localStorage.getItem('userId');
        if (!userId) {
            // Generate a new user ID
            userId = 'user_' + Math.random().toString(36).substr(2, 9);
            localStorage.setItem('userId', userId);
        }
        return userId;
    }
    
    updateUserDisplay() {
        document.getElementById('user-id').textContent = this.userId;
    }
    
    connect() {
        this.showMessage('Connecting to server...', 'info');
        
        this.ws = new WebSocket(this.wsUrl);
        
        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.reconnectAttempts = 0;
            this.updateConnectionStatus(true);
            
            // Subscribe with user ID
            this.send({
                type: 'SUBSCRIBE',
                data: { user_id: this.userId }
            });
        };
        
        this.ws.onmessage = (event) => {
            this.handleMessage(event);
        };
        
        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            this.showMessage('Connection error occurred', 'error');
        };
        
        this.ws.onclose = () => {
            console.log('WebSocket disconnected');
            this.updateConnectionStatus(false);
            this.reconnect();
        };
    }
    
    reconnect() {
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            this.showMessage('Failed to connect after multiple attempts', 'error');
            return;
        }
        
        this.reconnectAttempts++;
        const delay = Math.min(this.reconnectDelay * this.reconnectAttempts, 10000);
        
        this.showMessage(`Reconnecting in ${delay/1000}s... (attempt ${this.reconnectAttempts})`, 'info');
        
        setTimeout(() => {
            this.connect();
        }, delay);
    }
    
    send(message) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(message));
        } else {
            console.error('WebSocket not connected');
            this.showMessage('Cannot send message - not connected', 'error');
        }
    }
    
    renderSeatMap() {
        const seatMap = document.getElementById('seat-map');
        seatMap.innerHTML = '';
        
        // Create 10x10 grid (A1-J10)
        for (let row = 0; row < 10; row++) {
            for (let col = 0; col < 10; col++) {
                const rowLetter = String.fromCharCode(65 + row); // A-J
                const seatId = `${rowLetter}${col + 1}`;
                
                const seatElement = document.createElement('div');
                seatElement.className = 'seat available';
                seatElement.dataset.seatId = seatId;
                seatElement.textContent = seatId;
                
                seatElement.addEventListener('click', () => {
                    this.handleSeatClick(seatId);
                });
                
                seatMap.appendChild(seatElement);
                
                // Store reference
                this.seats[seatId] = {
                    element: seatElement,
                    status: 0,
                    heldBy: null,
                    expiresAt: null
                };
            }
        }
    }
    
    handleSeatClick(seatId) {
        const seat = this.seats[seatId];
        
        // Check if seat is available or held by current user
        if (seat.status === 2) {
            this.showMessage(`Seat ${seatId} is already booked`, 'error');
            return;
        }
        
        if (seat.status === 1 && seat.heldBy !== this.userId) {
            this.showMessage(`Seat ${seatId} is held by another user`, 'error');
            return;
        }
        
        // If clicking the same seat that's already selected
        if (this.selectedSeat === seatId && seat.heldBy === this.userId) {
            this.showSelectedSeatInfo(seatId);
            return;
        }
        
        // Select the seat
        this.send({
            type: 'SELECT_SEAT',
            data: {
                seat_id: seatId,
                user_id: this.userId
            }
        });
    }
    
    handleMessage(event) {
        try {
            const message = JSON.parse(event.data);
            console.log('Received message:', message);
            
            switch (message.type) {
                case 'WELCOME':
                    this.handleWelcome(message.data);
                    break;
                    
                case 'SUBSCRIBE_ACK':
                    this.showMessage('Connected and subscribed successfully', 'success');
                    break;
                    
                case 'VENUE_STATE':
                    this.handleVenueState(message.data);
                    break;
                    
                case 'SEAT_UPDATE':
                    this.handleSeatUpdate(message.data);
                    break;
                    
                case 'SELECT_SEAT_RESPONSE':
                    this.handleSelectResponse(message.data);
                    break;
                    
                case 'BOOK_SEAT_RESPONSE':
                    this.handleBookResponse(message.data);
                    break;
                    
                case 'RELEASE_SEAT_RESPONSE':
                    this.handleReleaseResponse(message.data);
                    break;
                    
                case 'ERROR':
                    this.showMessage(message.data.error, 'error');
                    break;
                    
                default:
                    console.log('Unknown message type:', message.type);
            }
        } catch (error) {
            console.error('Error handling message:', error);
        }
    }
    
    handleWelcome(data) {
        console.log('Welcome message received:', data);
        this.showMessage(`Connected as ${data.client_id}`, 'success');
    }
    
    handleVenueState(data) {
        // Update all seats with current state
        data.seats.forEach(seatData => {
            this.updateSeat(seatData);
        });
        this.updateAvailableCount();
    }
    
    handleSeatUpdate(data) {
        // Real-time seat update from NATS
        const seat = data.seat;
        if (seat) {
            this.updateSeat(seat);
            
            // Show notification
            if (data.event_type === 'held' && data.user_id !== this.userId) {
                this.showMessage(`Seat ${data.seat_id} was selected by another user`, 'info');
            } else if (data.event_type === 'booked' && data.user_id !== this.userId) {
                this.showMessage(`Seat ${data.seat_id} was booked`, 'info');
            } else if (data.event_type === 'released' || data.event_type === 'auto_released') {
                this.showMessage(`Seat ${data.seat_id} is now available`, 'success');
            }
        }
        this.updateAvailableCount();
    }
    
    handleSelectResponse(data) {
        if (data.success) {
            const seatId = data.data.seat_id;
            this.selectedSeat = seatId;
            this.showMessage(data.message, 'success');
            this.showSelectedSeatInfo(seatId);
            
            // Start timer for 30 seconds
            this.startTimer(seatId, 30);
        } else {
            this.showMessage(data.message, 'error');
        }
    }
    
    handleBookResponse(data) {
        if (data.success) {
            this.showMessage(data.message, 'success');
            this.hideSelectedSeatInfo();
            this.selectedSeat = null;
            this.stopTimer(data.data.seat_id);
        } else {
            this.showMessage(data.message, 'error');
        }
    }
    
    handleReleaseResponse(data) {
        if (data.success) {
            this.showMessage(data.message, 'success');
            this.hideSelectedSeatInfo();
            this.selectedSeat = null;
            this.stopTimer(data.data.seat_id);
        } else {
            this.showMessage(data.message, 'error');
        }
    }
    
    updateSeat(seatData) {
        const seat = this.seats[seatData.id];
        if (!seat) return;
        
        // Update internal state
        seat.status = seatData.status;
        seat.heldBy = seatData.held_by || null;
        seat.expiresAt = seatData.expires_at || null;
        
        // Update visual state
        seat.element.className = 'seat';
        
        switch (seatData.status) {
            case 0: // Available
                seat.element.classList.add('available');
                break;
            case 1: // Held
                seat.element.classList.add('held');
                if (seatData.held_by === this.userId) {
                    seat.element.classList.add('mine');
                }
                break;
            case 2: // Booked
                seat.element.classList.add('booked');
                break;
        }
    }
    
    updateAvailableCount() {
        const availableCount = Object.values(this.seats).filter(s => s.status === 0).length;
        document.getElementById('count').textContent = availableCount;
    }
    
    showSelectedSeatInfo(seatId) {
        const selectedInfo = document.getElementById('selected-info');
        document.getElementById('selected-seat').textContent = seatId;
        selectedInfo.style.display = 'block';
    }
    
    hideSelectedSeatInfo() {
        document.getElementById('selected-info').style.display = 'none';
        document.getElementById('timer').textContent = '';
    }
    
    startTimer(seatId, seconds) {
        this.stopTimer(seatId);
        
        const timerElement = document.getElementById('timer');
        let remaining = seconds;
        
        const updateTimer = () => {
            if (remaining > 0) {
                timerElement.textContent = `Time remaining: ${remaining}s`;
                remaining--;
            } else {
                timerElement.textContent = 'Hold expired';
                this.stopTimer(seatId);
                this.hideSelectedSeatInfo();
                this.selectedSeat = null;
            }
        };
        
        updateTimer();
        this.timers[seatId] = setInterval(updateTimer, 1000);
    }
    
    stopTimer(seatId) {
        if (this.timers[seatId]) {
            clearInterval(this.timers[seatId]);
            delete this.timers[seatId];
        }
    }
    
    updateConnectionStatus(connected) {
        const statusElement = document.getElementById('connection-status');
        if (connected) {
            statusElement.textContent = '● Connected';
            statusElement.className = 'connected';
        } else {
            statusElement.textContent = '● Disconnected';
            statusElement.className = 'disconnected';
        }
    }
    
    showMessage(text, type = 'info') {
        const messagesContainer = document.getElementById('messages');
        const messageElement = document.createElement('div');
        messageElement.className = `message ${type}`;
        messageElement.textContent = text;
        
        messagesContainer.appendChild(messageElement);
        messagesContainer.scrollTop = messagesContainer.scrollHeight;
        
        // Auto-remove after 5 seconds
        setTimeout(() => {
            messageElement.remove();
        }, 5000);
    }
    
    setupEventListeners() {
        // Book button
        document.getElementById('book-btn').addEventListener('click', () => {
            if (this.selectedSeat) {
                this.send({
                    type: 'BOOK_SEAT',
                    data: {
                        seat_id: this.selectedSeat,
                        user_id: this.userId
                    }
                });
            }
        });
        
        // Release button
        document.getElementById('release-btn').addEventListener('click', () => {
            if (this.selectedSeat) {
                this.send({
                    type: 'RELEASE_SEAT',
                    data: {
                        seat_id: this.selectedSeat,
                        user_id: this.userId
                    }
                });
            }
        });
    }
}

// Initialize app when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    const app = new SeatBookingApp();
    app.init();
    
    // Make app globally accessible for debugging
    window.seatBookingApp = app;
});