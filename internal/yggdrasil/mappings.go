package yggdrasil

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/JB-SelfCompany/yggstack-gui/internal/logger"
	"github.com/JB-SelfCompany/yggstack-gui/internal/yggdrasil/netstack"
)

// MappingType represents the type of port mapping
type MappingType string

const (
	MappingLocalTCP  MappingType = "local-tcp"
	MappingRemoteTCP MappingType = "remote-tcp"
	MappingLocalUDP  MappingType = "local-udp"
	MappingRemoteUDP MappingType = "remote-udp"
)

// PortMapping represents a port forwarding rule
type PortMapping struct {
	ID       string      `json:"id"`
	Type     MappingType `json:"type"`
	Source   string      `json:"source"` // e.g., "127.0.0.1:8080"
	Target   string      `json:"target"` // e.g., "[200:...]:80"
	Enabled  bool        `json:"enabled"`
	Active   bool        `json:"active"` // Currently forwarding
	BytesIn  uint64      `json:"bytesIn"`
	BytesOut uint64      `json:"bytesOut"`
}

// MappingStats contains statistics for a port mapping
type MappingStats struct {
	ID                string `json:"id"`
	ActiveConnections int64  `json:"activeConnections"`
	TotalConnections  uint64 `json:"totalConnections"`
	BytesIn           uint64 `json:"bytesIn"`
	BytesOut          uint64 `json:"bytesOut"`
}

// portForwarder handles a single port forwarding instance
type portForwarder struct {
	mapping           PortMapping
	listener          net.Listener
	udpConn           net.PacketConn
	ctx               context.Context
	cancel            context.CancelFunc
	activeConnections int64
	totalConnections  uint64
	bytesIn           uint64
	bytesOut          uint64
	logger            *logger.Logger
}

// MappingManager manages port forwarding rules
type MappingManager struct {
	mu         sync.RWMutex
	service    *Service
	forwarders map[string]*portForwarder
	logger     *logger.Logger
	idCounter  uint64
}

// NewMappingManager creates a new mapping manager
func NewMappingManager(service *Service, log *logger.Logger) *MappingManager {
	return &MappingManager{
		service:    service,
		forwarders: make(map[string]*portForwarder),
		logger:     log,
	}
}

// AddMapping adds a new port mapping
func (mm *MappingManager) AddMapping(mapping PortMapping) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	// Validate mapping
	if err := mm.validateMapping(&mapping); err != nil {
		return err
	}

	// Generate ID if not provided
	if mapping.ID == "" {
		mm.idCounter++
		mapping.ID = fmt.Sprintf("mapping-%d", mm.idCounter)
	}

	// Check for duplicate ID
	if _, exists := mm.forwarders[mapping.ID]; exists {
		return fmt.Errorf("mapping with ID %s already exists", mapping.ID)
	}

	mm.logger.Info("Adding port mapping",
		"id", mapping.ID,
		"type", mapping.Type,
		"source", mapping.Source,
		"target", mapping.Target,
	)

	// Create forwarder
	fwd := &portForwarder{
		mapping: mapping,
		logger:  mm.logger,
	}

	mm.forwarders[mapping.ID] = fwd

	// Start if enabled and service is running
	if mapping.Enabled && mm.service.IsRunning() {
		if err := mm.startForwarder(fwd); err != nil {
			mm.logger.Warn("Failed to start mapping", "id", mapping.ID, "error", err)
		}
	}

	return nil
}

// RemoveMapping removes a port mapping
func (mm *MappingManager) RemoveMapping(id string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	fwd, exists := mm.forwarders[id]
	if !exists {
		return fmt.Errorf("mapping not found: %s", id)
	}

	// Stop if running
	if fwd.cancel != nil {
		fwd.cancel()
	}
	if fwd.listener != nil {
		fwd.listener.Close()
	}
	if fwd.udpConn != nil {
		fwd.udpConn.Close()
	}

	delete(mm.forwarders, id)

	mm.logger.Info("Removed port mapping", "id", id)
	return nil
}

