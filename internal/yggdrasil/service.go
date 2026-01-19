package yggdrasil

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"sync"
	"time"

	"github.com/yggdrasil-network/yggdrasil-go/src/address"
	"github.com/yggdrasil-network/yggdrasil-go/src/config"
	"github.com/yggdrasil-network/yggdrasil-go/src/core"
	"github.com/yggdrasil-network/yggdrasil-go/src/multicast"

	"github.com/JB-SelfCompany/yggstack-gui/internal/logger"
	"github.com/JB-SelfCompany/yggstack-gui/internal/yggdrasil/netstack"

	"github.com/gologme/log"
)

// ServiceState represents the current state of the Yggdrasil service
type ServiceState int

const (
	StateStopped ServiceState = iota
	StateStarting
	StateRunning
	StateStopping
)

// String returns string representation of the state
func (s ServiceState) String() string {
	switch s {
	case StateStopped:
		return "stopped"
	case StateStarting:
		return "starting"
	case StateRunning:
		return "running"
	case StateStopping:
		return "stopping"
	default:
		return "unknown"
	}
}

// NodeInfo contains information about the running node
type NodeInfo struct {
	IPv6Address string        `json:"ipv6Address"`
	Subnet      string        `json:"subnet"`
	PublicKey   string        `json:"publicKey"`
	Coords      []uint64      `json:"coords"`
	Uptime      time.Duration `json:"uptime"`
}

// StateListener is called when service state changes
type StateListener func(state ServiceState, info *NodeInfo)

// Service wraps Yggdrasil core functionality
type Service struct {
	mu            sync.RWMutex
	state         ServiceState
	configManager *ConfigManager
	startTime     time.Time
	listeners     []StateListener
	ctx           context.Context
	cancel        context.CancelFunc
	logger        *logger.Logger
	nodeInfo      *NodeInfo

	// Yggdrasil core components
	core      *core.Core
	multicast *multicast.Multicast
	netstack  *netstack.YggdrasilNetstack

	// Internal logger for yggdrasil-go
	yggLogger *log.Logger
}

// NewService creates a new Yggdrasil service
func NewService(log *logger.Logger) *Service {
	// Create internal logger for yggdrasil-go components
	yggLogger := createYggdrasilLogger(log)

	return &Service{
		state:         StateStopped,
		configManager: NewConfigManager(log),
		listeners:     make([]StateListener, 0),
		logger:        log,
		yggLogger:     yggLogger,
	}
}

// createYggdrasilLogger creates a logger compatible with yggdrasil-go
// Only errors and warnings are logged to reduce console noise
func createYggdrasilLogger(l *logger.Logger) *log.Logger {
	// Use gologme logger which yggdrasil-go expects
	// Create with io.Discard to suppress all output by default
	yggLog := log.New(io.Discard, "", 0)
	// Only enable error and warn levels to reduce noise from multicast discovery
	// "Discovered addresses" messages are at info level and will be discarded
	yggLog.EnableLevel("error")
	yggLog.EnableLevel("warn")
	return yggLog
}

