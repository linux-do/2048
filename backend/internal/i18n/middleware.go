package i18n

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	// LanguageKey is the context key for storing the current language
	LanguageKey = "language"
	// CookieName is the name of the language cookie
	CookieName = "lang"
)

// Middleware creates a middleware function for language detection and setting
func Middleware(i18n *I18n) gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := detectLanguage(c, i18n)
		c.Set(LanguageKey, lang)
		c.Next()
	}
}

// detectLanguage detects the user's preferred language from multiple sources
func detectLanguage(c *gin.Context, i18n *I18n) string {
	// 1. Check URL parameter first (highest priority)
	if lang := c.Query("lang"); lang != "" {
		if isValidLanguage(lang, i18n) {
			// Set cookie for future requests
			setLanguageCookie(c, lang)
			return lang
		}
	}

	// 2. Check cookie
	if cookie, err := c.Cookie(CookieName); err == nil && cookie != "" {
		if isValidLanguage(cookie, i18n) {
			return cookie
		}
	}

	// 3. Check Accept-Language header
	acceptLang := c.GetHeader("Accept-Language")
	if detected := i18n.DetectLanguage(acceptLang); detected != "" {
		// Set cookie for future requests
		setLanguageCookie(c, detected)
		return detected
	}

	// 4. Use default language
	return i18n.defaultLang
}

// isValidLanguage checks if the given language is supported
func isValidLanguage(lang string, i18n *I18n) bool {
	supportedLangs := i18n.GetSupportedLanguages()
	for _, supported := range supportedLangs {
		if supported == lang {
			return true
		}
	}
	return false
}

// setLanguageCookie sets the language preference cookie
func setLanguageCookie(c *gin.Context, lang string) {
	c.SetCookie(CookieName, lang, 365*24*3600, "/", "", false, false)
}

// GetLanguage returns the current language from context
func GetLanguage(c *gin.Context) string {
	if lang, exists := c.Get(LanguageKey); exists {
		if langStr, ok := lang.(string); ok {
			return langStr
		}
	}
	return "en" // fallback
}

// SetLanguage sets the language preference and redirects
func SetLanguage(i18n *I18n) gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := c.Param("lang")
		
		if !isValidLanguage(lang, i18n) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Unsupported language",
			})
			return
		}

		// Set cookie
		setLanguageCookie(c, lang)

		// Get redirect URL from query parameter or use referer
		redirectURL := c.Query("redirect")
		if redirectURL == "" {
			redirectURL = c.GetHeader("Referer")
		}
		if redirectURL == "" {
			redirectURL = "/"
		}

		c.Redirect(http.StatusFound, redirectURL)
	}
}