// EnableMapping enables a port mapping
func (mm *MappingManager) EnableMapping(id string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	fwd, exists := mm.forwarders[id]
	if !exists {
		return fmt.Errorf("mapping not found: %s", id)
	}

	if fwd.mapping.Active {
		return nil // Already running
	}

	fwd.mapping.Enabled = true

	if mm.service.IsRunning() {
		return mm.startForwarder(fwd)
	}

	return nil
}

// DisableMapping disables a port mapping
func (mm *MappingManager) DisableMapping(id string) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	fwd, exists := mm.forwarders[id]
	if !exists {
		return fmt.Errorf("mapping not found: %s", id)
	}

	fwd.mapping.Enabled = false
	return mm.stopForwarder(fwd)
}

// GetMappings returns all port mappings
func (mm *MappingManager) GetMappings() []PortMapping {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	mappings := make([]PortMapping, 0, len(mm.forwarders))
	for _, fwd := range mm.forwarders {
		mapping := fwd.mapping
		mapping.BytesIn = atomic.LoadUint64(&fwd.bytesIn)
		mapping.BytesOut = atomic.LoadUint64(&fwd.bytesOut)
		mappings = append(mappings, mapping)
	}

	return mappings
}

// GetMapping returns a specific port mapping
func (mm *MappingManager) GetMapping(id string) (*PortMapping, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	fwd, exists := mm.forwarders[id]
	if !exists {
		return nil, fmt.Errorf("mapping not found: %s", id)
	}

	mapping := fwd.mapping
	mapping.BytesIn = atomic.LoadUint64(&fwd.bytesIn)
	mapping.BytesOut = atomic.LoadUint64(&fwd.bytesOut)

	return &mapping, nil
}

// GetMappingStats returns statistics for a mapping
func (mm *MappingManager) GetMappingStats(id string) (*MappingStats, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	fwd, exists := mm.forwarders[id]
	if !exists {
		return nil, fmt.Errorf("mapping not found: %s", id)
	}

	return &MappingStats{
		ID:                id,
		ActiveConnections: atomic.LoadInt64(&fwd.activeConnections),
		TotalConnections:  atomic.LoadUint64(&fwd.totalConnections),
		BytesIn:           atomic.LoadUint64(&fwd.bytesIn),
		BytesOut:          atomic.LoadUint64(&fwd.bytesOut),
	}, nil
}

// validateMapping validates a port mapping configuration
func (mm *MappingManager) validateMapping(mapping *PortMapping) error {
	if mapping.Source == "" {
		return fmt.Errorf("source address is required")
	}

	if mapping.Target == "" {
		return fmt.Errorf("target address is required")
	}

	switch mapping.Type {
	case MappingLocalTCP, MappingRemoteTCP, MappingLocalUDP, MappingRemoteUDP:
		// Valid
	default:
		return fmt.Errorf("invalid mapping type: %s", mapping.Type)
	}

	return nil
}

// startForwarder starts a port forwarder
func (mm *MappingManager) startForwarder(fwd *portForwarder) error {
	if fwd.mapping.Active {
		return nil
	}

	ns := mm.service.GetNetstack()
	if ns == nil {
		return fmt.Errorf("netstack not available")
	}

	switch fwd.mapping.Type {
	case MappingLocalTCP:
		return mm.startLocalTCP(fwd, ns)
	case MappingRemoteTCP:
		return mm.startRemoteTCP(fwd, ns)
	case MappingLocalUDP:
		return mm.startLocalUDP(fwd, ns)
	case MappingRemoteUDP:
		return mm.startRemoteUDP(fwd, ns)
	default:
		return fmt.Errorf("unsupported mapping type: %s", fwd.mapping.Type)
	}
}

// stopForwarder stops a port forwarder
func (mm *MappingManager) stopForwarder(fwd *portForwarder) error {
	if !fwd.mapping.Active {
		return nil
	}

	if fwd.cancel != nil {
		fwd.cancel()
	}

	if fwd.listener != nil {
		fwd.listener.Close()
		fwd.listener = nil
	}

	if fwd.udpConn != nil {
		fwd.udpConn.Close()
		fwd.udpConn = nil
	}

	fwd.mapping.Active = false
	return nil
}

