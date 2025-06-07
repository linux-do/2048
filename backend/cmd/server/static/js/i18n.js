// i18n.js - Internationalization support for client-side
class ClientI18n {
    constructor() {
        this.currentLang = document.documentElement.lang || 'en';
        this.translations = {};
        this.loadedLanguages = new Set();
        
        // Initialize with server-provided translations if available
        if (window.i18nTranslations) {
            this.translations[this.currentLang] = window.i18nTranslations;
            this.loadedLanguages.add(this.currentLang);
        }
    }    // Load translations for a language
    async loadLanguage(lang) {
        if (this.loadedLanguages.has(lang)) {
            return;
        }

        try {
            const response = await fetch(`/api/translations/${lang}`);
            if (response.ok) {
                const data = await response.json();
                this.translations[lang] = data.translations;
                this.loadedLanguages.add(lang);
            }
        } catch (error) {
            console.error('Failed to load language:', lang, error);
        }
    }

    // Get translation for a key
    t(key, lang = null) {
        const targetLang = lang || this.currentLang;
        
        if (this.translations[targetLang] && this.translations[targetLang][key]) {
            return this.translations[targetLang][key];
        }
        
        // Fallback to English
        if (targetLang !== 'en' && this.translations['en'] && this.translations['en'][key]) {
            return this.translations['en'][key];
        }
        
        // Return key if no translation found
        return key;
    }

    // Get translation with formatting
    tf(key, ...args) {
        const translation = this.t(key);
        return this.sprintf(translation, ...args);
    }

    // Simple sprintf implementation
    sprintf(str, ...args) {
        return str.replace(/%s/g, () => args.shift() || '');
    }

    // Change language
    async changeLanguage(lang) {
        await this.loadLanguage(lang);
        this.currentLang = lang;
        
        // Update URL and reload page to get server-side translations
        const url = new URL(window.location);
        url.searchParams.set('lang', lang);
        window.location.href = url.toString();
    }

    // Load supported languages
    async loadSupportedLanguages() {
        try {
            const response = await fetch('/api/languages');
            if (response.ok) {
                return await response.json();
            }
        } catch (error) {
            console.error('Failed to load supported languages:', error);
        }
        return { languages: [] };
    }

    // Populate language selector
    async populateLanguageSelector(selectElement) {
        const data = await this.loadSupportedLanguages();
        const languages = data.languages || [];
        
        selectElement.innerHTML = '';
        
        languages.forEach(lang => {
            const option = document.createElement('option');
            option.value = lang.code;
            option.textContent = lang.name;
            option.selected = lang.code === this.currentLang;
            selectElement.appendChild(option);
        });
    }
}

// Global i18n instance
window.i18n = new ClientI18n();

// Global function for changing language
function changeLanguage(lang) {
    // Set language cookie and redirect
    window.location.href = `/lang/${lang}?redirect=${encodeURIComponent(window.location.pathname + window.location.search)}`;
}

// Update connection status text based on current language
function updateConnectionStatus(status) {
    const statusText = document.getElementById('status-text');
    if (!statusText) return;

    let textKey;
    switch (status) {
        case 'connecting':
            textKey = 'game.connecting';
            break;
        case 'connected':
            textKey = 'game.connected';
            break;
        case 'disconnected':
            textKey = 'game.disconnected';
            break;
        default:
            textKey = 'game.connecting';
    }
    
    statusText.textContent = window.i18n.t(textKey);
}

// Initialize language selector when DOM is ready
document.addEventListener('DOMContentLoaded', function() {
    const languageSelect = document.getElementById('language-select');
    if (languageSelect) {
        window.i18n.populateLanguageSelector(languageSelect);
    }
});

// Export for use in other scripts
if (typeof module !== 'undefined' && module.exports) {
    module.exports = ClientI18n;
}
