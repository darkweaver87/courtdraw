// Package i18n provides internationalization for CourtDraw.
// Translations are loaded from embedded YAML locale files.
// Call Load() once at startup, then use T(key) and Tf(key, args...) everywhere.
package i18n

import (
	"embed"
	"fmt"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed locales/*.yaml
var localeFS embed.FS

// Lang represents a supported language.
type Lang string

const (
	EN Lang = "en"
	FR Lang = "fr"
)

// SupportedLangs returns all supported languages in display order.
func SupportedLangs() []Lang {
	return []Lang{EN, FR}
}

// DetectSystemLang returns the user's system language preference.
// It checks LANGUAGE, LC_ALL, LC_MESSAGES, and LANG environment variables.
func DetectSystemLang() Lang {
	supported := make(map[string]Lang, len(SupportedLangs()))
	for _, l := range SupportedLangs() {
		supported[string(l)] = l
	}

	for _, env := range []string{"LANGUAGE", "LC_ALL", "LC_MESSAGES", "LANG"} {
		val := os.Getenv(env)
		if val == "" {
			continue
		}
		prefix := strings.ToLower(val)
		if len(prefix) >= 2 {
			prefix = prefix[:2]
		}
		if lang, ok := supported[prefix]; ok {
			return lang
		}
	}
	return EN
}

var (
	mu           sync.RWMutex
	translations map[Lang]map[string]string
	currentLang  Lang = EN
	loaded       bool
)

// Load reads all locale files from the embedded filesystem.
// Must be called once at startup before any T() or Tf() calls.
func Load() {
	mu.Lock()
	defer mu.Unlock()

	if loaded {
		return
	}

	translations = make(map[Lang]map[string]string)
	for _, lang := range SupportedLangs() {
		data, err := localeFS.ReadFile("locales/" + string(lang) + ".yaml")
		if err != nil {
			continue
		}
		m := make(map[string]string)
		if err := yaml.Unmarshal(data, &m); err != nil {
			continue
		}
		translations[lang] = m
	}
	loaded = true
}

// SetLang changes the active language.
func SetLang(lang Lang) {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := translations[lang]; ok {
		currentLang = lang
	}
}

// CurrentLang returns the active language.
func CurrentLang() Lang {
	mu.RLock()
	defer mu.RUnlock()
	return currentLang
}

// T returns the translation for the given key in the current language.
// Falls back to English, then to the key itself.
func T(key string) string {
	mu.RLock()
	defer mu.RUnlock()

	if m, ok := translations[currentLang]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	if currentLang != EN {
		if m, ok := translations[EN]; ok {
			if v, ok := m[key]; ok {
				return v
			}
		}
	}
	return key
}

// Tf returns a formatted translation (fmt.Sprintf with i18n).
func Tf(key string, args ...any) string {
	return fmt.Sprintf(T(key), args...)
}
