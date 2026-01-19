package yggdrasil

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/yggdrasil-network/yggdrasil-go/src/address"

	"github.com/JB-SelfCompany/yggstack-gui/internal/logger"
	"github.com/JB-SelfCompany/yggstack-gui/internal/platform"
)

// Config represents the Yggdrasil configuration
type Config struct {
	// Identity
	PrivateKey string `json:"PrivateKey,omitempty"`
	PublicKey  string `json:"PublicKey,omitempty"`

	// Network
	Peers               []string            `json:"Peers"`
	Listen              []string            `json:"Listen"`
	InterfacePeers      map[string][]string `json:"InterfacePeers,omitempty"`
	MulticastInterfaces []string            `json:"MulticastInterfaces"`
	AllowedPublicKeys   []string            `json:"AllowedPublicKeys"`

	// SOCKS Proxy settings
	SOCKS struct {
		Enabled       bool   `json:"Enabled"`
		ListenAddress string `json:"ListenAddress"`
		Nameserver    string `json:"Nameserver"`
	} `json:"SOCKS"`

	// Port forwarding mappings
	Mappings struct {
		LocalTCP  []MappingConfig `json:"LocalTCP"`
		LocalUDP  []MappingConfig `json:"LocalUDP"`
		RemoteTCP []MappingConfig `json:"RemoteTCP"`
		RemoteUDP []MappingConfig `json:"RemoteUDP"`
	} `json:"Mappings"`
}

// MappingConfig represents a port mapping configuration
type MappingConfig struct {
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
	LocalAddr   string `json:"localAddr"`
	LocalPort   int    `json:"localPort"`
	RemoteAddr  string `json:"remoteAddr"`
	RemotePort  int    `json:"remotePort"`
	Description string `json:"description,omitempty"`
}

// ConfigInfo contains non-sensitive config information for UI
type ConfigInfo struct {
	ConfigPath  string   `json:"configPath"`
	Peers       []string `json:"peers"`
	Listen      []string `json:"listen"`
	PublicKey   string   `json:"publicKey"`
	IPv6Address string   `json:"ipv6Address,omitempty"`
}

// ConfigManager handles Yggdrasil configuration
type ConfigManager struct {
	mu     sync.RWMutex
	config *Config
	path   string
	logger *logger.Logger
}

// NewConfigManager creates a new config manager
func NewConfigManager(log *logger.Logger) *ConfigManager {
	cm := &ConfigManager{
		config: defaultConfig(),
		path:   platform.GetYggdrasilConfigPath(),
		logger: log,
	}

	// Try to load existing config
	if err := cm.Load(); err != nil {
		log.Debug("Failed to load config, using defaults", "error", err)
	}

	return cm
}

// defaultConfig returns a default configuration
func defaultConfig() *Config {
	cfg := &Config{
		Peers:               DefaultPeers(),
		Listen:              []string{"tcp://0.0.0.0:0"},
		InterfacePeers:      make(map[string][]string),
		MulticastInterfaces: []string{},
		AllowedPublicKeys:   []string{},
	}
	cfg.SOCKS.Enabled = false
	cfg.SOCKS.ListenAddress = "127.0.0.1:1080"
	cfg.SOCKS.Nameserver = ""
	return cfg
}

// DefaultPeers returns default public peers
func DefaultPeers() []string {
	return []string{
		"tls://ygg.mkg20001.io:443",
		"tls://51.15.118.10:62486",
		"tcp://193.111.114.28:8080",
		"tls://ygg.yt:443",
		"tls://x-mow-1.sergeysedoy97.ru:65535",
	}
}

// Load loads configuration from disk
func (cm *ConfigManager) Load() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	data, err := os.ReadFile(cm.path)
	if err != nil {
		if os.IsNotExist(err) {
			cm.logger.Info("Config file not found, using defaults")
			return nil
		}
		return err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	// Preserve defaults if fields are empty
	if len(cfg.Listen) == 0 {
		cfg.Listen = []string{"tcp://0.0.0.0:0"}
	}
	if cfg.InterfacePeers == nil {
		cfg.InterfacePeers = make(map[string][]string)
	}

	cm.config = &cfg
	cm.logger.Info("Configuration loaded", "path", cm.path, "peers", len(cfg.Peers))
	return nil
}

// Save saves configuration to disk
func (cm *ConfigManager) Save() error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Ensure directory exists
	dir := filepath.Dir(cm.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(cm.path, data, 0600); err != nil {
		return err
	}

	cm.logger.Info("Configuration saved", "path", cm.path)
	return nil
}

// Generate generates new configuration with random keys
func (cm *ConfigManager) Generate() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Generate new Ed25519 keypair
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return err
	}

	cm.config = defaultConfig()
	cm.config.PrivateKey = hex.EncodeToString(privateKey)
	cm.config.PublicKey = hex.EncodeToString(publicKey)

	addr := address.AddrForKey(publicKey)
	cm.logger.Info("Generated new configuration",
		"publicKey", cm.config.PublicKey[:16]+"...",
		"address", net.IP(addr[:]).String())

	return nil
}

// GetPeers returns the list of configured peers
func (cm *ConfigManager) GetPeers() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	peers := make([]string, len(cm.config.Peers))
	copy(peers, cm.config.Peers)
	return peers
}

// SetPeers sets the list of peers
func (cm *ConfigManager) SetPeers(peers []string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.config.Peers = peers
}

// AddPeer adds a peer to the configuration
func (cm *ConfigManager) AddPeer(uri string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Check for duplicates
	for _, peer := range cm.config.Peers {
		if peer == uri {
			return nil
		}
	}

	cm.config.Peers = append(cm.config.Peers, uri)
	return nil
}

