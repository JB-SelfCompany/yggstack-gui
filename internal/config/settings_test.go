package config

import (
	"testing"
)

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()

	if s == nil {
		t.Fatal("DefaultSettings returned nil")
	}

	// Check app defaults
	if s.App.Language != "en" {
		t.Errorf("expected default language 'en', got %q", s.App.Language)
	}
	if s.App.Theme != "dark" {
		t.Errorf("expected default theme 'dark', got %q", s.App.Theme)
	}
	if s.App.LogLevel != "info" {
		t.Errorf("expected default logLevel 'info', got %q", s.App.LogLevel)
	}
	if s.App.MinimizeToTray != true {
		t.Error("expected default minimizeToTray true")
	}

	// Check node defaults
	if s.Node.AutoConnect != true {
		t.Error("expected default autoConnect true")
	}

	// Check proxy defaults
	if s.Proxy.Enabled != false {
		t.Error("expected default proxy disabled")
	}
	if s.Proxy.ListenAddress != "127.0.0.1:1080" {
		t.Errorf("expected default listen address '127.0.0.1:1080', got %q", s.Proxy.ListenAddress)
	}

	// Check mappings defaults
	if s.Mappings.LocalTCP == nil {
		t.Error("expected LocalTCP to be initialized")
	}
	if s.Mappings.RemoteTCP == nil {
		t.Error("expected RemoteTCP to be initialized")
	}
}

func TestSettingsValidate(t *testing.T) {
	tests := []struct {
		name     string
		settings Settings
		wantLang string
		wantTheme string
		wantLevel string
	}{
		{
			name: "valid settings unchanged",
			settings: Settings{
				App: AppSettings{
					Language: "ru",
					Theme:    "light",
					LogLevel: "debug",
				},
			},
			wantLang:  "ru",
			wantTheme: "light",
			wantLevel: "debug",
		},
		{
			name: "invalid language corrected",
			settings: Settings{
				App: AppSettings{
					Language: "fr",
					Theme:    "dark",
					LogLevel: "info",
				},
			},
			wantLang:  "en",
			wantTheme: "dark",
			wantLevel: "info",
		},
		{
			name: "invalid theme corrected",
			settings: Settings{
				App: AppSettings{
					Language: "en",
					Theme:    "blue",
					LogLevel: "info",
				},
			},
			wantLang:  "en",
			wantTheme: "dark",
			wantLevel: "info",
		},
		{
			name: "invalid log level corrected",
			settings: Settings{
				App: AppSettings{
					Language: "en",
					Theme:    "dark",
					LogLevel: "trace",
				},
			},
			wantLang:  "en",
			wantTheme: "dark",
			wantLevel: "info",
		},
		{
			name: "all invalid values corrected",
			settings: Settings{
				App: AppSettings{
					Language: "de",
					Theme:    "pink",
					LogLevel: "verbose",
				},
			},
			wantLang:  "en",
			wantTheme: "dark",
			wantLevel: "info",
		},
		{
			name: "empty values corrected",
			settings: Settings{
				App: AppSettings{
					Language: "",
					Theme:    "",
					LogLevel: "",
				},
			},
			wantLang:  "en",
			wantTheme: "dark",
			wantLevel: "info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.settings.Validate()
			if err != nil {
				t.Errorf("Validate() returned error: %v", err)
			}

			if tt.settings.App.Language != tt.wantLang {
				t.Errorf("Language = %q, want %q", tt.settings.App.Language, tt.wantLang)
			}
			if tt.settings.App.Theme != tt.wantTheme {
				t.Errorf("Theme = %q, want %q", tt.settings.App.Theme, tt.wantTheme)
			}
			if tt.settings.App.LogLevel != tt.wantLevel {
				t.Errorf("LogLevel = %q, want %q", tt.settings.App.LogLevel, tt.wantLevel)
			}
		})
	}
}

