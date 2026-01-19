package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestNewStore(t *testing.T) {
	s := NewStore()
	if s == nil {
		t.Fatal("NewStore returned nil")
	}
	if s.settings == nil {
		t.Fatal("settings not initialized")
	}
}

func TestNewStoreWithPath(t *testing.T) {
	customPath := "/custom/path/config.json"
	s := NewStoreWithPath(customPath)

	if s == nil {
		t.Fatal("NewStoreWithPath returned nil")
	}
	if s.path != customPath {
		t.Errorf("path = %q, want %q", s.path, customPath)
	}
}

func TestStoreSaveLoad(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "yggstack-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	s := NewStoreWithPath(configPath)

	// Modify settings
	s.SetLanguage("ru")
	s.SetTheme("light")

	// Save
	if err := s.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file not created")
	}

	// Create new store and load
	s2 := NewStoreWithPath(configPath)
	if err := s2.Load(); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Verify loaded settings
	if s2.GetLanguage() != "ru" {
		t.Errorf("loaded language = %q, want %q", s2.GetLanguage(), "ru")
	}
	if s2.GetTheme() != "light" {
		t.Errorf("loaded theme = %q, want %q", s2.GetTheme(), "light")
	}
}

func TestStoreLoadNonExistent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "yggstack-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "nonexistent", "config.json")
	s := NewStoreWithPath(configPath)

	// Load should not error for non-existent file
	if err := s.Load(); err != nil {
		t.Errorf("Load() should not error for non-existent file: %v", err)
	}

	// Should have default settings
	if s.GetLanguage() != "en" {
		t.Errorf("language should be default 'en', got %q", s.GetLanguage())
	}
}

func TestStoreLoadInvalidJSON(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "yggstack-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")

	// Write invalid JSON
	if err := os.WriteFile(configPath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("failed to write invalid file: %v", err)
	}

	s := NewStoreWithPath(configPath)
	err = s.Load()
	if err == nil {
		t.Error("Load() should error for invalid JSON")
	}
}

func TestStoreGet(t *testing.T) {
	s := NewStore()
	s.Update(func(settings *Settings) {
		settings.App.Language = "ru"
		settings.App.Theme = "light"
	})

	got := s.Get()

	// Verify it's a copy (modifications don't affect store)
	got.App.Language = "fr"

	if s.GetLanguage() != "ru" {
		t.Error("Get() should return a copy, not reference")
	}
}

func TestStoreSet(t *testing.T) {
	s := NewStore()

	newSettings := &Settings{
		App: AppSettings{
			Language: "ru",
			Theme:    "light",
			LogLevel: "debug",
		},
	}

	s.Set(newSettings)

	if s.GetLanguage() != "ru" {
		t.Errorf("GetLanguage() = %q, want %q", s.GetLanguage(), "ru")
	}
	if s.GetTheme() != "light" {
		t.Errorf("GetTheme() = %q, want %q", s.GetTheme(), "light")
	}
}

func TestStoreUpdate(t *testing.T) {
	s := NewStore()

	s.Update(func(settings *Settings) {
		settings.App.Language = "ru"
		settings.Proxy.Enabled = true
		settings.Proxy.ListenAddress = "0.0.0.0:9050"
	})

	got := s.Get()
	if got.App.Language != "ru" {
		t.Errorf("Language = %q, want %q", got.App.Language, "ru")
	}
	if !got.Proxy.Enabled {
		t.Error("Proxy.Enabled should be true")
	}
	if got.Proxy.ListenAddress != "0.0.0.0:9050" {
		t.Errorf("Proxy.ListenAddress = %q, want %q", got.Proxy.ListenAddress, "0.0.0.0:9050")
	}
}

func TestStoreGetSetLanguage(t *testing.T) {
	s := NewStore()

	if s.GetLanguage() != "en" {
		t.Errorf("default language = %q, want %q", s.GetLanguage(), "en")
	}

	s.SetLanguage("ru")
	if s.GetLanguage() != "ru" {
		t.Errorf("after SetLanguage, got %q, want %q", s.GetLanguage(), "ru")
	}
}

func TestStoreGetSetTheme(t *testing.T) {
	s := NewStore()

	if s.GetTheme() != "dark" {
		t.Errorf("default theme = %q, want %q", s.GetTheme(), "dark")
	}

	s.SetTheme("light")
	if s.GetTheme() != "light" {
		t.Errorf("after SetTheme, got %q, want %q", s.GetTheme(), "light")
	}
}

func TestStoreConcurrentAccess(t *testing.T) {
	s := NewStore()
	var wg sync.WaitGroup
	iterations := 100

	// Concurrent writes
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				s.SetLanguage("en")
			} else {
				s.SetLanguage("ru")
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.GetLanguage()
			_ = s.GetTheme()
			_ = s.Get()
		}()
	}

	wg.Wait()

	// Should not panic and language should be one of the valid values
	lang := s.GetLanguage()
	if lang != "en" && lang != "ru" {
		t.Errorf("unexpected language after concurrent access: %q", lang)
	}
}

func TestStoreSaveCreatesDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "yggstack-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Path with non-existent subdirectory
	configPath := filepath.Join(tmpDir, "subdir", "nested", "config.json")
	s := NewStoreWithPath(configPath)

	if err := s.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify directory was created
	dir := filepath.Dir(configPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("Save() should create parent directories")
	}
}

func TestStoreSaveJSONFormat(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "yggstack-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	s := NewStoreWithPath(configPath)
	s.SetLanguage("ru")

	if err := s.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Read and verify it's valid JSON
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	var loaded Settings
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("saved file is not valid JSON: %v", err)
	}

	if loaded.App.Language != "ru" {
		t.Errorf("loaded Language = %q, want %q", loaded.App.Language, "ru")
	}
}

func TestDefaultPeers(t *testing.T) {
	peers := DefaultPeers()

	if len(peers) == 0 {
		t.Fatal("DefaultPeers returned empty list")
	}

	// Verify all peers have valid URI format
	for _, peer := range peers {
		if peer == "" {
			t.Error("DefaultPeers contains empty string")
		}
		// Basic check for URI format
		if !(len(peer) > 6 && (peer[:4] == "tcp:" || peer[:4] == "tls:" || peer[:5] == "quic:")) {
			t.Errorf("Invalid peer URI format: %q", peer)
		}
	}
}

// Benchmarks
func BenchmarkStoreGet(b *testing.B) {
	s := NewStore()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.Get()
	}
}

func BenchmarkStoreGetLanguage(b *testing.B) {
	s := NewStore()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.GetLanguage()
	}
}

func BenchmarkStoreSetLanguage(b *testing.B) {
	s := NewStore()
	langs := []string{"en", "ru"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.SetLanguage(langs[i%2])
	}
}

func BenchmarkStoreUpdate(b *testing.B) {
	s := NewStore()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Update(func(settings *Settings) {
			settings.App.Language = "ru"
		})
	}
}

func BenchmarkStoreConcurrentRead(b *testing.B) {
	s := NewStore()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = s.GetLanguage()
		}
	})
}
