package ipc

import (
	"encoding/json"
	"time"

	"github.com/JB-SelfCompany/yggstack-gui/internal/config"
	"github.com/JB-SelfCompany/yggstack-gui/internal/logger"
	"github.com/JB-SelfCompany/yggstack-gui/internal/platform"
	"github.com/JB-SelfCompany/yggstack-gui/internal/yggdrasil"
)

// Handlers contains all IPC handlers with Yggdrasil integration
type Handlers struct {
	service        *yggdrasil.Service
	peerManager    *yggdrasil.PeerManager
	sessionManager *yggdrasil.SessionManager
	socksProxy     *yggdrasil.SOCKSProxy
	mappingManager *yggdrasil.MappingManager
	configStore    *config.Store
	logger         *logger.Logger
	bridge         *Bridge
}

// NewHandlers creates handlers with Yggdrasil service integration
func NewHandlers(log *logger.Logger) *Handlers {
	service := yggdrasil.NewService(log)

	return &Handlers{
		service:        service,
		peerManager:    yggdrasil.NewPeerManager(service),
		sessionManager: yggdrasil.NewSessionManager(service),
		socksProxy:     yggdrasil.NewSOCKSProxy(service, log),
		mappingManager: yggdrasil.NewMappingManager(service, log),
		logger:         log,
	}
}

// configManager returns the config manager from service
func (h *Handlers) configManager() *yggdrasil.ConfigManager {
	return h.service.ConfigManager()
}

// GetService returns the Yggdrasil service
func (h *Handlers) GetService() *yggdrasil.Service {
	return h.service
}

// GetSOCKSProxy returns the SOCKS proxy
func (h *Handlers) GetSOCKSProxy() *yggdrasil.SOCKSProxy {
	return h.socksProxy
}

// SetConfigStore sets the config store for settings persistence
func (h *Handlers) SetConfigStore(store *config.Store) {
	h.configStore = store
}

// RegisterAll registers all handlers with the bridge
func (h *Handlers) RegisterAll(bridge *Bridge) {
	h.bridge = bridge

	// Node management
	bridge.Register(EventNodeStart, h.handleNodeStart)
	bridge.Register(EventNodeStop, h.handleNodeStop)
	bridge.Register(EventNodeStatus, h.handleNodeStatus)

	// Peer management
	bridge.Register(EventPeersList, h.handlePeersList)
	bridge.Register(EventPeersAdd, h.handlePeersAdd)
	bridge.Register(EventPeersRemove, h.handlePeersRemove)

	// Configuration
	bridge.Register(EventConfigLoad, h.handleConfigLoad)
	bridge.Register(EventConfigSave, h.handleConfigSave)

	// Settings
	bridge.Register(EventSettingsGet, h.handleSettingsGet)
	bridge.Register(EventSettingsSet, h.handleSettingsSet)

	// Proxy
	bridge.Register(EventProxyConfig, h.handleProxyConfig)
	bridge.Register(EventProxyStatus, h.handleProxyStatus)

	// Mappings
	bridge.Register(EventMappingAdd, h.handleMappingAdd)
	bridge.Register(EventMappingRemove, h.handleMappingRemove)

	// Logs
	bridge.Register(EventLogList, h.handleLogList)
	bridge.Register(EventLogClear, h.handleLogClear)

	// Subscribe to service state changes to notify frontend
	h.setupStateChangeNotifier()

	h.logger.Info("Yggdrasil IPC handlers registered")
}

// setupStateChangeNotifier subscribes to service state changes and emits events to frontend
func (h *Handlers) setupStateChangeNotifier() {
	h.service.AddStateListener(func(state yggdrasil.ServiceState, info *yggdrasil.NodeInfo) {
		h.logger.Debug("Service state changed, notifying frontend", "state", state.String())

		// Build state change event
		data := &NodeStateChange{
			CurrentState: state.String(),
			Timestamp:    time.Now().UnixMilli(),
		}

		if info != nil {
			data.NodeInfo = &NodeStatus{
				State:       state.String(),
				IPv6Address: info.IPv6Address,
				Subnet:      info.Subnet,
				PublicKey:   info.PublicKey,
			}
		}

		// Emit to frontend via IPC bridge
		h.bridge.Emit(EventNodeStateChanged, data)
	})
}

