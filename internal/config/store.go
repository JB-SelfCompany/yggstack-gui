package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/JB-SelfCompany/yggstack-gui/internal/platform"
)

// Store handles configuration persistence
type Store struct {
	mu       sync.RWMutex
	path     string
	settings *Settings
}

// NewStore creates a new configuration store
func NewStore() *Store {
	return &Store{
		path:     platform.GetConfigPath(),
		settings: DefaultSettings(),
	}
}

// NewStoreWithPath creates a new configuration store with a custom path
func NewStoreWithPath(path string) *Store {
	return &Store{
		path:     path,
		settings: DefaultSettings(),
	}
}

// Load reads configuration from disk
func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, use defaults
			return nil
		}
		return err
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return err
	}

	s.settings = &settings
	return nil
}

// Save writes configuration to disk
func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Ensure directory exists
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s.settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0644)
}

// Get returns a copy of the current settings
func (s *Store) Get() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return *s.settings
}

// Set updates the settings
func (s *Store) Set(settings *Settings) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings = settings
}

// Update applies a function to update settings
func (s *Store) Update(fn func(*Settings)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(s.settings)
}

// GetLanguage returns the current language
func (s *Store) GetLanguage() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings.App.Language
}

// SetLanguage sets the language
func (s *Store) SetLanguage(lang string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings.App.Language = lang
}

// GetTheme returns the current theme
func (s *Store) GetTheme() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings.App.Theme
}

// SetTheme sets the theme
func (s *Store) SetTheme(theme string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings.App.Theme = theme
}
