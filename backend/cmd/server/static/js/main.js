// Main application initialization and coordination
(function() {
    'use strict';
    
    // Application state
    window.app = {
        initialized: false,
        components: {}
    };
    
    // Initialize the application
    function initializeApp() {
        if (window.app.initialized) return;
        
        console.log('Initializing 2048 Game Application...');
        
        // Check if we're on the game page
        if (document.getElementById('game-board')) {
            initializeGamePage();
        } else {
            initializeOtherPages();
        }
        
        // Set up global error handling
        setupErrorHandling();
        
        // Set up performance monitoring
        setupPerformanceMonitoring();
        
        window.app.initialized = true;
        console.log('Application initialized successfully');
    }
    
    function initializeGamePage() {
        console.log('Initializing game page...');
        
        // Wait for all components to be loaded
        const checkComponents = setInterval(() => {
            if (window.gameWS && window.game && window.leaderboard && window.auth) {
                clearInterval(checkComponents);
                
                // Store component references
                window.app.components = {
                    websocket: window.gameWS,
                    game: window.game,
                    leaderboard: window.leaderboard,
                    auth: window.auth
                };
                
                // Set up component interactions
                setupComponentInteractions();
                
                console.log('Game page initialized');
            }
        }, 100);
        
        // Timeout after 10 seconds
        setTimeout(() => {
            clearInterval(checkComponents);
            if (!window.app.components.websocket) {
                console.error('Failed to initialize components within timeout');
                showInitializationError();
            }
        }, 10000);
    }
    
    function initializeOtherPages() {
        console.log('Initializing non-game page...');
        
        // Set up any common functionality for non-game pages
        setupCommonFeatures();
    }
    
    function setupComponentInteractions() {
        // Set up cross-component communication
        
        // Game state updates should refresh leaderboard
        const originalUpdateGameState = window.game.updateGameState;
        window.game.updateGameState = function(gameState) {
            originalUpdateGameState.call(this, gameState);
            
            // If game finished, refresh current leaderboard
            if (gameState.game_over || gameState.victory) {
                setTimeout(() => {
                    if (window.leaderboard) {
                        window.leaderboard.loadLeaderboard(window.leaderboard.currentType);
                    }
                }, 1000);
            }
        };
        
        // WebSocket reconnection should reload leaderboard
        const originalConnect = window.gameWS.connect;
        window.gameWS.connect = function() {
            originalConnect.call(this);
            
            // Reload leaderboard when reconnected
            this.ws.addEventListener('open', () => {
                if (window.leaderboard) {
                    setTimeout(() => {
                        window.leaderboard.loadLeaderboard(window.leaderboard.currentType);
                    }, 500);
                }
            });
        };
    }
    
    function setupErrorHandling() {
        // Global error handler
        window.addEventListener('error', (event) => {
            console.error('Global error:', event.error);
            
            // Don't show error for script loading failures (common in development)
            if (event.filename && event.filename.includes('.js')) {
                return;
            }
            
            showGlobalError('An unexpected error occurred. Please refresh the page.');
        });
        
        // Unhandled promise rejection handler
        window.addEventListener('unhandledrejection', (event) => {
            console.error('Unhandled promise rejection:', event.reason);
            showGlobalError('A network error occurred. Please check your connection.');
        });
    }
    
    function setupPerformanceMonitoring() {
        // Monitor page load performance
        window.addEventListener('load', () => {
            if ('performance' in window) {
                const perfData = performance.getEntriesByType('navigation')[0];
                if (perfData) {
                    console.log('Page load time:', perfData.loadEventEnd - perfData.loadEventStart, 'ms');
                    console.log('DOM content loaded:', perfData.domContentLoadedEventEnd - perfData.domContentLoadedEventStart, 'ms');
                }
            }
        });
        
        // Monitor memory usage (if available)
        if ('memory' in performance) {
            setInterval(() => {
                const memory = performance.memory;
                if (memory.usedJSHeapSize > memory.jsHeapSizeLimit * 0.9) {
                    console.warn('High memory usage detected');
                }
            }, 30000); // Check every 30 seconds
        }
    }
    
    function setupCommonFeatures() {
        // Set up common features for all pages
        
        // Smooth scrolling for anchor links
        document.querySelectorAll('a[href^="#"]').forEach(anchor => {
            anchor.addEventListener('click', function (e) {
                e.preventDefault();
                const target = document.querySelector(this.getAttribute('href'));
                if (target) {
                    target.scrollIntoView({
                        behavior: 'smooth'
                    });
                }
            });
        });
        
        // Add loading states to buttons
        document.querySelectorAll('button[type="submit"]').forEach(button => {
            button.addEventListener('click', function() {
                if (!this.disabled) {
                    this.classList.add('loading');
                    this.disabled = true;
                    
                    // Re-enable after 5 seconds as fallback
                    setTimeout(() => {
                        this.classList.remove('loading');
                        this.disabled = false;
                    }, 5000);
                }
            });
        });
    }
    
    function showInitializationError() {
        const errorDiv = document.createElement('div');
        errorDiv.style.cssText = `
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(0, 0, 0, 0.8);
            color: white;
            display: flex;
            align-items: center;
            justify-content: center;
            z-index: 10000;
            font-family: inherit;
        `;
        
        errorDiv.innerHTML = `
            <div style="text-align: center; padding: 40px; background: white; color: #333; border-radius: 10px; max-width: 400px;">
                <h2>Initialization Failed</h2>
                <p>The game failed to load properly. Please refresh the page to try again.</p>
                <button onclick="window.location.reload()" style="background: #8f7a66; color: white; border: none; padding: 10px 20px; border-radius: 5px; cursor: pointer;">
                    Refresh Page
                </button>
            </div>
        `;
        
        document.body.appendChild(errorDiv);
    }
    
    function showGlobalError(message) {
        // Don't show multiple error messages
        if (document.getElementById('global-error')) return;
        
        const errorDiv = document.createElement('div');
        errorDiv.id = 'global-error';
        errorDiv.style.cssText = `
            position: fixed;
            top: 20px;
            left: 50%;
            transform: translateX(-50%);
            background: #f44336;
            color: white;
            padding: 15px 20px;
            border-radius: 8px;
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
            z-index: 10000;
            max-width: 400px;
            font-size: 14px;
            line-height: 1.4;
            text-align: center;
        `;
        
        errorDiv.textContent = message;
        document.body.appendChild(errorDiv);
        
        // Auto-hide after 10 seconds
        setTimeout(() => {
            if (errorDiv.parentNode) {
                errorDiv.parentNode.removeChild(errorDiv);
            }
        }, 10000);
    }
    
    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initializeApp);
    } else {
        initializeApp();
    }
    
    // Export utilities for debugging
    window.app.utils = {
        showError: showGlobalError,
        reinitialize: initializeApp
    };
    
})();
