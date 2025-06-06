class CanvasGame {
    constructor(canvasId, websocket) {
        this.canvas = document.getElementById(canvasId);
        this.ctx = this.canvas.getContext('2d');
        this.ws = websocket;

        // Game state
        this.board = Array(4).fill().map(() => Array(4).fill(0));
        this.score = 0;
        this.victory = false;
        this.gameOver = false;

        // Canvas settings
        this.setupCanvas();

        // Modern flat design colors
        this.colors = {
            background: '#faf8ef',
            empty: 'rgba(238, 228, 218, 0.35)',
            text: '#776e65',
            textLight: '#f9f6f2',
            tiles: {
                2: '#eee4da',
                4: '#ede0c8',
                8: '#f2b179',
                16: '#f59563',
                32: '#f67c5f',
                64: '#f65e3b',
                128: '#edcf72',
                256: '#edcc61',
                512: '#edc850',
                1024: '#edc53f',
                2048: '#edc22e',
                4096: '#ee5a24',
                8192: '#2c3e50',
                16384: '#9b59b6'
            }
        };

        // Animation
        this.animationFrame = null;
        this.animations = [];
        this.mergeAnimations = [];
        this.newTileAnimations = [];
        this.moveAnimations = [];
        this.particles = [];
        this.isAnimating = false;
        this.lastMoveDirection = null;

        // Input handling
        this.setupInputHandlers();

        // Initial render
        this.render();

        // Check for cached game state when WebSocket is connected
        if (this.ws) {
            this.checkCachedGameState();
        }
    }

    // Check if there's a cached game state and apply it
    checkCachedGameState() {
        if (window.gameWS && window.gameWS.cachedGameState) {
            console.log('Applying cached game state');
            this.updateGameState(window.gameWS.cachedGameState);
            window.gameWS.cachedGameState = null; // Clear cache
        }
    }

    // Set WebSocket connection (called when connection is established)
    setWebSocket(websocket) {
        this.ws = websocket;
        this.checkCachedGameState();
    }
    
    setupCanvas() {
        // Calculate optimal size for different screen sizes
        const isMobile = window.innerWidth <= 600;
        const maxSize = isMobile ? Math.min(window.innerWidth - 16, window.innerHeight * 0.5) : 500;
        const size = Math.min(window.innerWidth - (isMobile ? 16 : 40), maxSize);

        // Calculate dimensions properly
        this.padding = isMobile ? 8 : 12;
        this.gap = isMobile ? 6 : 8;
        this.tileSize = (size - 2 * this.padding - 3 * this.gap) / 4; // 3 gaps between 4 tiles

        // High DPI support
        const dpr = window.devicePixelRatio || 1;

        // Set actual canvas size (for drawing)
        this.canvas.width = size * dpr;
        this.canvas.height = size * dpr;

        // Set display size (CSS)
        this.canvas.style.width = size + 'px';
        this.canvas.style.height = size + 'px';

        // Scale context for high DPI
        this.ctx.scale(dpr, dpr);

        // Store the logical size for drawing calculations
        this.canvasSize = size;
    }
    
    setupInputHandlers() {
        // Touch handling
        let startX, startY;
        
        this.canvas.addEventListener('touchstart', (e) => {
            e.preventDefault();
            const touch = e.touches[0];
            startX = touch.clientX;
            startY = touch.clientY;
        });
        
        this.canvas.addEventListener('touchend', (e) => {
            e.preventDefault();
            if (!startX || !startY) return;
            
            const touch = e.changedTouches[0];
            const deltaX = touch.clientX - startX;
            const deltaY = touch.clientY - startY;
            
            const minSwipeDistance = 30;
            
            if (Math.abs(deltaX) > Math.abs(deltaY)) {
                if (Math.abs(deltaX) > minSwipeDistance) {
                    this.handleMove(deltaX > 0 ? 'right' : 'left');
                }
            } else {
                if (Math.abs(deltaY) > minSwipeDistance) {
                    this.handleMove(deltaY > 0 ? 'down' : 'up');
                }
            }
            
            startX = startY = null;
        });
        
        // Keyboard handling
        document.addEventListener('keydown', (e) => {
            if (this.gameOver || this.victory) return;
            
            switch(e.key) {
                case 'ArrowUp':
                case 'w':
                case 'W':
                    e.preventDefault();
                    this.handleMove('up');
                    break;
                case 'ArrowDown':
                case 's':
                case 'S':
                    e.preventDefault();
                    this.handleMove('down');
                    break;
                case 'ArrowLeft':
                case 'a':
                case 'A':
                    e.preventDefault();
                    this.handleMove('left');
                    break;
                case 'ArrowRight':
                case 'd':
                case 'D':
                    e.preventDefault();
                    this.handleMove('right');
                    break;
            }
        });
    }
    
    handleMove(direction) {
        // Prevent moves during animations to avoid visual glitches
        if (this.isAnimating) {
            return;
        }

        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.lastMoveDirection = direction;
            this.ws.send(JSON.stringify({
                type: 'move',
                data: {
                    direction: direction
                }
            }));
        }
    }
    
    updateGameState(gameState) {
        const oldBoard = this.board.map(row => [...row]); // Deep copy
        const newBoard = gameState.board;

        // Detect merges and new tiles
        this.detectAnimations(oldBoard, newBoard);

        this.board = newBoard;
        this.score = gameState.score;
        this.victory = gameState.victory;
        this.gameOver = gameState.game_over;

        // Update score display
        const scoreElement = document.getElementById('score');
        if (scoreElement) {
            scoreElement.textContent = this.score.toLocaleString();
        }

        // Start animations if any
        if (this.mergeAnimations.length > 0 || this.newTileAnimations.length > 0 || this.moveAnimations.length > 0) {
            this.startAnimations();
        } else {
            this.render();
        }

        // Show game over/victory overlay
        if (this.victory || this.gameOver) {
            this.showGameOverlay();
        }
    }

    detectAnimations(oldBoard, newBoard) {
        this.mergeAnimations = [];
        this.newTileAnimations = [];
        this.moveAnimations = [];

        // Track tiles to implement move animations using a more sophisticated approach
        const moveMap = this.calculateTileMoves(oldBoard, newBoard);

        // Create move animations for tiles that actually moved
        for (const move of moveMap) {
            this.moveAnimations.push({
                fromRow: move.fromRow,
                fromCol: move.fromCol,
                toRow: move.toRow,
                toCol: move.toCol,
                value: move.value,
                startTime: Date.now(),
                duration: 150 // Smooth move animation
            });
        }

        // Count total tiles to detect if merges occurred
        const oldTileCount = oldBoard.flat().filter(v => v > 0).length;
        const newTileCount = newBoard.flat().filter(v => v > 0).length;
        const mergesOccurred = oldTileCount > newTileCount;

        // Create a set of positions that are moving for quick lookup
        const movingPositions = new Set(moveMap.map(move => `${move.toRow},${move.toCol}`));

        // Find all positions that changed for merges and new tiles
        for (let row = 0; row < 4; row++) {
            for (let col = 0; col < 4; col++) {
                const oldValue = oldBoard[row][col];
                const newValue = newBoard[row][col];

                // Skip if this position has no change
                if (oldValue === newValue) {
                    continue;
                }

                // Case 1: Tile value doubled at same position (definite merge)
                if (oldValue > 0 && newValue === oldValue * 2) {
                    this.mergeAnimations.push({
                        row: row,
                        col: col,
                        value: newValue,
                        startTime: Date.now(),
                        duration: 200
                    });
                    this.createMergeParticles(row, col);
                }
                // Case 2: New tile appeared in previously empty space
                else if (oldValue === 0 && newValue > 0) {
                    // Only treat as new tile if it's not part of a move animation
                    if (!movingPositions.has(`${row},${col}`)) {
                        // Only treat as new tile if no merges occurred, or if it's a small value (2 or 4)
                        if (!mergesOccurred || newValue <= 4) {
                            this.newTileAnimations.push({
                                row: row,
                                col: col,
                                value: newValue,
                                startTime: Date.now(),
                                duration: 150
                            });
                        }
                        // If merges occurred and it's a larger value, it's likely a merge result
                        else {
                            // Only show merge animation if we can confirm it's actually a merge
                            if (this.isLikelyMergeResult(oldBoard, newBoard, row, col, newValue)) {
                                this.mergeAnimations.push({
                                    row: row,
                                    col: col,
                                    value: newValue,
                                    startTime: Date.now(),
                                    duration: 200
                                });
                                this.createMergeParticles(row, col);
                            }
                        }
                    }
                }
            }
        }
    }

    // Advanced tile movement calculation
    calculateTileMoves(oldBoard, newBoard) {
        if (!this.lastMoveDirection) {
            return [];
        }

        const moves = [];
        
        // For each direction, simulate the movement and track tile positions
        switch (this.lastMoveDirection) {
            case 'left':
                for (let row = 0; row < 4; row++) {
                    moves.push(...this.trackRowMovement(oldBoard, newBoard, row, 'left'));
                }
                break;
            case 'right':
                for (let row = 0; row < 4; row++) {
                    moves.push(...this.trackRowMovement(oldBoard, newBoard, row, 'right'));
                }
                break;
            case 'up':
                for (let col = 0; col < 4; col++) {
                    moves.push(...this.trackColMovement(oldBoard, newBoard, col, 'up'));
                }
                break;
            case 'down':
                for (let col = 0; col < 4; col++) {
                    moves.push(...this.trackColMovement(oldBoard, newBoard, col, 'down'));
                }
                break;
        }
        
        return moves;
    }

    // Track tile movements in a row
    trackRowMovement(oldBoard, newBoard, row, direction) {
        const moves = [];
        const oldRow = oldBoard[row];
        const newRow = newBoard[row];
        
        // Extract non-zero tiles from old and new rows
        const oldTiles = [];
        const newTiles = [];
        
        for (let col = 0; col < 4; col++) {
            if (oldRow[col] > 0) {
                oldTiles.push({ value: oldRow[col], originalCol: col });
            }
            if (newRow[col] > 0) {
                newTiles.push({ value: newRow[col], finalCol: col });
            }
        }
        
        if (direction === 'left') {
            // For left movement, tiles move to lower indices
            let oldIndex = 0;
            for (let newIndex = 0; newIndex < newTiles.length && oldIndex < oldTiles.length; newIndex++) {
                const newTile = newTiles[newIndex];
                const oldTile = oldTiles[oldIndex];
                
                if (newTile.value === oldTile.value) {
                    // Simple move
                    if (newTile.finalCol !== oldTile.originalCol) {
                        moves.push({
                            fromRow: row,
                            fromCol: oldTile.originalCol,
                            toRow: row,
                            toCol: newTile.finalCol,
                            value: oldTile.value
                        });
                    }
                    oldIndex++;
                } else if (newTile.value === oldTile.value * 2 && 
                          oldIndex + 1 < oldTiles.length && 
                          oldTiles[oldIndex + 1].value === oldTile.value) {
                    // Merge: two tiles of same value became one with double value
                    // Don't create move animation for merges, let merge animation handle it
                    oldIndex += 2; // Skip both merged tiles
                } else {
                    oldIndex++;
                }
            }
        } else if (direction === 'right') {
            // For right movement, tiles move to higher indices
            let oldIndex = oldTiles.length - 1;
            for (let newIndex = newTiles.length - 1; newIndex >= 0 && oldIndex >= 0; newIndex--) {
                const newTile = newTiles[newIndex];
                const oldTile = oldTiles[oldIndex];
                
                if (newTile.value === oldTile.value) {
                    // Simple move
                    if (newTile.finalCol !== oldTile.originalCol) {
                        moves.push({
                            fromRow: row,
                            fromCol: oldTile.originalCol,
                            toRow: row,
                            toCol: newTile.finalCol,
                            value: oldTile.value
                        });
                    }
                    oldIndex--;
                } else if (newTile.value === oldTile.value * 2 && 
                          oldIndex - 1 >= 0 && 
                          oldTiles[oldIndex - 1].value === oldTile.value) {
                    // Merge: two tiles of same value became one with double value
                    oldIndex -= 2; // Skip both merged tiles
                } else {
                    oldIndex--;
                }
            }
        }
        
        return moves;
    }

    // Track tile movements in a column
    trackColMovement(oldBoard, newBoard, col, direction) {
        const moves = [];
        const oldCol = oldBoard.map(row => row[col]);
        const newCol = newBoard.map(row => row[col]);
        
        // Extract non-zero tiles from old and new columns
        const oldTiles = [];
        const newTiles = [];
        
        for (let row = 0; row < 4; row++) {
            if (oldCol[row] > 0) {
                oldTiles.push({ value: oldCol[row], originalRow: row });
            }
            if (newCol[row] > 0) {
                newTiles.push({ value: newCol[row], finalRow: row });
            }
        }
        
        if (direction === 'up') {
            // For up movement, tiles move to lower indices
            let oldIndex = 0;
            for (let newIndex = 0; newIndex < newTiles.length && oldIndex < oldTiles.length; newIndex++) {
                const newTile = newTiles[newIndex];
                const oldTile = oldTiles[oldIndex];
                
                if (newTile.value === oldTile.value) {
                    // Simple move
                    if (newTile.finalRow !== oldTile.originalRow) {
                        moves.push({
                            fromRow: oldTile.originalRow,
                            fromCol: col,
                            toRow: newTile.finalRow,
                            toCol: col,
                            value: oldTile.value
                        });
                    }
                    oldIndex++;
                } else if (newTile.value === oldTile.value * 2 && 
                          oldIndex + 1 < oldTiles.length && 
                          oldTiles[oldIndex + 1].value === oldTile.value) {
                    // Merge: two tiles of same value became one with double value
                    oldIndex += 2; // Skip both merged tiles
                } else {
                    oldIndex++;
                }
            }
        } else if (direction === 'down') {
            // For down movement, tiles move to higher indices
            let oldIndex = oldTiles.length - 1;
            for (let newIndex = newTiles.length - 1; newIndex >= 0 && oldIndex >= 0; newIndex--) {
                const newTile = newTiles[newIndex];
                const oldTile = oldTiles[oldIndex];
                
                if (newTile.value === oldTile.value) {
                    // Simple move
                    if (newTile.finalRow !== oldTile.originalRow) {
                        moves.push({
                            fromRow: oldTile.originalRow,
                            fromCol: col,
                            toRow: newTile.finalRow,
                            toCol: col,
                            value: oldTile.value
                        });
                    }
                    oldIndex--;
                } else if (newTile.value === oldTile.value * 2 && 
                          oldIndex - 1 >= 0 && 
                          oldTiles[oldIndex - 1].value === oldTile.value) {
                    // Merge: two tiles of same value became one with double value
                    oldIndex -= 2; // Skip both merged tiles
                } else {
                    oldIndex--;
                }
            }
        }
        
        return moves;
    }



    isLikelyMergeResult(oldBoard, newBoard, row, col, value) {
        const halfValue = value / 2;

        // Count how many tiles of halfValue existed before and after
        const oldHalfCount = oldBoard.flat().filter(v => v === halfValue).length;
        const newHalfCount = newBoard.flat().filter(v => v === halfValue).length;

        // If we lost at least 2 tiles of halfValue, this is likely a merge
        return oldHalfCount - newHalfCount >= 2;
    }

    render() {
        // Clear the entire canvas
        this.ctx.clearRect(0, 0, this.canvasSize, this.canvasSize);

        // Draw clean background
        this.ctx.fillStyle = this.colors.background;
        const borderRadius = Math.min(this.canvasSize * 0.02, 8);
        this.drawRoundedRect(0, 0, this.canvasSize, this.canvasSize, borderRadius);
        this.ctx.fill();

        // Draw empty tiles
        for (let row = 0; row < 4; row++) {
            for (let col = 0; col < 4; col++) {
                this.drawEmptyTile(row, col);
            }
        }

        // Draw tiles with values (skip animated tiles)
        for (let row = 0; row < 4; row++) {
            for (let col = 0; col < 4; col++) {
                const value = this.board[row][col];
                if (value > 0 && !this.isAnimatingTile(row, col)) {
                    this.drawTile(row, col, value);
                }
            }
        }
    }

    isAnimatingTile(row, col) {
        return this.mergeAnimations.some(anim => anim.row === row && anim.col === col) ||
               this.newTileAnimations.some(anim => anim.row === row && anim.col === col) ||
               this.moveAnimations.some(anim => anim.toRow === row && anim.toCol === col);
    }

    createMergeParticles(row, col) {
        const centerX = this.padding + col * (this.tileSize + this.gap) + this.tileSize / 2;
        const centerY = this.padding + row * (this.tileSize + this.gap) + this.tileSize / 2;

        // Subtle particle colors
        const colors = ['#f39c12', '#e67e22', '#d35400'];

        // Create fewer, more elegant particles
        for (let i = 0; i < 6; i++) {
            const angle = (i / 6) * Math.PI * 2;
            const speed = 2 + Math.random() * 2;
            const color = colors[Math.floor(Math.random() * colors.length)];

            this.particles.push({
                x: centerX,
                y: centerY,
                vx: Math.cos(angle) * speed,
                vy: Math.sin(angle) * speed,
                life: 1.0,
                decay: 0.02,
                size: 2 + Math.random() * 2,
                color: color
            });
        }
    }

    updateParticles() {
        this.particles = this.particles.filter(particle => {
            particle.x += particle.vx;
            particle.y += particle.vy;
            particle.life -= particle.decay;
            particle.vx *= 0.95; // Friction
            particle.vy *= 0.95;

            return particle.life > 0;
        });
    }

    drawParticles() {
        this.particles.forEach(particle => {
            this.ctx.save();
            this.ctx.globalAlpha = particle.life;
            this.ctx.fillStyle = particle.color;
            this.ctx.beginPath();
            this.ctx.arc(particle.x, particle.y, particle.size, 0, Math.PI * 2);
            this.ctx.fill();
            this.ctx.restore();
        });
    }

    // Helper method to draw rounded rectangles
    drawRoundedRect(x, y, width, height, radius) {
        this.ctx.beginPath();
        this.ctx.moveTo(x + radius, y);
        this.ctx.lineTo(x + width - radius, y);
        this.ctx.quadraticCurveTo(x + width, y, x + width, y + radius);
        this.ctx.lineTo(x + width, y + height - radius);
        this.ctx.quadraticCurveTo(x + width, y + height, x + width - radius, y + height);
        this.ctx.lineTo(x + radius, y + height);
        this.ctx.quadraticCurveTo(x, y + height, x, y + height - radius);
        this.ctx.lineTo(x, y + radius);
        this.ctx.quadraticCurveTo(x, y, x + radius, y);
        this.ctx.closePath();
    }
    
    drawEmptyTile(row, col) {
        const x = this.padding + col * (this.tileSize + this.gap);
        const y = this.padding + row * (this.tileSize + this.gap);

        this.ctx.fillStyle = this.colors.empty;
        const borderRadius = Math.min(this.tileSize * 0.1, 6);
        this.drawRoundedRect(x, y, this.tileSize, this.tileSize, borderRadius);
        this.ctx.fill();
    }

    drawTile(row, col, value, scale = 1, opacity = 1, offsetX = 0, offsetY = 0) {
        const x = this.padding + col * (this.tileSize + this.gap) + offsetX;
        const y = this.padding + row * (this.tileSize + this.gap) + offsetY;

        // Save context for transformations
        this.ctx.save();

        // Apply opacity
        this.ctx.globalAlpha = opacity;

        // Apply scale transformation
        if (scale !== 1) {
            const centerX = x + this.tileSize / 2;
            const centerY = y + this.tileSize / 2;
            this.ctx.translate(centerX, centerY);
            this.ctx.scale(scale, scale);
            this.ctx.translate(-centerX, -centerY);
        }

        // Draw tile with clean flat design
        const tileColor = this.colors.tiles[value];
        this.ctx.fillStyle = tileColor || '#3c3a32';

        // Calculate border radius based on tile size
        const borderRadius = Math.min(this.tileSize * 0.1, 6);
        this.drawRoundedRect(x, y, this.tileSize, this.tileSize, borderRadius);
        this.ctx.fill();

        // Add subtle inner border for depth (only for higher values)
        if (value >= 8) {
            this.ctx.strokeStyle = 'rgba(255, 255, 255, 0.1)';
            this.ctx.lineWidth = 1;
            this.drawRoundedRect(x + 0.5, y + 0.5, this.tileSize - 1, this.tileSize - 1, borderRadius);
            this.ctx.stroke();
        }

        // Tile text
        const fontSize = this.getTileTextSize(value);
        this.ctx.font = `bold ${fontSize}px -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif`;
        this.ctx.textAlign = 'center';
        this.ctx.textBaseline = 'middle';

        const centerX = x + this.tileSize / 2;
        const centerY = y + this.tileSize / 2;

        // Text color
        this.ctx.fillStyle = value <= 4 ? this.colors.text : this.colors.textLight;
        this.ctx.fillText(value.toString(), centerX, centerY);

        // Restore context
        this.ctx.restore();
    }
    
    getTileTextSize(value) {
        const isMobile = window.innerWidth <= 600;
        const baseSize = this.tileSize * (isMobile ? 0.4 : 0.35);

        if (value < 100) return Math.max(baseSize, isMobile ? 12 : 14);
        if (value < 1000) return Math.max(baseSize * 0.85, isMobile ? 11 : 12);
        if (value < 10000) return Math.max(baseSize * 0.7, isMobile ? 10 : 11);
        return Math.max(baseSize * 0.6, isMobile ? 9 : 10);
    }
    
    showGameOverlay() {
        const overlay = document.getElementById('game-overlay');
        const message = document.getElementById('overlay-message');
        
        if (overlay && message) {
            if (this.victory) {
                message.textContent = '🎉 You Win! You merged two 8192 tiles!';
                overlay.className = 'game-overlay victory';
            } else {
                message.textContent = '😔 Game Over! No more moves available.';
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
    
    newGame() {
        this.hideGameOverlay();
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({
                type: 'new_game',
                data: {}
            }));
        }
    }
    
    startAnimations() {
        if (this.isAnimating) return;

        this.isAnimating = true;
        this.animateFrame();
    }

    animateFrame() {
        const now = Date.now();
        let hasActiveAnimations = false;

        // Update particles
        this.updateParticles();

        // Clear and draw base state
        this.render();

        // Draw merge animations
        this.mergeAnimations = this.mergeAnimations.filter(anim => {
            const elapsed = now - anim.startTime;
            const progress = Math.min(elapsed / anim.duration, 1);

            if (progress < 1) {
                hasActiveAnimations = true;

                // Bounce effect for merges
                const scale = 1 + 0.3 * Math.sin(progress * Math.PI);
                this.drawTile(anim.row, anim.col, anim.value, scale);

                return true;
            }
            return false;
        });

        // Draw move animations
        this.moveAnimations = this.moveAnimations.filter(anim => {
            const elapsed = now - anim.startTime;
            const progress = Math.min(elapsed / anim.duration, 1);

            if (progress < 1) {
                hasActiveAnimations = true;

                // Calculate smooth movement with easing
                const easedProgress = this.easeOutCubic(progress);
                
                // Calculate position interpolation
                const fromX = this.padding + anim.fromCol * (this.tileSize + this.gap);
                const fromY = this.padding + anim.fromRow * (this.tileSize + this.gap);
                const toX = this.padding + anim.toCol * (this.tileSize + this.gap);
                const toY = this.padding + anim.toRow * (this.tileSize + this.gap);
                
                const currentX = fromX + (toX - fromX) * easedProgress;
                const currentY = fromY + (toY - fromY) * easedProgress;
                
                // Calculate offset from grid position
                const offsetX = currentX - (this.padding + anim.toCol * (this.tileSize + this.gap));
                const offsetY = currentY - (this.padding + anim.toRow * (this.tileSize + this.gap));
                
                this.drawTile(anim.toRow, anim.toCol, anim.value, 1, 1, offsetX, offsetY);

                return true;
            }
            return false;
        });

        // Draw new tile animations
        this.newTileAnimations = this.newTileAnimations.filter(anim => {
            const elapsed = now - anim.startTime;
            const progress = Math.min(elapsed / anim.duration, 1);

            if (progress < 1) {
                hasActiveAnimations = true;

                // Scale up effect for new tiles with bounce
                const scale = this.easeOutBack(progress);
                this.drawTile(anim.row, anim.col, anim.value, scale);

                return true;
            }
            return false;
        });

        // Draw particles
        this.drawParticles();

        // Continue animation if there are active animations or particles
        if (hasActiveAnimations || this.particles.length > 0) {
            requestAnimationFrame(() => this.animateFrame());
        } else {
            this.isAnimating = false;
            this.render(); // Final render
        }
    }

    // Easing functions for smooth animations
    easeOutCubic(t) {
        return 1 - Math.pow(1 - t, 3);
    }

    easeOutBack(t) {
        const c1 = 1.70158;
        const c3 = c1 + 1;
        return 1 + c3 * Math.pow(t - 1, 3) + c1 * Math.pow(t - 1, 2);
    }

    // Handle window resize
    resize() {
        this.setupCanvas();
        this.render();
    }
}

// Handle window resize
window.addEventListener('resize', () => {
    if (window.canvasGame) {
        window.canvasGame.resize();
    }
});
