package config

// Settings represents all application configuration
type Settings struct {
	App      AppSettings      `json:"app"`
	Node     NodeSettings     `json:"node"`
	Proxy    ProxySettings    `json:"proxy"`
	Mappings MappingsSettings `json:"mappings"`
}

// AppSettings contains general application settings
type AppSettings struct {
	Language       string `json:"language"`       // "en", "ru"
	Theme          string `json:"theme"`          // "light", "dark", "system"
	MinimizeToTray bool   `json:"minimizeToTray"`
	StartMinimized bool   `json:"startMinimized"`
	Autostart      bool   `json:"autostart"`
	LogLevel       string `json:"logLevel"`       // "debug", "info", "warn", "error"
}

// NodeSettings contains Yggdrasil node settings
type NodeSettings struct {
	ConfigPath  string `json:"configPath"`
	AutoConnect bool   `json:"autoConnect"`
}

// ProxySettings contains SOCKS5 proxy settings
type ProxySettings struct {
	Enabled       bool   `json:"enabled"`
	ListenAddress string `json:"listenAddress"`
	Nameserver    string `json:"nameserver"`
}

// MappingsSettings contains port forwarding mappings
type MappingsSettings struct {
	LocalTCP  []PortMapping `json:"localTcp"`
	RemoteTCP []PortMapping `json:"remoteTcp"`
	LocalUDP  []PortMapping `json:"localUdp"`
	RemoteUDP []PortMapping `json:"remoteUdp"`
}

// PortMapping represents a single port forwarding rule
type PortMapping struct {
	ID      string `json:"id"`
	Source  string `json:"source"`
	Target  string `json:"target"`
	Enabled bool   `json:"enabled"`
}

// Validate checks if the settings are valid
func (s *Settings) Validate() error {
	// Validate language
	if s.App.Language != "en" && s.App.Language != "ru" {
		s.App.Language = "en"
	}

	// Validate theme
	if s.App.Theme != "light" && s.App.Theme != "dark" && s.App.Theme != "system" {
		s.App.Theme = "dark"
	}

	// Validate log level
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[s.App.LogLevel] {
		s.App.LogLevel = "info"
	}

	return nil
}
