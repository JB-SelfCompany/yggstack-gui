package yggdrasil

import (
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/yggdrasil-network/yggdrasil-go/src/address"
	"github.com/yggdrasil-network/yggdrasil-go/src/core"
)

// PeerInfo contains information about a peer
type PeerInfo struct {
	URI       string    `json:"uri"`
	Address   string    `json:"address"`
	PublicKey string    `json:"publicKey"`
	Connected bool      `json:"connected"`
	Inbound   bool      `json:"inbound"`
	Latency   float64   `json:"latency"` // milliseconds
	RxBytes   uint64    `json:"rxBytes"`
	TxBytes   uint64    `json:"txBytes"`
	Uptime    int64     `json:"uptime"` // seconds
	LastSeen  time.Time `json:"lastSeen"`
	Priority  uint8     `json:"priority"`
}

// PeerStats contains aggregate peer statistics
type PeerStats struct {
	Total        int    `json:"total"`
	Connected    int    `json:"connected"`
	Inbound      int    `json:"inbound"`
	Outbound     int    `json:"outbound"`
	TotalRxBytes uint64 `json:"totalRxBytes"`
	TotalTxBytes uint64 `json:"totalTxBytes"`
}

// PeerManager handles peer operations
type PeerManager struct {
	mu      sync.RWMutex
	service *Service
}

// NewPeerManager creates a new peer manager
func NewPeerManager(service *Service) *PeerManager {
	return &PeerManager{
		service: service,
	}
}

// GetPeers returns information about all connected peers
func (pm *PeerManager) GetPeers() ([]PeerInfo, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if !pm.service.IsRunning() {
		return nil, fmt.Errorf("service not running")
	}

	corePeers := pm.service.GetPeers()
	if corePeers == nil {
		return []PeerInfo{}, nil
	}

	peers := make([]PeerInfo, 0, len(corePeers))
	for _, cp := range corePeers {
		peer := PeerInfo{
			URI:       cp.URI,
			PublicKey: hex.EncodeToString(cp.Key),
			Connected: cp.Up,
			Inbound:   cp.Inbound,
			RxBytes:   cp.RXBytes,
			TxBytes:   cp.TXBytes,
			Uptime:    int64(cp.Uptime.Seconds()),
			LastSeen:  time.Now(), // Use current time for connected peers
			Priority:  cp.Priority,
		}

		// Calculate address from public key
		if len(cp.Key) > 0 {
			addr := address.AddrForKey(cp.Key)
			if addr != nil {
				peer.Address = net.IP(addr[:]).String()
			}
		}

		// Calculate latency if available (from RTT)
		if cp.Latency > 0 {
			peer.Latency = float64(cp.Latency.Milliseconds())
		}

		peers = append(peers, peer)
	}

	return peers, nil
}

// GetPeerCount returns the number of connected peers
func (pm *PeerManager) GetPeerCount() int {
	if !pm.service.IsRunning() {
		return 0
	}

	corePeers := pm.service.GetPeers()
	if corePeers == nil {
		return 0
	}

	return len(corePeers)
}

// AddPeer adds a new peer
func (pm *PeerManager) AddPeer(peerURI string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Validate URI
	if err := ValidatePeerURI(peerURI); err != nil {
		return err
	}

	// Add to running service if available
	if pm.service.IsRunning() {
		if err := pm.service.AddPeer(peerURI); err != nil {
			return err
		}
	}

	// Also add to config for persistence
	cm := pm.service.ConfigManager()
	if err := cm.AddPeer(peerURI); err != nil {
		return err
	}

	return cm.Save()
}

// RemovePeer removes a peer
func (pm *PeerManager) RemovePeer(peerURI string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Remove from running service if available
	if pm.service.IsRunning() {
		if err := pm.service.RemovePeer(peerURI); err != nil {
			// Log but don't fail - peer might not be connected
			pm.service.logger.Debug("Failed to remove peer from running service", "uri", peerURI, "error", err)
		}
	}

	// Remove from config
	cm := pm.service.ConfigManager()
	if err := cm.RemovePeer(peerURI); err != nil {
		return err
	}

	return cm.Save()
}

// GetPeerByURI returns information about a specific peer
func (pm *PeerManager) GetPeerByURI(uri string) (*PeerInfo, error) {
	peers, err := pm.GetPeers()
	if err != nil {
		return nil, err
	}

	for _, peer := range peers {
		if peer.URI == uri {
			return &peer, nil
		}
	}

	return nil, fmt.Errorf("peer not found: %s", uri)
}

// GetPeerByPublicKey returns information about a peer by public key
func (pm *PeerManager) GetPeerByPublicKey(publicKey string) (*PeerInfo, error) {
	peers, err := pm.GetPeers()
	if err != nil {
		return nil, err
	}

	for _, peer := range peers {
		if peer.PublicKey == publicKey {
			return &peer, nil
		}
	}

	return nil, fmt.Errorf("peer not found with public key: %s", publicKey)
}

