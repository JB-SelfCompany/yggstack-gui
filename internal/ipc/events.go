package ipc

import "time"

// Event names for IPC communication
const (
	// Application events
	EventAppVersion = "app:version"
	EventAppReady   = "app:ready"
	EventAppPing    = "app:ping"
	EventAppQuit    = "app:quit"

	// Node management events
	EventNodeStart  = "node:start"
	EventNodeStop   = "node:stop"
	EventNodeStatus = "node:status"

	// Push events from backend to frontend
	EventNodeStateChanged = "node:stateChanged"
	EventNodeError        = "node:error"

	// Peer management events
	EventPeersList   = "peers:list"
	EventPeersAdd    = "peers:add"
	EventPeersRemove = "peers:remove"

	// Push events for peers
	EventPeersUpdate      = "peers:update"
	EventPeerConnected    = "peer:connected"
	EventPeerDisconnected = "peer:disconnected"

	// Session events
	EventSessionsList = "sessions:list"
	EventSessionsStats = "sessions:stats"

	// Configuration events
	EventConfigLoad = "config:load"
	EventConfigSave = "config:save"

	// Settings events
	EventSettingsGet = "settings:get"
	EventSettingsSet = "settings:set"

	// Proxy events
	EventProxyConfig = "proxy:config"
	EventProxyStatus = "proxy:status"
	EventProxyStart  = "proxy:start"
	EventProxyStop   = "proxy:stop"

	// Mapping events
	EventMappingList   = "mapping:list"
	EventMappingAdd    = "mapping:add"
	EventMappingRemove = "mapping:remove"
	EventMappingEnable = "mapping:enable"
	EventMappingDisable = "mapping:disable"

	// State synchronization events
	EventStateChanged = "state:changed"
	EventStateSync    = "state:sync"
	EventStateRequest = "state:request"

	// Log events
	EventLogEntry = "log:entry"
	EventLogLevel = "log:level"
	EventLogList  = "log:list"
	EventLogClear = "log:clear"

	// Stats events
	EventStatsUpdate = "stats:update"
)

// NodeStatus represents the current node status
type NodeStatus struct {
	State       string `json:"state"` // "stopped", "starting", "running", "stopping"
	IPv6Address string `json:"ipv6Address,omitempty"`
	Subnet      string `json:"subnet,omitempty"`
	PublicKey   string `json:"publicKey,omitempty"`
	Coords      []uint64 `json:"coords,omitempty"`
	Uptime      int64  `json:"uptime,omitempty"` // seconds
	PeerCount   int    `json:"peerCount,omitempty"`
}

// NodeStateChange represents a node state change event
type NodeStateChange struct {
	PreviousState string      `json:"previousState"`
	CurrentState  string      `json:"currentState"`
	NodeInfo      *NodeStatus `json:"nodeInfo,omitempty"`
	Error         string      `json:"error,omitempty"`
	Timestamp     int64       `json:"timestamp"`
}

// PeerInfo represents information about a peer
type PeerInfo struct {
	URI       string  `json:"uri"`
	Address   string  `json:"address"`
	PublicKey string  `json:"publicKey"`
	Connected bool    `json:"connected"`
	Inbound   bool    `json:"inbound"`
	Latency   float64 `json:"latency,omitempty"` // ms
	RxBytes   int64   `json:"rxBytes,omitempty"`
	TxBytes   int64   `json:"txBytes,omitempty"`
	Uptime    int64   `json:"uptime,omitempty"`  // seconds
}

// PeerEvent represents a peer connection/disconnection event
type PeerEvent struct {
	Type      string    `json:"type"` // "connected", "disconnected"
	Peer      *PeerInfo `json:"peer"`
	Timestamp int64     `json:"timestamp"`
}

// SessionInfo represents information about a session
type SessionInfo struct {
	Address   string `json:"address"`
	PublicKey string `json:"publicKey"`
	RxBytes   int64  `json:"rxBytes"`
	TxBytes   int64  `json:"txBytes"`
	Uptime    int64  `json:"uptime"` // seconds
}

// AppSettings represents application settings
type AppSettings struct {
	Language       string `json:"language"`       // "en", "ru"
	Theme          string `json:"theme"`          // "light", "dark", "system"
	MinimizeToTray bool   `json:"minimizeToTray"`
	StartMinimized bool   `json:"startMinimized"`
	Autostart      bool   `json:"autostart"`
	LogLevel       string `json:"logLevel"` // "debug", "info", "warn", "error"
}

// ProxyConfig represents SOCKS5 proxy configuration
type ProxyConfig struct {
	Enabled       bool   `json:"enabled"`
	ListenAddress string `json:"listenAddress"`
	Nameserver    string `json:"nameserver,omitempty"`
}

// ProxyStatus represents SOCKS5 proxy status
type ProxyStatus struct {
	Enabled           bool   `json:"enabled"`
	ListenAddress     string `json:"listenAddress"`
	ActiveConnections int64  `json:"activeConnections"`
	TotalConnections  int64  `json:"totalConnections"`
	BytesIn           int64  `json:"bytesIn"`
	BytesOut          int64  `json:"bytesOut"`
}

// PortMapping represents a port forwarding mapping
type PortMapping struct {
	ID       string `json:"id"`
	Type     string `json:"type"`    // "local-tcp", "remote-tcp", "local-udp", "remote-udp"
	Source   string `json:"source"`  // e.g., "127.0.0.1:8080"
	Target   string `json:"target"`  // e.g., "[200:...]:80"
	Enabled  bool   `json:"enabled"`
	Active   bool   `json:"active"`
	BytesIn  int64  `json:"bytesIn,omitempty"`
	BytesOut int64  `json:"bytesOut,omitempty"`
}

// LogEntry represents a log entry
type LogEntry struct {
	Level     string    `json:"level"` // "debug", "info", "warn", "error"
	Message   string    `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// NetworkStats represents network statistics
type NetworkStats struct {
	TotalRxBytes   int64   `json:"totalRxBytes"`
	TotalTxBytes   int64   `json:"totalTxBytes"`
	RxBytesPerSec  float64 `json:"rxBytesPerSec"`
	TxBytesPerSec  float64 `json:"txBytesPerSec"`
	PeerCount      int     `json:"peerCount"`
	SessionCount   int     `json:"sessionCount"`
	Uptime         int64   `json:"uptime"` // seconds
	Timestamp      int64   `json:"timestamp"`
}

// StateChangeEvent represents a state change notification
type StateChangeEvent struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	Previous  interface{} `json:"previous,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// ErrorEvent represents an error notification
type ErrorEvent struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   string `json:"details,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// Request payloads

// AddPeerRequest is the payload for adding a peer
type AddPeerRequest struct {
	URI string `json:"uri"`
}

// RemovePeerRequest is the payload for removing a peer
type RemovePeerRequest struct {
	URI string `json:"uri"`
}

// AddMappingRequest is the payload for adding a port mapping
type AddMappingRequest struct {
	Type    string `json:"type"`
	Source  string `json:"source"`
	Target  string `json:"target"`
	Enabled bool   `json:"enabled"`
}

// RemoveMappingRequest is the payload for removing a port mapping
type RemoveMappingRequest struct {
	ID string `json:"id"`
}

// SetSettingsRequest is the payload for updating settings
type SetSettingsRequest struct {
	Settings AppSettings `json:"settings"`
}

// SetProxyConfigRequest is the payload for configuring proxy
type SetProxyConfigRequest struct {
	Config ProxyConfig `json:"config"`
}
