package logger

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Level != "info" {
		t.Errorf("Level = %q, want %q", cfg.Level, "info")
	}
	if !cfg.Console {
		t.Error("Console should be true by default")
	}
	if cfg.Production {
		t.Error("Production should be false by default")
	}
}

func TestNew(t *testing.T) {
	log := New()
	if log == nil {
		t.Fatal("New() returned nil")
	}
	if log.SugaredLogger == nil {
		t.Error("SugaredLogger should not be nil")
	}
	if log.underlying == nil {
		t.Error("underlying should not be nil")
	}
}

func TestNewWithConfig(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name:   "debug level",
			config: Config{Level: "debug", Console: false},
		},
		{
			name:   "info level",
			config: Config{Level: "info", Console: false},
		},
		{
			name:   "warn level",
			config: Config{Level: "warn", Console: false},
		},
		{
			name:   "error level",
			config: Config{Level: "error", Console: false},
		},
		{
			name:   "unknown level defaults to info",
			config: Config{Level: "unknown", Console: false},
		},
		{
			name:   "production mode",
			config: Config{Level: "info", Console: false, Production: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := NewWithConfig(tt.config)
			if log == nil {
				t.Fatal("NewWithConfig() returned nil")
			}
		})
	}
}

func TestNewWithConfig_FileOutput(t *testing.T) {
	// Skip on Windows due to file locking issues in tests
	if os.Getenv("OS") == "Windows_NT" {
		t.Skip("Skipping file output test on Windows")
	}

	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	cfg := Config{
		Level:    "info",
		Console:  false,
		FilePath: logFile,
	}

	log := NewWithConfig(cfg)
	if log == nil {
		t.Fatal("NewWithConfig() returned nil")
	}

	log.Info("test message")
	log.Sync()

	// Verify file was created
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("log file was not created")
	}

	// Verify content
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if !bytes.Contains(data, []byte("test message")) {
		t.Error("log file does not contain test message")
	}
}

func TestLogger_Named(t *testing.T) {
	log := NewWithConfig(Config{Level: "info", Console: false})
	namedLog := log.Named("testlogger")

	if namedLog == nil {
		t.Fatal("Named() returned nil")
	}
	if namedLog.underlying == nil {
		t.Error("underlying should not be nil")
	}
}

func TestLogger_With(t *testing.T) {
	log := NewWithConfig(Config{Level: "info", Console: false})
	childLog := log.With("key", "value")

	if childLog == nil {
		t.Fatal("With() returned nil")
	}
}

func TestLogger_Zap(t *testing.T) {
	log := NewWithConfig(Config{Level: "info", Console: false})
	zapLog := log.Zap()

	if zapLog == nil {
		t.Error("Zap() should not return nil")
	}
}

func TestLogger_Sync(t *testing.T) {
	log := NewWithConfig(Config{Level: "info", Console: false})
	// Should not panic
	log.Sync()
}

func TestLogger_SetLevel(t *testing.T) {
	log := NewWithConfig(Config{Level: "info", Console: false})
	// Should not panic
	log.SetLevel("debug")
}

func TestGetLogFilePath(t *testing.T) {
	path := GetLogFilePath()
	if path == "" {
		t.Error("GetLogFilePath() should not return empty string")
	}
	if !strings.Contains(path, "yggstack-gui") {
		t.Errorf("path %q should contain 'yggstack-gui'", path)
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"error", "error"},
		{"unknown", "info"}, // defaults to info
		{"", "info"},        // defaults to info
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level := parseLevel(tt.input)
			// Compare string representation
			levelStr := level.String()
			if levelStr != tt.expected {
				t.Errorf("parseLevel(%q) = %q, want %q", tt.input, levelStr, tt.expected)
			}
		})
	}
}