// Node handlers

func (h *Handlers) handleNodeStart(req *Request) *Response {
	h.logger.Info("Starting Yggdrasil node")

	// Load or generate configuration
	if err := h.configManager().Load(); err != nil {
		h.logger.Error("Failed to load configuration", "error", err)
		return &Response{
			Success: false,
			Error: &Error{
				Code:    "CONFIG_ERROR",
				Message: err.Error(),
			},
		}
	}

	cfg := h.configManager().GetConfig()

	// Start the service
	if err := h.service.Start(cfg); err != nil {
		h.logger.Error("Failed to start node", "error", err)
		return &Response{
			Success: false,
			Error: &Error{
				Code:    "START_ERROR",
				Message: err.Error(),
			},
		}
	}

	// Auto-start SOCKS proxy with settings from config store
	if h.configStore != nil {
		appSettings := h.configStore.Get()
		socksConfig := yggdrasil.SOCKSConfig{
			Enabled:       true, // Always start proxy when node starts
			ListenAddress: appSettings.Proxy.ListenAddress,
			Nameserver:    appSettings.Proxy.Nameserver,
		}
		h.logger.Info("Starting SOCKS proxy", "address", socksConfig.ListenAddress)
		if err := h.socksProxy.Start(socksConfig); err != nil {
			h.logger.Warn("Failed to start SOCKS proxy", "error", err)
		} else {
			h.logger.Info("SOCKS proxy started", "address", socksConfig.ListenAddress)
		}
	}

	// Return node info
	info := h.service.GetNodeInfo()
	data := map[string]interface{}{
		"state": h.service.GetState().String(),
	}

	if info != nil {
		data["ipv6Address"] = info.IPv6Address
		data["subnet"] = info.Subnet
		data["publicKey"] = info.PublicKey
	}

	return &Response{
		Success: true,
		Data:    data,
	}
}

func (h *Handlers) handleNodeStop(req *Request) *Response {
	h.logger.Info("Stopping Yggdrasil node")

	// Stop SOCKS proxy first if running
	if h.socksProxy.IsRunning() {
		h.logger.Info("Stopping SOCKS proxy")
		if err := h.socksProxy.Stop(); err != nil {
			h.logger.Warn("Failed to stop SOCKS proxy", "error", err)
		}
	}

	if err := h.service.Stop(); err != nil {
		h.logger.Error("Failed to stop node", "error", err)
		return &Response{
			Success: false,
			Error: &Error{
				Code:    "STOP_ERROR",
				Message: err.Error(),
			},
		}
	}

	return &Response{
		Success: true,
		Data: map[string]interface{}{
			"state": h.service.GetState().String(),
		},
	}
}

func (h *Handlers) handleNodeStatus(req *Request) *Response {
	state := h.service.GetState()
	info := h.service.GetNodeInfo()

	data := map[string]interface{}{
		"state": state.String(),
	}

	if info != nil {
		data["ipv6Address"] = info.IPv6Address
		data["subnet"] = info.Subnet
		data["publicKey"] = info.PublicKey
		data["uptime"] = info.Uptime.Seconds()

		// Add stats: peerCount, sessionCount, traffic
		peerStats := h.peerManager.GetPeerStats()
		sessionStats := h.sessionManager.GetSessionStats()

		data["stats"] = map[string]interface{}{
			"peerCount":    peerStats.Total,
			"sessionCount": sessionStats.Total,
			"rxBytes":      peerStats.TotalRxBytes,
			"txBytes":      peerStats.TotalTxBytes,
		}
	}

	return &Response{
		Success: true,
		Data:    data,
	}
}

// Peer handlers

func (h *Handlers) handlePeersList(req *Request) *Response {
	peers, err := h.peerManager.GetPeers()
	if err != nil {
		// If not running, return configured peers instead
		configPeers := h.configManager().GetPeers()
		peerList := make([]map[string]interface{}, len(configPeers))
		for i, uri := range configPeers {
			peerList[i] = map[string]interface{}{
				"uri":       uri,
				"connected": false,
			}
		}
		return &Response{
			Success: true,
			Data:    peerList,
		}
	}

	return &Response{
		Success: true,
		Data:    peers,
	}
}

