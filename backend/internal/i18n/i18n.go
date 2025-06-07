package i18n

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"sync"

	"embed"
)

//go:embed locales/*.json
var localeFiles embed.FS

// I18n represents the internationalization manager
type I18n struct {
	defaultLang string
	languages   map[string]map[string]string
	mu          sync.RWMutex
}

// New creates a new I18n instance
func New(defaultLang string) *I18n {
	i18n := &I18n{
		defaultLang: defaultLang,
		languages:   make(map[string]map[string]string),
	}
	
	// Load default languages
	i18n.loadLanguages()
	
	return i18n
}

// loadLanguages loads all language files from embedded filesystem
func (i *I18n) loadLanguages() {
	supportedLangs := []string{"en", "zh-CN", "zh-TW", "ja", "ko", "es", "fr", "de", "ru"}
	
	for _, lang := range supportedLangs {
		if err := i.LoadLanguage(lang); err != nil {
			// If loading fails, create empty map to avoid panics
			i.languages[lang] = make(map[string]string)
		}
	}
}

// LoadLanguage loads a specific language file
func (i *I18n) LoadLanguage(lang string) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	
	filename := fmt.Sprintf("locales/%s.json", lang)
	
	data, err := localeFiles.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read language file %s: %w", filename, err)
	}
	
	var translations map[string]string
	if err := json.Unmarshal(data, &translations); err != nil {
		return fmt.Errorf("failed to parse language file %s: %w", filename, err)
	}
	
	i.languages[lang] = translations
	return nil
}

// T translates a key for the given language
func (i *I18n) T(lang, key string) string {
	i.mu.RLock()
	defer i.mu.RUnlock()
	
	// Try the requested language first
	if translations, ok := i.languages[lang]; ok {
		if translation, exists := translations[key]; exists {
			return translation
		}
	}
	
	// Fallback to default language
	if lang != i.defaultLang {
		if translations, ok := i.languages[i.defaultLang]; ok {
			if translation, exists := translations[key]; exists {
				return translation
			}
		}
	}
	
	// Return the key itself if no translation found
	return key
}

// Tf translates a key with format arguments
func (i *I18n) Tf(lang, key string, args ...interface{}) string {
	translation := i.T(lang, key)
	return fmt.Sprintf(translation, args...)
}

// GetSupportedLanguages returns all supported languages
func (i *I18n) GetSupportedLanguages() []string {
	i.mu.RLock()
	defer i.mu.RUnlock()
	
	var langs []string
	for lang := range i.languages {
		langs = append(langs, lang)
	}
	return langs
}

// GetLanguageName returns the native name of a language
func (i *I18n) GetLanguageName(lang string) string {
	names := map[string]string{
		"en":    "English",
		"zh-CN": "简体中文",
		"zh-TW": "繁體中文",
		"ja":    "日本語",
		"ko":    "한국어",
		"es":    "Español",
		"fr":    "Français",
		"de":    "Deutsch",
		"ru":    "Русский",
	}
	
	if name, ok := names[lang]; ok {
		return name
	}
	return lang
}

// DetectLanguage detects language from Accept-Language header
func (i *I18n) DetectLanguage(acceptLang string) string {
	if acceptLang == "" {
		return i.defaultLang
	}
	
	// Parse Accept-Language header
	languages := parseAcceptLanguage(acceptLang)
	
	// Find the first supported language
	for _, lang := range languages {
		if _, ok := i.languages[lang]; ok {
			return lang
		}
		
		// Try language without region (e.g., "zh" from "zh-CN")
		if strings.Contains(lang, "-") {
			baseLang := strings.Split(lang, "-")[0]
			for supportedLang := range i.languages {
				if strings.HasPrefix(supportedLang, baseLang) {
					return supportedLang
				}
			}
		}
	}
	
	return i.defaultLang
}

// parseAcceptLanguage parses the Accept-Language header
func parseAcceptLanguage(acceptLang string) []string {
	var languages []string
	
	parts := strings.Split(acceptLang, ",")
	for _, part := range parts {
		lang := strings.TrimSpace(part)
		if idx := strings.Index(lang, ";"); idx != -1 {
			lang = lang[:idx]
		}
		lang = strings.TrimSpace(lang)
		if lang != "" {
			languages = append(languages, lang)
		}
	}
	
	return languages
}

// TemplateFuncMap returns template functions for use in HTML templates
func (i *I18n) TemplateFuncMap(lang string) template.FuncMap {
	return template.FuncMap{
		"t": func(key string) string {
			return i.T(lang, key)
		},
		"tf": func(key string, args ...interface{}) string {
			return i.Tf(lang, key, args...)
		},
		"langName": func(langCode string) string {
			return i.GetLanguageName(langCode)
		},
		"currentLang": func() string {
			return lang
		},
	}
}
