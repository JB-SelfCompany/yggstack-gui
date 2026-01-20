package config

// DefaultSettings returns the default application settings
func DefaultSettings() *Settings {
	return &Settings{
		App: AppSettings{
			Language:       "en",
			Theme:          "dark",
			MinimizeToTray: true,
			StartMinimized: false,
			Autostart:      false,
			LogLevel:       "info",
		},
		Node: NodeSettings{
			ConfigPath:  "",
			AutoConnect: true,
		},
		Proxy: ProxySettings{
			Enabled:       false,
			ListenAddress: "127.0.0.1:1080",
			Nameserver:    "",
		},
		Mappings: MappingsSettings{
			LocalTCP:  []PortMapping{},
			RemoteTCP: []PortMapping{},
			LocalUDP:  []PortMapping{},
			RemoteUDP: []PortMapping{},
		},
	}
}

// DefaultPeers returns the default list of public peers
func DefaultPeers() []string {
	return []string{
		"tcp://ekb.itrus.su:7991",
		"tcp://yggno.de:18226",
		"tcp://srv.itrus.su:7991",
	}
}