// Start starts the Yggdrasil node using configuration from ConfigManager
func (s *Service) Start(cfg interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != StateStopped {
		return fmt.Errorf("service is not stopped (current state: %s)", s.state.String())
	}

	s.logger.Info("Starting Yggdrasil service")
	s.setState(StateStarting)

	// Create context for this session
	s.ctx, s.cancel = context.WithCancel(context.Background())

	// Prepare configuration from ConfigManager
	// Note: cfg parameter is accepted for interface compatibility but configuration
	// is always loaded from the shared ConfigManager
	yggCfg, err := s.prepareConfig()
	if err != nil {
		s.setState(StateStopped)
		return fmt.Errorf("failed to prepare config: %w", err)
	}

	// Setup core options
	options := []core.SetupOption{
		core.NodeInfo(yggCfg.NodeInfo),
		core.NodeInfoPrivacy(yggCfg.NodeInfoPrivacy),
	}

	// Add listen addresses
	for _, addr := range yggCfg.Listen {
		options = append(options, core.ListenAddress(addr))
	}

	// Add peers from config
	s.logger.Info("Adding peers to core options", "count", len(yggCfg.Peers))
	for _, peer := range yggCfg.Peers {
		s.logger.Debug("Adding peer to core", "uri", peer)
		options = append(options, core.Peer{URI: peer})
	}

	// Add interface-specific peers
	for intf, peers := range yggCfg.InterfacePeers {
		for _, peer := range peers {
			options = append(options, core.Peer{URI: peer, SourceInterface: intf})
		}
	}

	// Add allowed public keys
	for _, allowed := range yggCfg.AllowedPublicKeys {
		k, err := hex.DecodeString(allowed)
		if err != nil {
			s.logger.Warn("Invalid allowed public key", "key", allowed, "error", err)
			continue
		}
		options = append(options, core.AllowedPublicKey(k[:]))
	}

	// Create and start the core
	s.core, err = core.New(yggCfg.Certificate, s.yggLogger, options...)
	if err != nil {
		s.setState(StateStopped)
		return fmt.Errorf("failed to start core: %w", err)
	}

	// Setup multicast
	if err := s.setupMulticast(yggCfg); err != nil {
		s.logger.Warn("Failed to setup multicast", "error", err)
		// Continue without multicast
	}

	// Setup netstack
	s.netstack, err = netstack.CreateYggdrasilNetstack(s.core)
	if err != nil {
		s.core.Stop()
		s.setState(StateStopped)
		return fmt.Errorf("failed to create netstack: %w", err)
	}

	// Build node info from actual core
	addr := s.core.Address()
	subnet := s.core.Subnet()
	publicKey := s.core.PublicKey()

	// Calculate subnet prefix length from mask
	ones, _ := subnet.Mask.Size()

	s.nodeInfo = &NodeInfo{
		IPv6Address: addr.String(),
		Subnet:      fmt.Sprintf("%s/%d", subnet.IP.String(), ones),
		PublicKey:   hex.EncodeToString(publicKey),
		Coords:      []uint64{}, // Coords are available per-peer, not globally
	}

	s.startTime = time.Now()
	s.setState(StateRunning)

	s.logger.Info("Yggdrasil service started",
		"address", s.nodeInfo.IPv6Address,
		"subnet", s.nodeInfo.Subnet,
		"publicKey", s.nodeInfo.PublicKey[:16]+"...")

	return nil
}

// prepareConfig converts our config format to yggdrasil-go config
func (s *Service) prepareConfig() (*config.NodeConfig, error) {
	ourCfg := s.configManager.GetConfig()

	s.logger.Info("Preparing config",
		"peersCount", len(ourCfg.Peers),
		"listenCount", len(ourCfg.Listen))

	// Log each peer for debugging
	for i, peer := range ourCfg.Peers {
		s.logger.Debug("Configured peer", "index", i, "uri", peer)
	}

	// Generate new config with defaults
	yggCfg := config.GenerateConfig()

	// Override with our settings
	if len(ourCfg.Peers) > 0 {
		yggCfg.Peers = ourCfg.Peers
		s.logger.Info("Using configured peers", "count", len(ourCfg.Peers))
	} else {
		s.logger.Warn("No peers configured, using generated defaults", "count", len(yggCfg.Peers))
	}
	if len(ourCfg.Listen) > 0 {
		yggCfg.Listen = ourCfg.Listen
	}
	if len(ourCfg.AllowedPublicKeys) > 0 {
		yggCfg.AllowedPublicKeys = ourCfg.AllowedPublicKeys
	}

	// If we have stored private key, use it
	if ourCfg.PrivateKey != "" {
		privKeyBytes, err := hex.DecodeString(ourCfg.PrivateKey)
		if err == nil && len(privKeyBytes) == ed25519.PrivateKeySize {
			yggCfg.PrivateKey = config.KeyBytes(privKeyBytes)
			s.logger.Debug("Using stored private key")
		}
	}

	// Disable admin socket for GUI
	yggCfg.AdminListen = "none"

	// Store the generated keys back to our config if new
	if ourCfg.PrivateKey == "" {
		ourCfg.PrivateKey = hex.EncodeToString(yggCfg.PrivateKey)
		// Get public key from private key
		privKey := ed25519.PrivateKey(yggCfg.PrivateKey)
		publicKey := privKey.Public().(ed25519.PublicKey)
		ourCfg.PublicKey = hex.EncodeToString(publicKey)

		// Also calculate and store the address
		addr := address.AddrForKey(publicKey)
		s.logger.Info("Generated new keys", "address", net.IP(addr[:]).String())

		// Save the config with new keys
		if err := s.configManager.Save(); err != nil {
			s.logger.Warn("Failed to save config with new keys", "error", err)
		}
	}

	// CRITICAL: Regenerate certificate after changing PrivateKey
	// The certificate must match the private key for the node to work correctly
	if err := yggCfg.GenerateSelfSignedCertificate(); err != nil {
		return nil, fmt.Errorf("failed to generate certificate: %w", err)
	}
	s.logger.Debug("Generated self-signed certificate")

	return yggCfg, nil
}