// GetPeerStats returns aggregate statistics about peers
func (pm *PeerManager) GetPeerStats() *PeerStats {
	peers, err := pm.GetPeers()
	if err != nil {
		return &PeerStats{}
	}

	stats := &PeerStats{
		Total:     len(peers),
		Connected: 0,
		Inbound:   0,
		Outbound:  0,
	}

	for _, peer := range peers {
		if peer.Connected {
			stats.Connected++
		}
		if peer.Inbound {
			stats.Inbound++
		} else {
			stats.Outbound++
		}
		stats.TotalRxBytes += peer.RxBytes
		stats.TotalTxBytes += peer.TxBytes
	}

	return stats
}

// GetConfiguredPeers returns the list of configured peers (from config file)
func (pm *PeerManager) GetConfiguredPeers() []string {
	return pm.service.ConfigManager().GetPeers()
}

// ValidatePeerURI validates a peer URI
func ValidatePeerURI(peerURI string) error {
	if peerURI == "" {
		return fmt.Errorf("peer URI cannot be empty")
	}

	u, err := url.Parse(peerURI)
	if err != nil {
		return fmt.Errorf("invalid URI format: %w", err)
	}

	// Check scheme
	switch u.Scheme {
	case "tcp", "tls", "quic", "ws", "wss", "unix":
		// Valid schemes
	default:
		return fmt.Errorf("unsupported scheme: %s (expected tcp, tls, quic, ws, wss, or unix)", u.Scheme)
	}

	// Check host for non-unix schemes
	if u.Scheme != "unix" && u.Host == "" {
		return fmt.Errorf("missing host in URI")
	}

	return nil
}

// SelfInfo contains information about the local node
type SelfInfo struct {
	Address   string   `json:"address"`
	Subnet    string   `json:"subnet"`
	PublicKey string   `json:"publicKey"`
	Coords    []uint64 `json:"coords"`
}

// GetSelfInfo returns information about the local node
func (pm *PeerManager) GetSelfInfo() (*SelfInfo, error) {
	if !pm.service.IsRunning() {
		return nil, fmt.Errorf("service not running")
	}

	nodeInfo := pm.service.GetNodeInfo()
	if nodeInfo == nil {
		return nil, fmt.Errorf("node info not available")
	}

	return &SelfInfo{
		Address:   nodeInfo.IPv6Address,
		Subnet:    nodeInfo.Subnet,
		PublicKey: nodeInfo.PublicKey,
		Coords:    nodeInfo.Coords,
	}, nil
}

// SessionInfo contains information about an active session
type SessionInfo struct {
	Address   string `json:"address"`
	PublicKey string `json:"publicKey"`
	RxBytes   uint64 `json:"rxBytes"`
	TxBytes   uint64 `json:"txBytes"`
	Uptime    int64  `json:"uptime"`
}

// GetSessions returns information about active sessions
func (pm *PeerManager) GetSessions() ([]SessionInfo, error) {
	if !pm.service.IsRunning() {
		return nil, fmt.Errorf("service not running")
	}

	coreSessions := pm.service.GetSessions()
	if coreSessions == nil {
		return []SessionInfo{}, nil
	}

	sessions := make([]SessionInfo, 0, len(coreSessions))
	for _, cs := range coreSessions {
		session := SessionInfo{
			PublicKey: hex.EncodeToString(cs.Key),
			RxBytes:   cs.RXBytes,
			TxBytes:   cs.TXBytes,
			Uptime:    int64(cs.Uptime.Seconds()),
		}

		// Calculate address from public key
		if len(cs.Key) > 0 {
			addr := address.AddrForKey(cs.Key)
			if addr != nil {
				session.Address = net.IP(addr[:]).String()
			}
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// TreeEntryInfo contains tree routing information
type TreeEntryInfo struct {
	Address  string `json:"address"`
	Parent   string `json:"parent"`
	Sequence uint64 `json:"sequence"`
}

// GetTreeInfo returns tree routing information
func (pm *PeerManager) GetTreeInfo() ([]TreeEntryInfo, error) {
	if !pm.service.IsRunning() {
		return nil, fmt.Errorf("service not running")
	}

	yggCore := pm.service.GetCore()
	if yggCore == nil {
		return nil, fmt.Errorf("core not available")
	}

	tree := yggCore.GetTree()
	if tree == nil {
		return []TreeEntryInfo{}, nil
	}

	entries := make([]TreeEntryInfo, 0, len(tree))
	for _, t := range tree {
		entry := TreeEntryInfo{
			Sequence: t.Sequence,
		}

		// Calculate address from public key
		if len(t.Key) > 0 {
			addr := address.AddrForKey(t.Key)
			if addr != nil {
				entry.Address = net.IP(addr[:]).String()
			}
		}

		// Parent address
		if len(t.Parent) > 0 {
			parentAddr := address.AddrForKey(t.Parent)
			if parentAddr != nil {
				entry.Parent = net.IP(parentAddr[:]).String()
			}
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// GetPaths returns path information to known nodes
func (pm *PeerManager) GetPaths() ([]core.PathEntryInfo, error) {
	if !pm.service.IsRunning() {
		return nil, fmt.Errorf("service not running")
	}

	yggCore := pm.service.GetCore()
	if yggCore == nil {
		return nil, fmt.Errorf("core not available")
	}

	return yggCore.GetPaths(), nil
}