func TestPortMapping(t *testing.T) {
	pm := PortMapping{
		ID:      "test-id",
		Source:  "127.0.0.1:8080",
		Target:  "192.168.1.1:80",
		Enabled: true,
	}

	if pm.ID != "test-id" {
		t.Errorf("ID = %q, want %q", pm.ID, "test-id")
	}
	if pm.Source != "127.0.0.1:8080" {
		t.Errorf("Source = %q, want %q", pm.Source, "127.0.0.1:8080")
	}
	if pm.Target != "192.168.1.1:80" {
		t.Errorf("Target = %q, want %q", pm.Target, "192.168.1.1:80")
	}
	if !pm.Enabled {
		t.Error("Enabled should be true")
	}
}

func TestAppSettings(t *testing.T) {
	app := AppSettings{
		Language:       "ru",
		Theme:          "light",
		MinimizeToTray: true,
		StartMinimized: false,
		Autostart:      true,
		LogLevel:       "debug",
	}

	if app.Language != "ru" {
		t.Errorf("Language = %q, want %q", app.Language, "ru")
	}
	if app.Theme != "light" {
		t.Errorf("Theme = %q, want %q", app.Theme, "light")
	}
	if !app.MinimizeToTray {
		t.Error("MinimizeToTray should be true")
	}
	if app.StartMinimized {
		t.Error("StartMinimized should be false")
	}
	if !app.Autostart {
		t.Error("Autostart should be true")
	}
	if app.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", app.LogLevel, "debug")
	}
}

func TestNodeSettings(t *testing.T) {
	node := NodeSettings{
		ConfigPath:  "/path/to/config.json",
		AutoConnect: true,
	}

	if node.ConfigPath != "/path/to/config.json" {
		t.Errorf("ConfigPath = %q, want %q", node.ConfigPath, "/path/to/config.json")
	}
	if !node.AutoConnect {
		t.Error("AutoConnect should be true")
	}
}

func TestProxySettings(t *testing.T) {
	proxy := ProxySettings{
		Enabled:       true,
		ListenAddress: "0.0.0.0:9050",
		Nameserver:    "8.8.8.8",
	}

	if !proxy.Enabled {
		t.Error("Enabled should be true")
	}
	if proxy.ListenAddress != "0.0.0.0:9050" {
		t.Errorf("ListenAddress = %q, want %q", proxy.ListenAddress, "0.0.0.0:9050")
	}
	if proxy.Nameserver != "8.8.8.8" {
		t.Errorf("Nameserver = %q, want %q", proxy.Nameserver, "8.8.8.8")
	}
}

func TestMappingsSettings(t *testing.T) {
	mappings := MappingsSettings{
		LocalTCP: []PortMapping{
			{ID: "1", Source: "8080", Target: "192.168.1.1:80", Enabled: true},
		},
		RemoteTCP: []PortMapping{
			{ID: "2", Source: "9000", Target: "10.0.0.1:22", Enabled: false},
		},
		LocalUDP:  []PortMapping{},
		RemoteUDP: []PortMapping{},
	}

	if len(mappings.LocalTCP) != 1 {
		t.Errorf("LocalTCP len = %d, want 1", len(mappings.LocalTCP))
	}
	if len(mappings.RemoteTCP) != 1 {
		t.Errorf("RemoteTCP len = %d, want 1", len(mappings.RemoteTCP))
	}
	if len(mappings.LocalUDP) != 0 {
		t.Errorf("LocalUDP len = %d, want 0", len(mappings.LocalUDP))
	}
	if len(mappings.RemoteUDP) != 0 {
		t.Errorf("RemoteUDP len = %d, want 0", len(mappings.RemoteUDP))
	}

	if mappings.LocalTCP[0].ID != "1" {
		t.Errorf("LocalTCP[0].ID = %q, want %q", mappings.LocalTCP[0].ID, "1")
	}
	if mappings.RemoteTCP[0].Enabled {
		t.Error("RemoteTCP[0].Enabled should be false")
	}
}