// setupMulticast initializes multicast discovery
func (s *Service) setupMulticast(cfg *config.NodeConfig) error {
	if len(cfg.MulticastInterfaces) == 0 {
		return nil
	}

	options := []multicast.SetupOption{}
	for _, intf := range cfg.MulticastInterfaces {
		options = append(options, multicast.MulticastInterface{
			Regex:    regexp.MustCompile(intf.Regex),
			Beacon:   intf.Beacon,
			Listen:   intf.Listen,
			Port:     intf.Port,
			Priority: uint8(intf.Priority),
			Password: intf.Password,
		})
	}

	var err error
	s.multicast, err = multicast.New(s.core, s.yggLogger, options...)
	return err
}

// Stop stops the Yggdrasil node
func (s *Service) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != StateRunning {
		return fmt.Errorf("service is not running (current state: %s)", s.state.String())
	}

	s.logger.Info("Stopping Yggdrasil service")
	s.setState(StateStopping)

	// Cancel context
	if s.cancel != nil {
		s.cancel()
	}

	// Stop multicast
	if s.multicast != nil {
		s.multicast.Stop()
		s.multicast = nil
	}

	// Stop core
	if s.core != nil {
		s.core.Stop()
		s.core = nil
	}

	s.netstack = nil
	s.nodeInfo = nil
	s.setState(StateStopped)

	s.logger.Info("Yggdrasil service stopped")
	return nil
}

// Restart restarts the service
func (s *Service) Restart() error {
	if s.state == StateRunning {
		if err := s.Stop(); err != nil {
			return err
		}
	}
	return s.Start(nil)
}

// GetState returns the current service state
func (s *Service) GetState() ServiceState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// GetNodeInfo returns information about the running node
func (s *Service) GetNodeInfo() *NodeInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.nodeInfo == nil {
		return nil
	}
	info := *s.nodeInfo
	info.Uptime = time.Since(s.startTime)

	return &info
}

// IsRunning returns whether the service is running
func (s *Service) IsRunning() bool {
	return s.GetState() == StateRunning
}

// GetUptime returns how long the service has been running
func (s *Service) GetUptime() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.state != StateRunning {
		return 0
	}
	return time.Since(s.startTime)
}

// AddStateListener adds a listener for state changes
func (s *Service) AddStateListener(listener StateListener) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.listeners = append(s.listeners, listener)
}

// setState updates the service state and notifies listeners
func (s *Service) setState(state ServiceState) {
	s.state = state

	// Notify listeners (copy to avoid holding lock)
	listeners := make([]StateListener, len(s.listeners))
	copy(listeners, s.listeners)

	go func() {
		for _, listener := range listeners {
			listener(state, s.nodeInfo)
		}
	}()
}

// AddPeer adds a peer to the running node
func (s *Service) AddPeer(uri string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != StateRunning || s.core == nil {
		return fmt.Errorf("service is not running")
	}

	s.logger.Info("Adding peer", "uri", uri)

	// Parse URI to *url.URL
	u, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("invalid peer URI: %w", err)
	}

	// Add peer through core API
	if err := s.core.AddPeer(u, ""); err != nil {
		return fmt.Errorf("failed to add peer: %w", err)
	}

	return nil
}

// RemovePeer removes a peer from the running node
func (s *Service) RemovePeer(uri string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != StateRunning || s.core == nil {
		return fmt.Errorf("service is not running")
	}

	s.logger.Info("Removing peer", "uri", uri)

	// Parse URI to *url.URL
	u, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("invalid peer URI: %w", err)
	}

	// Remove peer through core API
	if err := s.core.RemovePeer(u, ""); err != nil {
		return fmt.Errorf("failed to remove peer: %w", err)
	}

	return nil
}

// GetPeers returns information about connected peers
func (s *Service) GetPeers() []core.PeerInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.core == nil {
		return nil
	}

	return s.core.GetPeers()
}

// GetSessions returns information about active sessions
func (s *Service) GetSessions() []core.SessionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.core == nil {
		return nil
	}

	return s.core.GetSessions()
}

// GetCore returns the underlying yggdrasil core (for advanced operations)
func (s *Service) GetCore() *core.Core {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.core
}

// GetNetstack returns the netstack for network operations
func (s *Service) GetNetstack() *netstack.YggdrasilNetstack {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.netstack
}

// GetMTU returns the MTU for the Yggdrasil network
func (s *Service) GetMTU() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.core == nil {
		return 65535 // Default MTU
	}

	return s.core.MTU()
}

// ConfigManager returns the config manager
func (s *Service) ConfigManager() *ConfigManager {
	return s.configManager
}