// RemovePeer removes a peer from the configuration
func (cm *ConfigManager) RemovePeer(uri string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for i, peer := range cm.config.Peers {
		if peer == uri {
			cm.config.Peers = append(cm.config.Peers[:i], cm.config.Peers[i+1:]...)
			return nil
		}
	}

	return nil
}

// GetPublicKey returns the public key
func (cm *ConfigManager) GetPublicKey() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config.PublicKey
}

// GetPrivateKey returns the private key
func (cm *ConfigManager) GetPrivateKey() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config.PrivateKey
}

// IsMulticastEnabled returns whether multicast is enabled
func (cm *ConfigManager) IsMulticastEnabled() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.config.MulticastInterfaces) > 0
}

// SetMulticastEnabled enables or disables multicast
func (cm *ConfigManager) SetMulticastEnabled(enabled bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if enabled {
		cm.config.MulticastInterfaces = []string{".*"}
	} else {
		cm.config.MulticastInterfaces = []string{}
	}
}

// GetSOCKSConfig returns the SOCKS proxy configuration
func (cm *ConfigManager) GetSOCKSConfig() SOCKSConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return SOCKSConfig{
		Enabled:       cm.config.SOCKS.Enabled,
		ListenAddress: cm.config.SOCKS.ListenAddress,
		Nameserver:    cm.config.SOCKS.Nameserver,
	}
}

// SetSOCKSConfig sets the SOCKS proxy configuration
func (cm *ConfigManager) SetSOCKSConfig(cfg SOCKSConfig) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.config.SOCKS.Enabled = cfg.Enabled
	cm.config.SOCKS.ListenAddress = cfg.ListenAddress
	cm.config.SOCKS.Nameserver = cfg.Nameserver
}

// GetMappings returns all port mapping configurations
func (cm *ConfigManager) GetMappings() (localTCP, localUDP, remoteTCP, remoteUDP []MappingConfig) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	localTCP = make([]MappingConfig, len(cm.config.Mappings.LocalTCP))
	copy(localTCP, cm.config.Mappings.LocalTCP)

	localUDP = make([]MappingConfig, len(cm.config.Mappings.LocalUDP))
	copy(localUDP, cm.config.Mappings.LocalUDP)

	remoteTCP = make([]MappingConfig, len(cm.config.Mappings.RemoteTCP))
	copy(remoteTCP, cm.config.Mappings.RemoteTCP)

	remoteUDP = make([]MappingConfig, len(cm.config.Mappings.RemoteUDP))
	copy(remoteUDP, cm.config.Mappings.RemoteUDP)

	return
}

// AddMapping adds a port mapping configuration
func (cm *ConfigManager) AddMapping(mappingType string, cfg MappingConfig) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	switch mappingType {
	case "local-tcp":
		cm.config.Mappings.LocalTCP = append(cm.config.Mappings.LocalTCP, cfg)
	case "local-udp":
		cm.config.Mappings.LocalUDP = append(cm.config.Mappings.LocalUDP, cfg)
	case "remote-tcp":
		cm.config.Mappings.RemoteTCP = append(cm.config.Mappings.RemoteTCP, cfg)
	case "remote-udp":
		cm.config.Mappings.RemoteUDP = append(cm.config.Mappings.RemoteUDP, cfg)
	}
}

// RemoveMapping removes a port mapping configuration by name
func (cm *ConfigManager) RemoveMapping(mappingType, name string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	removeByName := func(mappings []MappingConfig) []MappingConfig {
		for i, m := range mappings {
			if m.Name == name {
				return append(mappings[:i], mappings[i+1:]...)
			}
		}
		return mappings
	}

	switch mappingType {
	case "local-tcp":
		cm.config.Mappings.LocalTCP = removeByName(cm.config.Mappings.LocalTCP)
	case "local-udp":
		cm.config.Mappings.LocalUDP = removeByName(cm.config.Mappings.LocalUDP)
	case "remote-tcp":
		cm.config.Mappings.RemoteTCP = removeByName(cm.config.Mappings.RemoteTCP)
	case "remote-udp":
		cm.config.Mappings.RemoteUDP = removeByName(cm.config.Mappings.RemoteUDP)
	}
}

// GetConfigInfo returns non-sensitive config information
func (cm *ConfigManager) GetConfigInfo() *ConfigInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	info := &ConfigInfo{
		ConfigPath: cm.path,
		Peers:      cm.config.Peers,
		Listen:     cm.config.Listen,
		PublicKey:  cm.config.PublicKey,
	}

	// Calculate IPv6 address if public key is available
	if cm.config.PublicKey != "" {
		if pubKeyBytes, err := hex.DecodeString(cm.config.PublicKey); err == nil {
			addr := address.AddrForKey(ed25519.PublicKey(pubKeyBytes))
			info.IPv6Address = net.IP(addr[:]).String()
		}
	}

	return info
}

// GetConfig returns the raw configuration
func (cm *ConfigManager) GetConfig() *Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

// GetPath returns the configuration file path
func (cm *ConfigManager) GetPath() string {
	return cm.path
}

// GetListen returns the listen addresses
func (cm *ConfigManager) GetListen() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	listen := make([]string, len(cm.config.Listen))
	copy(listen, cm.config.Listen)
	return listen
}

// SetListen sets the listen addresses
func (cm *ConfigManager) SetListen(listen []string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.config.Listen = listen
}

// GetAllowedPublicKeys returns the allowed public keys
func (cm *ConfigManager) GetAllowedPublicKeys() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	keys := make([]string, len(cm.config.AllowedPublicKeys))
	copy(keys, cm.config.AllowedPublicKeys)
	return keys
}

// SetAllowedPublicKeys sets the allowed public keys
func (cm *ConfigManager) SetAllowedPublicKeys(keys []string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.config.AllowedPublicKeys = keys
}
