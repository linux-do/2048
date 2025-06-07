// Game board management and UI
class Game2048 {
    constructor() {
        this.board = Array(4).fill().map(() => Array(4).fill(0));
        this.score = 0;
        this.gameOver = false;
        this.victory = false;
        
        this.setupEventListeners();
        this.initializeBoard();
    }
    
    setupEventListeners() {
        // Keyboard controls
        document.addEventListener('keydown', (e) => {
            if (this.gameOver || this.victory) return;
            
            let direction = null;
            switch (e.key) {
                case 'ArrowUp':
                    direction = 'up';
                    break;
                case 'ArrowDown':
                    direction = 'down';
                    break;
                case 'ArrowLeft':
                    direction = 'left';
                    break;
                case 'ArrowRight':
                    direction = 'right';
                    break;
                default:
                    return;
            }
            
            e.preventDefault();
            this.makeMove(direction);
        });
        
        // Touch controls for mobile
        this.setupTouchControls();
    }
    
    setupTouchControls() {
        const gameBoard = document.getElementById('game-board');
        let startX = 0;
        let startY = 0;
        let endX = 0;
        let endY = 0;
        
        gameBoard.addEventListener('touchstart', (e) => {
            e.preventDefault();
            const touch = e.touches[0];
            startX = touch.clientX;
            startY = touch.clientY;
        }, { passive: false });
        
        gameBoard.addEventListener('touchmove', (e) => {
            e.preventDefault();
        }, { passive: false });
        
        gameBoard.addEventListener('touchend', (e) => {
            e.preventDefault();
            if (this.gameOver || this.victory) return;
            
            const touch = e.changedTouches[0];
            endX = touch.clientX;
            endY = touch.clientY;
            
            const deltaX = endX - startX;
            const deltaY = endY - startY;
            const minSwipeDistance = 30;
            
            if (Math.abs(deltaX) < minSwipeDistance && Math.abs(deltaY) < minSwipeDistance) {
                return;
            }
            
            let direction = null;
            if (Math.abs(deltaX) > Math.abs(deltaY)) {
                // Horizontal swipe
                direction = deltaX > 0 ? 'right' : 'left';
            } else {
                // Vertical swipe
                direction = deltaY > 0 ? 'down' : 'up';
            }
            
            if (direction) {
                this.makeMove(direction);
            }
        }, { passive: false });
    }
    
    initializeBoard() {
        const gameBoard = document.getElementById('game-board');
        gameBoard.innerHTML = '';
        
        // Create 16 tile containers
        for (let i = 0; i < 16; i++) {
            const tile = document.createElement('div');
            tile.className = 'tile';
            tile.id = `tile-${i}`;
            gameBoard.appendChild(tile);
        }
        
        this.updateDisplay();
    }
    
    makeMove(direction) {
        if (window.gameWS) {
            window.gameWS.send('move', { direction: direction });
        }
    }
    
    updateGameState(gameState) {
        this.board = gameState.board;
        this.score = gameState.score;
        this.gameOver = gameState.game_over;
        this.victory = gameState.victory;
        
        this.updateDisplay();
        
        if (gameState.message) {
            this.showMessage(gameState.message);
        }
        
        if (this.gameOver || this.victory) {
            this.showGameOverlay();
        }
    }
    
    updateDisplay() {
        // Update score
        const scoreElement = document.getElementById('score');
        if (scoreElement) {
            scoreElement.textContent = this.score.toLocaleString();
        }
        
        // Update board
        for (let row = 0; row < 4; row++) {
            for (let col = 0; col < 4; col++) {
                const index = row * 4 + col;
                const tile = document.getElementById(`tile-${index}`);
                const value = this.board[row][col];
                
                if (tile) {
                    tile.textContent = value === 0 ? '' : value;
                    tile.className = `tile ${value === 0 ? 'tile-empty' : `tile-${value}`}`;
                }
            }
        }
    }
    
    showGameOverlay() {
        const overlay = document.getElementById('game-overlay');
        const message = document.getElementById('overlay-message');
        
        if (overlay && message) {
            if (this.victory) {
                message.textContent = window.i18n ? window.i18n.t('game.victory_message') : '🎉 You Win! You merged two 8192 tiles!';
                overlay.className = 'game-overlay victory';
            } else {
                message.textContent = window.i18n ? window.i18n.t('game.game_over_message') : '😔 Game Over! No more moves available.';
                overlay.className = 'game-overlay game-over';
            }
            overlay.style.display = 'flex';
        }
    }
    
    hideGameOverlay() {
        const overlay = document.getElementById('game-overlay');
        if (overlay) {
            overlay.style.display = 'none';
        }
    }
    
    showMessage(message) {
        // Create temporary message notification
        const notification = document.createElement('div');
        notification.style.cssText = `
            position: fixed;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            background: rgba(0, 0, 0, 0.8);
            color: white;
            padding: 20px 30px;
            border-radius: 10px;
            font-size: 18px;
            font-weight: 500;
            z-index: 1000;
            pointer-events: none;
            opacity: 0;
            transition: opacity 0.3s ease;
        `;
        notification.textContent = message;
        document.body.appendChild(notification);
        
        // Animate in
        setTimeout(() => {
            notification.style.opacity = '1';
        }, 10);
        
        // Animate out and remove
        setTimeout(() => {
            notification.style.opacity = '0';
            setTimeout(() => {
                document.body.removeChild(notification);
            }, 300);
        }, 2000);
    }
}

// Global functions for button handlers
function startNewGame() {
    if (window.gameWS) {
        window.gameWS.send('new_game', {});
        if (window.game) {
            window.game.hideGameOverlay();
        }
    }
}

// Initialize game when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.game = new Game2048();
});