func (h *Handlers) handlePeersAdd(req *Request) *Response {
	var payload struct {
		URI string `json:"uri"`
	}

	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return &Response{
			Success: false,
			Error: &Error{
				Code:    "PARSE_ERROR",
				Message: "Failed to parse peer URI",
			},
		}
	}

	// Validate URI
	if err := yggdrasil.ValidatePeerURI(payload.URI); err != nil {
		return &Response{
			Success: false,
			Error: &Error{
				Code:    "VALIDATION_ERROR",
				Message: err.Error(),
			},
		}
	}

	// Add to config
	if err := h.configManager().AddPeer(payload.URI); err != nil {
		return &Response{
			Success: false,
			Error: &Error{
				Code:    "CONFIG_ERROR",
				Message: err.Error(),
			},
		}
	}

	// If running, add to live node
	if h.service.IsRunning() {
		if err := h.peerManager.AddPeer(payload.URI); err != nil {
			h.logger.Warn("Failed to add peer to running node", "uri", payload.URI, "error", err)
		}
	}

	// Save config
	if err := h.configManager().Save(); err != nil {
		h.logger.Warn("Failed to save config", "error", err)
	}

	return &Response{
		Success: true,
		Data: map[string]interface{}{
			"uri":   payload.URI,
			"added": true,
		},
	}
}

func (h *Handlers) handlePeersRemove(req *Request) *Response {
	var payload struct {
		URI string `json:"uri"`
	}

	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return &Response{
			Success: false,
			Error: &Error{
				Code:    "PARSE_ERROR",
				Message: "Failed to parse peer URI",
			},
		}
	}

	// Remove from config
	if err := h.configManager().RemovePeer(payload.URI); err != nil {
		return &Response{
			Success: false,
			Error: &Error{
				Code:    "CONFIG_ERROR",
				Message: err.Error(),
			},
		}
	}

	// If running, remove from live node
	if h.service.IsRunning() {
		if err := h.peerManager.RemovePeer(payload.URI); err != nil {
			h.logger.Warn("Failed to remove peer from running node", "uri", payload.URI, "error", err)
		}
	}

	// Save config
	if err := h.configManager().Save(); err != nil {
		h.logger.Warn("Failed to save config", "error", err)
	}

	return &Response{
		Success: true,
		Data: map[string]interface{}{
			"uri":     payload.URI,
			"removed": true,
		},
	}
}

// Config handlers

func (h *Handlers) handleConfigLoad(req *Request) *Response {
	if err := h.configManager().Load(); err != nil {
		return &Response{
			Success: false,
			Error: &Error{
				Code:    "LOAD_ERROR",
				Message: err.Error(),
			},
		}
	}

	cfg := h.configManager().GetConfig()

	return &Response{
		Success: true,
		Data: map[string]interface{}{
			"path":             h.configManager().GetPath(),
			"peers":            cfg.Peers,
			"multicastEnabled": len(cfg.MulticastInterfaces) > 0,
		},
	}
}

func (h *Handlers) handleConfigSave(req *Request) *Response {
	if err := h.configManager().Save(); err != nil {
		return &Response{
			Success: false,
			Error: &Error{
				Code:    "SAVE_ERROR",
				Message: err.Error(),
			},
		}
	}

	return &Response{
		Success: true,
		Data: map[string]interface{}{
			"saved": true,
			"path":  h.configManager().GetPath(),
		},
	}
}

// Settings handlers

