package yggdrasil

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/things-go/go-socks5"

	"github.com/JB-SelfCompany/yggstack-gui/internal/logger"
	"github.com/JB-SelfCompany/yggstack-gui/internal/yggdrasil/netstack"
)

// SOCKSConfig contains SOCKS5 proxy configuration
type SOCKSConfig struct {
	Enabled       bool   `json:"enabled"`
	ListenAddress string `json:"listenAddress"` // e.g., "127.0.0.1:1080"
	Nameserver    string `json:"nameserver"`    // Optional DNS resolver for Yggdrasil
}

// SOCKSStats contains SOCKS5 proxy statistics
type SOCKSStats struct {
	Enabled           bool   `json:"enabled"`
	ListenAddress     string `json:"listenAddress"`
	ActiveConnections int64  `json:"activeConnections"`
	TotalConnections  uint64 `json:"totalConnections"`
	BytesIn           uint64 `json:"bytesIn"`
	BytesOut          uint64 `json:"bytesOut"`
}

// SOCKSProxy manages a SOCKS5 proxy server that routes through Yggdrasil
type SOCKSProxy struct {
	mu                sync.RWMutex
	config            SOCKSConfig
	service           *Service
	server            *socks5.Server
	listener          net.Listener
	ctx               context.Context
	cancel            context.CancelFunc
	running           bool
	activeConnections int64
	totalConnections  uint64
	bytesIn           uint64
	bytesOut          uint64
	logger            *logger.Logger
}

// NewSOCKSProxy creates a new SOCKS5 proxy
func NewSOCKSProxy(service *Service, log *logger.Logger) *SOCKSProxy {
	return &SOCKSProxy{
		config: SOCKSConfig{
			Enabled:       false,
			ListenAddress: "127.0.0.1:1080",
		},
		service: service,
		logger:  log,
	}
}

// Start starts the SOCKS5 proxy server
func (sp *SOCKSProxy) Start(config SOCKSConfig) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if sp.running {
		return fmt.Errorf("SOCKS proxy already running")
	}

	if !sp.service.IsRunning() {
		return fmt.Errorf("Yggdrasil service not running")
	}

	ns := sp.service.GetNetstack()
	if ns == nil {
		return fmt.Errorf("netstack not available")
	}

	sp.config = config

	// Create SOCKS5 server options
	socksOptions := []socks5.Option{
		socks5.WithDial(sp.createDialer(ns)),
	}

	// Add DNS resolver if configured
	if config.Nameserver != "" {
		resolver := NewNameResolver(ns, config.Nameserver)
		socksOptions = append(socksOptions, socks5.WithResolver(resolver))
	} else {
		// Use default resolver that only supports .pk.ygg addresses
		resolver := NewNameResolver(ns, "")
		socksOptions = append(socksOptions, socks5.WithResolver(resolver))
	}

	// Create SOCKS5 server
	sp.server = socks5.NewServer(socksOptions...)

	// Create listener
	listener, err := net.Listen("tcp", config.ListenAddress)
	if err != nil {
		return fmt.Errorf("failed to start SOCKS listener: %w", err)
	}

	sp.listener = listener
	sp.ctx, sp.cancel = context.WithCancel(context.Background())
	sp.running = true
	sp.config.Enabled = true

	// Start accepting connections
	go func() {
		if err := sp.server.Serve(listener); err != nil {
			sp.logger.Debug("SOCKS5 server stopped", "error", err)
		}
	}()

	sp.logger.Info("SOCKS5 proxy started", "address", config.ListenAddress)
	return nil
}

// createDialer creates a dial function that routes through Yggdrasil netstack
func (sp *SOCKSProxy) createDialer(ns *netstack.YggdrasilNetstack) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		atomic.AddInt64(&sp.activeConnections, 1)
		atomic.AddUint64(&sp.totalConnections, 1)

		conn, err := ns.DialContext(ctx, network, addr)
		if err != nil {
			atomic.AddInt64(&sp.activeConnections, -1)
			return nil, err
		}

		// Wrap connection to track statistics
		return &trackedConn{
			Conn:   conn,
			proxy:  sp,
			closed: false,
		}, nil
	}
}

// trackedConn wraps a connection to track statistics
type trackedConn struct {
	net.Conn
	proxy  *SOCKSProxy
	closed bool
	mu     sync.Mutex
}

func (tc *trackedConn) Read(b []byte) (int, error) {
	n, err := tc.Conn.Read(b)
	if n > 0 {
		atomic.AddUint64(&tc.proxy.bytesIn, uint64(n))
	}
	return n, err
}

func (tc *trackedConn) Write(b []byte) (int, error) {
	n, err := tc.Conn.Write(b)
	if n > 0 {
		atomic.AddUint64(&tc.proxy.bytesOut, uint64(n))
	}
	return n, err
}

func (tc *trackedConn) Close() error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if !tc.closed {
		tc.closed = true
		atomic.AddInt64(&tc.proxy.activeConnections, -1)
	}
	return tc.Conn.Close()
}

// Stop stops the SOCKS5 proxy server
func (sp *SOCKSProxy) Stop() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if !sp.running {
		return fmt.Errorf("SOCKS proxy not running")
	}

	sp.cancel()

	if sp.listener != nil {
		sp.listener.Close()
		sp.listener = nil
	}

	sp.server = nil
	sp.running = false
	sp.config.Enabled = false

	sp.logger.Info("SOCKS5 proxy stopped")
	return nil
}

// GetStats returns current SOCKS proxy statistics
func (sp *SOCKSProxy) GetStats() *SOCKSStats {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	return &SOCKSStats{
		Enabled:           sp.running,
		ListenAddress:     sp.config.ListenAddress,
		ActiveConnections: atomic.LoadInt64(&sp.activeConnections),
		TotalConnections:  atomic.LoadUint64(&sp.totalConnections),
		BytesIn:           atomic.LoadUint64(&sp.bytesIn),
		BytesOut:          atomic.LoadUint64(&sp.bytesOut),
	}
}

// GetConfig returns the current SOCKS configuration
func (sp *SOCKSProxy) GetConfig() SOCKSConfig {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return sp.config
}

// IsRunning returns true if the SOCKS proxy is running
func (sp *SOCKSProxy) IsRunning() bool {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return sp.running
}

// SetConfig updates the SOCKS proxy configuration
// Note: Requires restart if proxy is running
func (sp *SOCKSProxy) SetConfig(config SOCKSConfig) error {
	sp.mu.Lock()
	wasRunning := sp.running
	sp.mu.Unlock()

	if wasRunning {
		// Stop with lock released to avoid deadlock
		if err := sp.Stop(); err != nil {
			return err
		}
	}

	sp.mu.Lock()
	sp.config = config
	sp.mu.Unlock()

	if wasRunning && config.Enabled {
		// Restart with new config
		return sp.Start(config)
	}

	return nil
}

// ResetStats resets the connection statistics
func (sp *SOCKSProxy) ResetStats() {
	atomic.StoreUint64(&sp.totalConnections, 0)
	atomic.StoreUint64(&sp.bytesIn, 0)
	atomic.StoreUint64(&sp.bytesOut, 0)
}
