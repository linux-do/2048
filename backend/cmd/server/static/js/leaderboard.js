// Leaderboard management
class Leaderboard {
    constructor() {
        this.currentType = 'daily';
        this.cache = new Map();
        
        // Load initial leaderboard
        this.loadLeaderboard('daily');
    }
    
    loadLeaderboard(type) {
        this.currentType = type;
        
        // Check cache first
        if (this.cache.has(type)) {
            this.displayLeaderboard(this.cache.get(type));
            return;
        }
        
        // Show loading state
        this.showLoading();
        
        // Request from server
        if (window.gameWS) {
            window.gameWS.send('get_leaderboard', { type: type });
        }
    }
    
    updateLeaderboard(data) {
        // Cache the data
        this.cache.set(data.type, data);
        
        // Display if it's the current type
        if (data.type === this.currentType) {
            this.displayLeaderboard(data);
        }
    }
    
    displayLeaderboard(data) {
        const content = document.getElementById('leaderboard-content');
        if (!content) return;
        
        if (!data.rankings || data.rankings.length === 0) {
            content.innerHTML = `
                <div class="empty-leaderboard">
                    <p>${window.i18n ? window.i18n.t('leaderboard.no_scores').replace('%s', data.type) : `No scores yet for ${data.type} leaderboard.`}</p>
                    <p>${window.i18n ? window.i18n.t('leaderboard.be_first') : 'Be the first to set a score!'}</p>
                </div>
            `;
            return;
        }
        
        const html = `
            <div class="leaderboard-list">
                ${data.rankings.map((entry, index) => this.renderLeaderboardEntry(entry, index)).join('')}
            </div>
        `;
        
        content.innerHTML = html;
    }
    
    renderLeaderboardEntry(entry, index) {
        const isCurrentUser = window.gameData && entry.user_id === window.gameData.user.id;
        const rankClass = index < 3 ? `rank-${index + 1}` : '';
        const userClass = isCurrentUser ? 'current-user' : '';
        
        return `
            <div class="leaderboard-entry ${rankClass} ${userClass}">
                <div class="rank">
                    ${this.getRankDisplay(entry.rank)}
                </div>
                <div class="user-info">
                    ${entry.user_avatar ? `<img src="${entry.user_avatar}" alt="${entry.user_name}" class="user-avatar">` : ''}
                    <span class="user-name">${this.escapeHtml(entry.user_name)}</span>
                    ${isCurrentUser ? `<span class="you-badge">${window.i18n ? window.i18n.t('leaderboard.you') : 'You'}</span>` : ''}
                </div>
                <div class="score">
                    ${entry.score.toLocaleString()}
                </div>
            </div>
        `;
    }
    
    getRankDisplay(rank) {
        switch (rank) {
            case 1:
                return '🥇';
            case 2:
                return '🥈';
            case 3:
                return '🥉';
            default:
                return `#${rank}`;
        }
    }
    
    showLoading() {
        const content = document.getElementById('leaderboard-content');
        if (content) {
            content.innerHTML = `
                <div class="loading">
                    <div class="loading-spinner"></div>
                    <p>${window.i18n ? window.i18n.t('leaderboard.loading') : 'Loading leaderboard...'}</p>
                </div>
            `;
        }
    }
    
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// Global function for tab switching
function switchLeaderboard(type) {
    // Update active tab
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.classList.remove('active');
    });
    document.querySelector(`[data-type="${type}"]`).classList.add('active');
    
    // Load leaderboard
    if (window.leaderboard) {
        window.leaderboard.loadLeaderboard(type);
    }
}

// Initialize leaderboard when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.leaderboard = new Leaderboard();
});