func (h *Handlers) handleSettingsGet(req *Request) *Response {
	// Return default settings if config store not set
	if h.configStore == nil {
		return &Response{
			Success: true,
			Data: map[string]interface{}{
				"language":       "en",
				"theme":          "dark",
				"minimizeToTray": true,
				"startMinimized": false,
				"autostart":      false,
				"logLevel":       "info",
			},
		}
	}

	settings := h.configStore.Get()

	// Get actual autostart state from system (registry/desktop file)
	// This is important because user may have changed autostart outside the app
	autostartEnabled, err := platform.IsAutoStartEnabled()
	if err != nil {
		h.logger.Warn("Failed to check autostart status", "error", err)
		autostartEnabled = settings.App.Autostart // Fallback to saved value
	}

	return &Response{
		Success: true,
		Data: map[string]interface{}{
			"language":       settings.App.Language,
			"theme":          settings.App.Theme,
			"minimizeToTray": settings.App.MinimizeToTray,
			"startMinimized": settings.App.StartMinimized,
			"autostart":      autostartEnabled,
			"logLevel":       settings.App.LogLevel,
			"proxy": map[string]interface{}{
				"enabled":       settings.Proxy.Enabled,
				"listenAddress": settings.Proxy.ListenAddress,
				"nameserver":    settings.Proxy.Nameserver,
			},
			"node": map[string]interface{}{
				"autoConnect": settings.Node.AutoConnect,
			},
		},
	}
}

func (h *Handlers) handleSettingsSet(req *Request) *Response {
	var payload struct {
		Language       *string `json:"language,omitempty"`
		Theme          *string `json:"theme,omitempty"`
		MinimizeToTray *bool   `json:"minimizeToTray,omitempty"`
		StartMinimized *bool   `json:"startMinimized,omitempty"`
		Autostart      *bool   `json:"autostart,omitempty"`
		LogLevel       *string `json:"logLevel,omitempty"`
		Proxy          *struct {
			Enabled       *bool   `json:"enabled,omitempty"`
			ListenAddress *string `json:"listenAddress,omitempty"`
			Nameserver    *string `json:"nameserver,omitempty"`
		} `json:"proxy,omitempty"`
		Node *struct {
			AutoConnect *bool `json:"autoConnect,omitempty"`
		} `json:"node,omitempty"`
	}

	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return &Response{
			Success: false,
			Error: &Error{
				Code:    "PARSE_ERROR",
				Message: "Failed to parse settings",
			},
		}
	}

	// Update settings in config store
	if h.configStore != nil {
		h.configStore.Update(func(s *config.Settings) {
			if payload.Language != nil {
				s.App.Language = *payload.Language
			}
			if payload.Theme != nil {
				s.App.Theme = *payload.Theme
			}
			if payload.MinimizeToTray != nil {
				s.App.MinimizeToTray = *payload.MinimizeToTray
			}
			if payload.StartMinimized != nil {
				s.App.StartMinimized = *payload.StartMinimized
			}
			if payload.Autostart != nil {
				s.App.Autostart = *payload.Autostart
				// Apply autostart setting to OS
				if *payload.Autostart {
					if err := platform.EnableAutoStart(); err != nil {
						h.logger.Warn("Failed to enable autostart", "error", err)
					} else {
						h.logger.Info("Autostart enabled")
					}
				} else {
					if err := platform.DisableAutoStart(); err != nil {
						h.logger.Warn("Failed to disable autostart", "error", err)
					} else {
						h.logger.Info("Autostart disabled")
					}
				}
			}
			if payload.LogLevel != nil {
				s.App.LogLevel = *payload.LogLevel
			}
			if payload.Proxy != nil {
				if payload.Proxy.Enabled != nil {
					s.Proxy.Enabled = *payload.Proxy.Enabled
				}
				if payload.Proxy.ListenAddress != nil {
					s.Proxy.ListenAddress = *payload.Proxy.ListenAddress
				}
				if payload.Proxy.Nameserver != nil {
					s.Proxy.Nameserver = *payload.Proxy.Nameserver
				}
			}
			if payload.Node != nil {
				if payload.Node.AutoConnect != nil {
					s.Node.AutoConnect = *payload.Node.AutoConnect
				}
			}
		})

		// Validate and save
		settings := h.configStore.Get()
		settings.Validate()
		h.configStore.Set(&settings)

		if err := h.configStore.Save(); err != nil {
			h.logger.Warn("Failed to save settings", "error", err)
		}
	}

	h.logger.Info("Settings updated")

	// Return updated settings
	return h.handleSettingsGet(req)
}

