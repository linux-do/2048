// Authentication management
class Auth {
    constructor() {
        this.setupEventListeners();
    }
    
    setupEventListeners() {
        // Handle logout
        window.logout = () => {
            this.logout();
        };
    }
    
    async logout() {
        try {
            // Call logout endpoint
            const response = await fetch('/auth/logout', {
                method: 'POST',
                credentials: 'include'
            });
            
            if (response.ok) {
                // Clear local storage
                localStorage.removeItem('auth_token');
                
                // Disconnect WebSocket
                if (window.gameWS) {
                    window.gameWS.disconnect();
                }
                
                // Redirect to login page
                window.location.href = '/';
            } else {
                console.error('Logout failed');
                this.showError('Failed to logout');
            }
        } catch (error) {
            console.error('Logout error:', error);
            this.showError('Network error during logout');
        }
    }
    
    showError(message) {
        // Create or update error notification
        let errorDiv = document.getElementById('auth-error-notification');
        if (!errorDiv) {
            errorDiv = document.createElement('div');
            errorDiv.id = 'auth-error-notification';
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
                z-index: 1000;
                max-width: 400px;
                font-size: 14px;
                line-height: 1.4;
                text-align: center;
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
}

// Initialize auth when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.auth = new Auth();
});