// startLocalTCP starts a local TCP forwarder
// Local TCP: Listen locally, forward to Yggdrasil address
func (mm *MappingManager) startLocalTCP(fwd *portForwarder, ns *netstack.YggdrasilNetstack) error {
	listener, err := net.Listen("tcp", fwd.mapping.Source)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", fwd.mapping.Source, err)
	}

	fwd.listener = listener
	fwd.ctx, fwd.cancel = context.WithCancel(context.Background())
	fwd.mapping.Active = true

	go mm.acceptLocalTCP(fwd, ns)

	mm.logger.Info("Started local TCP forwarding",
		"source", fwd.mapping.Source,
		"target", fwd.mapping.Target,
	)

	return nil
}

// startRemoteTCP starts a remote TCP forwarder
// Remote TCP: Listen on Yggdrasil, forward to local address
func (mm *MappingManager) startRemoteTCP(fwd *portForwarder, ns *netstack.YggdrasilNetstack) error {
	// Parse the source address (Yggdrasil address)
	tcpAddr, err := net.ResolveTCPAddr("tcp", fwd.mapping.Source)
	if err != nil {
		return fmt.Errorf("failed to resolve source address %s: %w", fwd.mapping.Source, err)
	}

	listener, err := ns.ListenTCP(tcpAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on Yggdrasil %s: %w", fwd.mapping.Source, err)
	}

	fwd.listener = listener
	fwd.ctx, fwd.cancel = context.WithCancel(context.Background())
	fwd.mapping.Active = true

	go mm.acceptRemoteTCP(fwd)

	mm.logger.Info("Started remote TCP forwarding",
		"source", fwd.mapping.Source,
		"target", fwd.mapping.Target,
	)

	return nil
}