// Proxy handlers

func (h *Handlers) handleProxyConfig(req *Request) *Response {
	var config yggdrasil.SOCKSConfig
	if err := json.Unmarshal(req.Payload, &config); err != nil {
		return &Response{
			Success: false,
			Error: &Error{
				Code:    "PARSE_ERROR",
				Message: "Failed to parse proxy config",
			},
		}
	}

	if config.Enabled {
		if err := h.socksProxy.Start(config); err != nil {
			return &Response{
				Success: false,
				Error: &Error{
					Code:    "PROXY_ERROR",
					Message: err.Error(),
				},
			}
		}
	} else {
		if err := h.socksProxy.Stop(); err != nil {
			return &Response{
				Success: false,
				Error: &Error{
					Code:    "PROXY_ERROR",
					Message: err.Error(),
				},
			}
		}
	}

	return &Response{
		Success: true,
		Data:    h.socksProxy.GetStats(),
	}
}

func (h *Handlers) handleProxyStatus(req *Request) *Response {
	return &Response{
		Success: true,
		Data:    h.socksProxy.GetStats(),
	}
}

// Mapping handlers

func (h *Handlers) handleMappingAdd(req *Request) *Response {
	var mapping yggdrasil.PortMapping
	if err := json.Unmarshal(req.Payload, &mapping); err != nil {
		return &Response{
			Success: false,
			Error: &Error{
				Code:    "PARSE_ERROR",
				Message: "Failed to parse mapping",
			},
		}
	}

	if err := h.mappingManager.AddMapping(mapping); err != nil {
		return &Response{
			Success: false,
			Error: &Error{
				Code:    "MAPPING_ERROR",
				Message: err.Error(),
			},
		}
	}

	return &Response{
		Success: true,
		Data: map[string]interface{}{
			"id":    mapping.ID,
			"added": true,
		},
	}
}

func (h *Handlers) handleMappingRemove(req *Request) *Response {
	var payload struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return &Response{
			Success: false,
			Error: &Error{
				Code:    "PARSE_ERROR",
				Message: "Failed to parse mapping ID",
			},
		}
	}

	if err := h.mappingManager.RemoveMapping(payload.ID); err != nil {
		return &Response{
			Success: false,
			Error: &Error{
				Code:    "MAPPING_ERROR",
				Message: err.Error(),
			},
		}
	}

	return &Response{
		Success: true,
		Data: map[string]interface{}{
			"id":      payload.ID,
			"removed": true,
		},
	}
}

// Log handlers

func (h *Handlers) handleLogList(req *Request) *Response {
	var payload struct {
		Since int64 `json:"since,omitempty"` // Timestamp to get logs since
		Limit int   `json:"limit,omitempty"` // Max number of logs to return
	}

	// Parse optional parameters
	if req.Payload != nil && len(req.Payload) > 0 {
		json.Unmarshal(req.Payload, &payload)
	}

	// Default limit
	if payload.Limit <= 0 {
		payload.Limit = 100
	}

	// Get logs from logger buffer
	var logs []logger.LogEntry
	if payload.Since > 0 {
		logs = h.logger.GetLogs(payload.Since, payload.Limit)
	} else {
		logs = h.logger.GetAllLogs()
		// Apply limit
		if len(logs) > payload.Limit {
			logs = logs[len(logs)-payload.Limit:]
		}
	}

	// Convert to response format
	entries := make([]map[string]interface{}, len(logs))
	for i, log := range logs {
		entries[i] = map[string]interface{}{
			"level":     log.Level,
			"message":   log.Message,
			"source":    log.Source,
			"timestamp": log.Timestamp,
		}
		if log.Fields != nil && len(log.Fields) > 0 {
			entries[i]["fields"] = log.Fields
		}
	}

	return &Response{
		Success: true,
		Data: map[string]interface{}{
			"logs":  entries,
			"count": len(entries),
		},
	}
}

func (h *Handlers) handleLogClear(req *Request) *Response {
	h.logger.ClearLogBuffer()

	return &Response{
		Success: true,
		Data: map[string]interface{}{
			"cleared": true,
		},
	}
}
