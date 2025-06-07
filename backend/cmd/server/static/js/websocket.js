// WebSocket connection management
class GameWebSocket {
    constructor() {
        this.ws = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 1000;
        this.messageHandlers = new Map();
        this.connectionStatus = 'disconnected';
        
        this.setupEventHandlers();
        this.connect();
    }
    
    setupEventHandlers() {
        // Register message handlers
        this.onMessage('game_state', (data) => {
            if (window.canvasGame) {
                window.canvasGame.updateGameState(data);
            } else {
                // Cache the game state until canvas game is ready
                this.cachedGameState = data;
                console.log('Cached game state for later use');
            }
        });
        
        this.onMessage('leaderboard', (data) => {
            if (window.leaderboard) {
                window.leaderboard.updateLeaderboard(data);
            }
        });
        
        this.onMessage('leaderboard_update', (data) => {
            if (window.leaderboard) {
                window.leaderboard.updateLeaderboard(data);
            }
        });
        
        this.onMessage('error', (data) => {
            console.error('WebSocket error:', data.message);
            this.showError(data.message);
        });
    }
    
    connect() {
        try {
            // Get auth token
            const token = localStorage.getItem('auth_token') || this.getCookie('auth_token');
            if (!token) {
                console.error('No auth token found');
                this.updateConnectionStatus('disconnected', window.i18n ? window.i18n.t('websocket.not_authenticated') : 'Not authenticated');
                return;
            }
            
            // Create WebSocket connection
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/ws?token=${encodeURIComponent(token)}`;
            
            this.ws = new WebSocket(wsUrl);
            this.updateConnectionStatus('connecting', window.i18n ? window.i18n.t('game.connecting') : 'Connecting...');
            
            this.ws.onopen = () => {
                console.log('WebSocket connected');
                this.reconnectAttempts = 0;
                this.updateConnectionStatus('connected', window.i18n ? window.i18n.t('game.connected') : 'Connected');
            };
            
            this.ws.onmessage = (event) => {
                try {
                    const message = JSON.parse(event.data);
                    this.handleMessage(message);
                } catch (error) {
                    console.error('Failed to parse WebSocket message:', error);
                }
            };
            
            this.ws.onclose = (event) => {
                console.log('WebSocket disconnected:', event.code, event.reason);
                this.updateConnectionStatus('disconnected', window.i18n ? window.i18n.t('game.disconnected') : 'Disconnected');
                
                // Attempt to reconnect
                if (this.reconnectAttempts < this.maxReconnectAttempts) {
                    this.reconnectAttempts++;
                    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
                    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);
                    
                    setTimeout(() => {
                        this.connect();
                    }, delay);
                } else {
                    this.updateConnectionStatus('disconnected', window.i18n ? window.i18n.t('websocket.connection_failed') : 'Connection failed');
                    this.showError(window.i18n ? window.i18n.t('websocket.connection_lost') : 'Connection lost. Please refresh the page.');
                }
            };
            
            this.ws.onerror = (error) => {
                console.error('WebSocket error:', error);
                this.updateConnectionStatus('disconnected', window.i18n ? window.i18n.t('websocket.connection_error') : 'Connection error');
            };
            
        } catch (error) {
            console.error('Failed to create WebSocket connection:', error);
            this.updateConnectionStatus('disconnected', window.i18n ? window.i18n.t('websocket.connection_failed') : 'Connection failed');
        }
    }
    
    disconnect() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
    }
    
    send(type, data = {}) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            const message = {
                type: type,
                data: data
            };
            this.ws.send(JSON.stringify(message));
        } else {
            console.error('WebSocket not connected');
            this.showError(window.i18n ? window.i18n.t('websocket.not_connected') : 'Not connected to server');
        }
    }
    
    onMessage(type, handler) {
        this.messageHandlers.set(type, handler);
    }
    
    handleMessage(message) {
        const handler = this.messageHandlers.get(message.type);
        if (handler) {
            handler(message.data);
        } else {
            console.warn('No handler for message type:', message.type);
        }
    }
    
    updateConnectionStatus(status, text) {
        this.connectionStatus = status;
        
        const statusIndicator = document.getElementById('status-indicator');
        const statusText = document.getElementById('status-text');
        
        if (statusIndicator && statusText) {
            statusIndicator.className = `status-indicator ${status}`;
            statusText.textContent = text;
        }
    }
    
    showError(message) {
        // Create or update error notification
        let errorDiv = document.getElementById('error-notification');
        if (!errorDiv) {
            errorDiv = document.createElement('div');
            errorDiv.id = 'error-notification';
            errorDiv.style.cssText = `
                position: fixed;
                top: 20px;
                right: 20px;
                background: #f44336;
                color: white;
                padding: 15px 20px;
                border-radius: 8px;
                box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
                z-index: 1000;
                max-width: 300px;
                font-size: 14px;
                line-height: 1.4;
            `;
            document.body.appendChild(errorDiv);
        }
        
        errorDiv.textContent = message;
        errorDiv.style.display = 'block';
        
        // Auto-hide after 5 seconds
        setTimeout(() => {
            if (errorDiv) {
                errorDiv.style.display = 'none';
            }
        }, 5000);
    }
    
    getCookie(name) {
        const value = `; ${document.cookie}`;
        const parts = value.split(`; ${name}=`);
        if (parts.length === 2) return parts.pop().split(';').shift();
        return null;
    }
}

// Initialize WebSocket connection
window.gameWS = new GameWebSocket();
