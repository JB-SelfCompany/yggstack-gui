package yggdrasil

import (
	"fmt"
	"net"
	"sync"

	"github.com/yggdrasil-network/yggdrasil-go/src/address"
)

// NOTE: SessionInfo is defined in peers.go to avoid redeclaration

// SessionStats contains aggregate session statistics
type SessionStats struct {
	Total        int    `json:"total"`
	TotalRxBytes uint64 `json:"totalRxBytes"`
	TotalTxBytes uint64 `json:"totalTxBytes"`
}

// SessionManager handles session operations
// NOTE: Stub implementation for MVP
type SessionManager struct {
	mu      sync.RWMutex
	service *Service
}

// NewSessionManager creates a new session manager
func NewSessionManager(service *Service) *SessionManager {
	return &SessionManager{
		service: service,
	}
}

// GetSessions returns information about all active sessions
func (sm *SessionManager) GetSessions() ([]SessionInfo, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.service.IsRunning() {
		return nil, fmt.Errorf("service not running")
	}

	// Get sessions from PeerManager which has the real implementation
	pm := NewPeerManager(sm.service)
	return pm.GetSessions()
}

// GetSessionCount returns the number of active sessions
func (sm *SessionManager) GetSessionCount() int {
	sessions, err := sm.GetSessions()
	if err != nil {
		return 0
	}
	return len(sessions)
}

// GetSessionByAddress returns information about a specific session
func (sm *SessionManager) GetSessionByAddress(address string) (*SessionInfo, error) {
	sessions, err := sm.GetSessions()
	if err != nil {
		return nil, err
	}

	for _, session := range sessions {
		if session.Address == address {
			return &session, nil
		}
	}

	return nil, fmt.Errorf("session not found: %s", address)
}

// GetSessionStats returns aggregate statistics about sessions
func (sm *SessionManager) GetSessionStats() *SessionStats {
	sessions, err := sm.GetSessions()
	if err != nil {
		return &SessionStats{}
	}

	stats := &SessionStats{
		Total: len(sessions),
	}

	for _, session := range sessions {
		stats.TotalRxBytes += session.RxBytes
		stats.TotalTxBytes += session.TxBytes
	}

	return stats
}

// PathInfo contains information about a routing path
type PathInfo struct {
	Address string   `json:"address"`
	Path    []uint64 `json:"path"`
}

// GetPaths returns routing paths to known destinations
func (sm *SessionManager) GetPaths() ([]PathInfo, error) {
	if !sm.service.IsRunning() {
		return nil, fmt.Errorf("service not running")
	}

	// Get paths from PeerManager
	pm := NewPeerManager(sm.service)
	corePaths, err := pm.GetPaths()
	if err != nil {
		return nil, err
	}

	paths := make([]PathInfo, 0, len(corePaths))
	for _, cp := range corePaths {
		// Calculate address from public key
		var addrStr string
		if len(cp.Key) > 0 {
			addr := address.AddrForKey(cp.Key)
			if addr != nil {
				addrStr = net.IP(addr[:]).String()
			}
		}
		paths = append(paths, PathInfo{
			Address: addrStr,
			Path:    cp.Path,
		})
	}

	return paths, nil
}