// startLocalUDP starts a local UDP forwarder
func (mm *MappingManager) startLocalUDP(fwd *portForwarder, ns *netstack.YggdrasilNetstack) error {
	udpAddr, err := net.ResolveUDPAddr("udp", fwd.mapping.Source)
	if err != nil {
		return fmt.Errorf("failed to resolve source address %s: %w", fwd.mapping.Source, err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", fwd.mapping.Source, err)
	}

	fwd.udpConn = conn
	fwd.ctx, fwd.cancel = context.WithCancel(context.Background())
	fwd.mapping.Active = true

	go mm.handleLocalUDP(fwd, ns)

	mm.logger.Info("Started local UDP forwarding",
		"source", fwd.mapping.Source,
		"target", fwd.mapping.Target,
	)

	return nil
}

// startRemoteUDP starts a remote UDP forwarder
func (mm *MappingManager) startRemoteUDP(fwd *portForwarder, ns *netstack.YggdrasilNetstack) error {
	udpAddr, err := net.ResolveUDPAddr("udp", fwd.mapping.Source)
	if err != nil {
		return fmt.Errorf("failed to resolve source address %s: %w", fwd.mapping.Source, err)
	}

	conn, err := ns.ListenUDP(udpAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on Yggdrasil %s: %w", fwd.mapping.Source, err)
	}

	fwd.udpConn = conn
	fwd.ctx, fwd.cancel = context.WithCancel(context.Background())
	fwd.mapping.Active = true

	go mm.handleRemoteUDP(fwd)

	mm.logger.Info("Started remote UDP forwarding",
		"source", fwd.mapping.Source,
		"target", fwd.mapping.Target,
	)

	return nil
}

// acceptLocalTCP accepts TCP connections on local listener and forwards to Yggdrasil
func (mm *MappingManager) acceptLocalTCP(fwd *portForwarder, ns *netstack.YggdrasilNetstack) {
	mtu := mm.service.GetMTU()

	for {
		conn, err := fwd.listener.Accept()
		if err != nil {
			select {
			case <-fwd.ctx.Done():
				return
			default:
				fwd.logger.Debug("Accept error", "error", err)
				continue
			}
		}

		atomic.AddInt64(&fwd.activeConnections, 1)
		atomic.AddUint64(&fwd.totalConnections, 1)

		go mm.handleLocalTCPConnection(fwd, conn, ns, mtu)
	}
}

// acceptRemoteTCP accepts TCP connections from Yggdrasil and forwards to local
func (mm *MappingManager) acceptRemoteTCP(fwd *portForwarder) {
	mtu := mm.service.GetMTU()

	for {
		conn, err := fwd.listener.Accept()
		if err != nil {
			select {
			case <-fwd.ctx.Done():
				return
			default:
				fwd.logger.Debug("Accept error", "error", err)
				continue
			}
		}

		atomic.AddInt64(&fwd.activeConnections, 1)
		atomic.AddUint64(&fwd.totalConnections, 1)

		go mm.handleRemoteTCPConnection(fwd, conn, mtu)
	}
}

// handleLocalTCPConnection handles a single local TCP connection
func (mm *MappingManager) handleLocalTCPConnection(fwd *portForwarder, conn net.Conn, ns *netstack.YggdrasilNetstack, mtu uint64) {
	defer func() {
		conn.Close()
		atomic.AddInt64(&fwd.activeConnections, -1)
	}()

	// Connect to target via Yggdrasil
	target, err := ns.DialContext(fwd.ctx, "tcp", fwd.mapping.Target)
	if err != nil {
		fwd.logger.Debug("Failed to connect to target",
			"target", fwd.mapping.Target,
			"error", err,
		)
		return
	}
	defer target.Close()

	// Proxy data
	proxyTCP(mtu, conn, target, &fwd.bytesIn, &fwd.bytesOut)
}

// handleRemoteTCPConnection handles a single remote TCP connection
func (mm *MappingManager) handleRemoteTCPConnection(fwd *portForwarder, conn net.Conn, mtu uint64) {
	defer func() {
		conn.Close()
		atomic.AddInt64(&fwd.activeConnections, -1)
	}()

	// Connect to local target
	target, err := net.Dial("tcp", fwd.mapping.Target)
	if err != nil {
		fwd.logger.Debug("Failed to connect to target",
			"target", fwd.mapping.Target,
			"error", err,
		)
		return
	}
	defer target.Close()

	// Proxy data
	proxyTCP(mtu, conn, target, &fwd.bytesIn, &fwd.bytesOut)
}

// handleLocalUDP handles local UDP forwarding
func (mm *MappingManager) handleLocalUDP(fwd *portForwarder, ns *netstack.YggdrasilNetstack) {
	mtu := mm.service.GetMTU()
	buf := make([]byte, mtu)

	// Map of client addresses to Yggdrasil connections
	sessions := sync.Map{}

	for {
		select {
		case <-fwd.ctx.Done():
			return
		default:
		}

		n, addr, err := fwd.udpConn.ReadFrom(buf)
		if err != nil {
			fwd.logger.Debug("UDP read error", "error", err)
			continue
		}

		atomic.AddUint64(&fwd.bytesIn, uint64(n))

		// Get or create session for this client
		sessionKey := addr.String()
		var yggConn net.Conn

		if existing, ok := sessions.Load(sessionKey); ok {
			yggConn = existing.(net.Conn)
		} else {
			// Create new connection to Yggdrasil target
			conn, err := ns.DialContext(fwd.ctx, "udp", fwd.mapping.Target)
			if err != nil {
				fwd.logger.Debug("Failed to dial Yggdrasil target", "error", err)
				continue
			}
			yggConn = conn
			sessions.Store(sessionKey, conn)

			// Start reverse proxy for this session
			go func(clientAddr net.Addr, conn net.Conn) {
				defer func() {
					conn.Close()
					sessions.Delete(sessionKey)
				}()

				respBuf := make([]byte, mtu)
				for {
					select {
					case <-fwd.ctx.Done():
						return
					default:
					}

					n, err := conn.Read(respBuf)
					if err != nil {
						return
					}

					atomic.AddUint64(&fwd.bytesOut, uint64(n))
					fwd.udpConn.WriteTo(respBuf[:n], clientAddr)
				}
			}(addr, yggConn)
		}

		// Forward packet to Yggdrasil
		yggConn.Write(buf[:n])
	}
}

// handleRemoteUDP handles remote UDP forwarding (from Yggdrasil to local)
func (mm *MappingManager) handleRemoteUDP(fwd *portForwarder) {
	mtu := mm.service.GetMTU()
	buf := make([]byte, mtu)

	// Map of client addresses to local connections
	sessions := sync.Map{}

	for {
		select {
		case <-fwd.ctx.Done():
			return
		default:
		}

		n, addr, err := fwd.udpConn.(net.PacketConn).ReadFrom(buf)
		if err != nil {
			fwd.logger.Debug("UDP read error", "error", err)
			continue
		}

		atomic.AddUint64(&fwd.bytesIn, uint64(n))

		// Get or create session for this client
		sessionKey := addr.String()
		var localConn *net.UDPConn

		if existing, ok := sessions.Load(sessionKey); ok {
			localConn = existing.(*net.UDPConn)
		} else {
			// Resolve target address
			targetAddr, err := net.ResolveUDPAddr("udp", fwd.mapping.Target)
			if err != nil {
				fwd.logger.Debug("Failed to resolve target", "error", err)
				continue
			}

			// Create new connection to local target
			conn, err := net.DialUDP("udp", nil, targetAddr)
			if err != nil {
				fwd.logger.Debug("Failed to dial local target", "error", err)
				continue
			}
			localConn = conn
			sessions.Store(sessionKey, conn)

			// Start reverse proxy for this session
			go func(clientAddr net.Addr, conn *net.UDPConn) {
				defer func() {
					conn.Close()
					sessions.Delete(sessionKey)
				}()

				respBuf := make([]byte, mtu)
				for {
					select {
					case <-fwd.ctx.Done():
						return
					default:
					}

					n, err := conn.Read(respBuf)
					if err != nil {
						return
					}

					atomic.AddUint64(&fwd.bytesOut, uint64(n))
					fwd.udpConn.(net.PacketConn).WriteTo(respBuf[:n], clientAddr)
				}
			}(addr, localConn)
		}

		// Forward packet to local target
		localConn.Write(buf[:n])
	}
}

// proxyTCP proxies data between two TCP connections
func proxyTCP(mtu uint64, c1, c2 net.Conn, bytesIn, bytesOut *uint64) {
	var wg sync.WaitGroup
	wg.Add(2)

	// c1 -> c2
	go func() {
		defer wg.Done()
		buf := make([]byte, mtu)
		for {
			n, err := c1.Read(buf)
			if n > 0 {
				atomic.AddUint64(bytesOut, uint64(n))
				c2.Write(buf[:n])
			}
			if err != nil {
				if err != io.EOF {
					// Connection closed or error
				}
				break
			}
		}
	}()

	// c2 -> c1
	go func() {
		defer wg.Done()
		buf := make([]byte, mtu)
		for {
			n, err := c2.Read(buf)
			if n > 0 {
				atomic.AddUint64(bytesIn, uint64(n))
				c1.Write(buf[:n])
			}
			if err != nil {
				if err != io.EOF {
					// Connection closed or error
				}
				break
			}
		}
	}()

	wg.Wait()
}

// StopAll stops all port forwarders
func (mm *MappingManager) StopAll() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	for _, fwd := range mm.forwarders {
		mm.stopForwarder(fwd)
	}
}

// StartAllEnabled starts all enabled mappings (called when service starts)
func (mm *MappingManager) StartAllEnabled() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	for _, fwd := range mm.forwarders {
		if fwd.mapping.Enabled && !fwd.mapping.Active {
			if err := mm.startForwarder(fwd); err != nil {
				mm.logger.Warn("Failed to start mapping", "id", fwd.mapping.ID, "error", err)
			}
		}
	}
}
