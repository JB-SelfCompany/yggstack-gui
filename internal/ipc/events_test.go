package ipc

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEventConstants(t *testing.T) {
	// Verify event constants are unique and non-empty
	events := []string{
		EventAppVersion,
		EventAppReady,
		EventAppPing,
		EventAppQuit,
		EventNodeStart,
		EventNodeStop,
		EventNodeStatus,
		EventNodeStateChanged,
		EventNodeError,
		EventPeersList,
		EventPeersAdd,
		EventPeersRemove,
		EventPeersUpdate,
		EventPeerConnected,
		EventPeerDisconnected,
		EventSessionsList,
		EventSessionsStats,
		EventConfigLoad,
		EventConfigSave,
		EventSettingsGet,
		EventSettingsSet,
		EventProxyConfig,
		EventProxyStatus,
		EventProxyStart,
		EventProxyStop,
		EventMappingList,
		EventMappingAdd,
		EventMappingRemove,
		EventMappingEnable,
		EventMappingDisable,
		EventStateChanged,
		EventStateSync,
		EventStateRequest,
		EventLogEntry,
		EventLogLevel,
		EventStatsUpdate,
	}

	seen := make(map[string]bool)
	for _, event := range events {
		if event == "" {
			t.Error("Event constant is empty")
		}
		if seen[event] {
			t.Errorf("Duplicate event constant: %s", event)
		}
		seen[event] = true
	}
}

func TestNodeStatusJSON(t *testing.T) {
	status := NodeStatus{
		State:       "running",
		IPv6Address: "200:1234:5678:9abc:def0:1234:5678:9abc",
		Subnet:      "300::/64",
		PublicKey:   "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Coords:      []uint64{1, 2, 3},
		Uptime:      3600,
		PeerCount:   5,
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal NodeStatus: %v", err)
	}

	var decoded NodeStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal NodeStatus: %v", err)
	}

	if decoded.State != status.State {
		t.Errorf("State = %q, want %q", decoded.State, status.State)
	}
	if decoded.IPv6Address != status.IPv6Address {
		t.Errorf("IPv6Address = %q, want %q", decoded.IPv6Address, status.IPv6Address)
	}
	if decoded.PeerCount != status.PeerCount {
		t.Errorf("PeerCount = %d, want %d", decoded.PeerCount, status.PeerCount)
	}
}

func TestNodeStateChangeJSON(t *testing.T) {
	change := NodeStateChange{
		PreviousState: "stopped",
		CurrentState:  "running",
		NodeInfo: &NodeStatus{
			State:       "running",
			IPv6Address: "200::",
		},
		Error:     "",
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(change)
	if err != nil {
		t.Fatalf("Failed to marshal NodeStateChange: %v", err)
	}

	var decoded NodeStateChange
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal NodeStateChange: %v", err)
	}

	if decoded.PreviousState != change.PreviousState {
		t.Errorf("PreviousState = %q, want %q", decoded.PreviousState, change.PreviousState)
	}
	if decoded.CurrentState != change.CurrentState {
		t.Errorf("CurrentState = %q, want %q", decoded.CurrentState, change.CurrentState)
	}
	if decoded.NodeInfo == nil {
		t.Fatal("NodeInfo is nil after unmarshal")
	}
}

func TestPeerInfoJSON(t *testing.T) {
	peer := PeerInfo{
		URI:       "tls://example.com:443",
		Address:   "200:1234::1",
		PublicKey: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Connected: true,
		Inbound:   false,
		Latency:   15.5,
		RxBytes:   1024000,
		TxBytes:   512000,
		Uptime:    7200,
	}

	data, err := json.Marshal(peer)
	if err != nil {
		t.Fatalf("Failed to marshal PeerInfo: %v", err)
	}

	var decoded PeerInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal PeerInfo: %v", err)
	}

	if decoded.URI != peer.URI {
		t.Errorf("URI = %q, want %q", decoded.URI, peer.URI)
	}
	if decoded.Connected != peer.Connected {
		t.Errorf("Connected = %v, want %v", decoded.Connected, peer.Connected)
	}
	if decoded.Latency != peer.Latency {
		t.Errorf("Latency = %f, want %f", decoded.Latency, peer.Latency)
	}
}

func TestPeerEventJSON(t *testing.T) {
	event := PeerEvent{
		Type: "connected",
		Peer: &PeerInfo{
			URI:       "tcp://192.168.1.1:12345",
			Connected: true,
		},
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal PeerEvent: %v", err)
	}

	var decoded PeerEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal PeerEvent: %v", err)
	}

	if decoded.Type != event.Type {
		t.Errorf("Type = %q, want %q", decoded.Type, event.Type)
	}
	if decoded.Peer == nil {
		t.Fatal("Peer is nil after unmarshal")
	}
}

func TestSessionInfoJSON(t *testing.T) {
	session := SessionInfo{
		Address:   "200:abcd::1",
		PublicKey: "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		RxBytes:   2048000,
		TxBytes:   1024000,
		Uptime:    1800,
	}

	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("Failed to marshal SessionInfo: %v", err)
	}

	var decoded SessionInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal SessionInfo: %v", err)
	}

	if decoded.Address != session.Address {
		t.Errorf("Address = %q, want %q", decoded.Address, session.Address)
	}
	if decoded.RxBytes != session.RxBytes {
		t.Errorf("RxBytes = %d, want %d", decoded.RxBytes, session.RxBytes)
	}
}

func TestAppSettingsJSON(t *testing.T) {
	settings := AppSettings{
		Language:       "ru",
		Theme:          "light",
		MinimizeToTray: true,
		StartMinimized: false,
		Autostart:      true,
		LogLevel:       "debug",
	}

	data, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("Failed to marshal AppSettings: %v", err)
	}

	var decoded AppSettings
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal AppSettings: %v", err)
	}

	if decoded.Language != settings.Language {
		t.Errorf("Language = %q, want %q", decoded.Language, settings.Language)
	}
	if decoded.Theme != settings.Theme {
		t.Errorf("Theme = %q, want %q", decoded.Theme, settings.Theme)
	}
	if decoded.MinimizeToTray != settings.MinimizeToTray {
		t.Errorf("MinimizeToTray = %v, want %v", decoded.MinimizeToTray, settings.MinimizeToTray)
	}
}

func TestProxyConfigJSON(t *testing.T) {
	config := ProxyConfig{
		Enabled:       true,
		ListenAddress: "127.0.0.1:1080",
		Nameserver:    "8.8.8.8",
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal ProxyConfig: %v", err)
	}

	var decoded ProxyConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ProxyConfig: %v", err)
	}

	if decoded.Enabled != config.Enabled {
		t.Errorf("Enabled = %v, want %v", decoded.Enabled, config.Enabled)
	}
	if decoded.ListenAddress != config.ListenAddress {
		t.Errorf("ListenAddress = %q, want %q", decoded.ListenAddress, config.ListenAddress)
	}
}

func TestProxyStatusJSON(t *testing.T) {
	status := ProxyStatus{
		Enabled:           true,
		ListenAddress:     "0.0.0.0:9050",
		ActiveConnections: 10,
		TotalConnections:  1000,
		BytesIn:           1024 * 1024 * 100,
		BytesOut:          1024 * 1024 * 50,
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal ProxyStatus: %v", err)
	}

	var decoded ProxyStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ProxyStatus: %v", err)
	}

	if decoded.ActiveConnections != status.ActiveConnections {
		t.Errorf("ActiveConnections = %d, want %d", decoded.ActiveConnections, status.ActiveConnections)
	}
}

func TestPortMappingJSON(t *testing.T) {
	mapping := PortMapping{
		ID:       "mapping-1",
		Type:     "local-tcp",
		Source:   "127.0.0.1:8080",
		Target:   "[200::1]:80",
		Enabled:  true,
		Active:   true,
		BytesIn:  50000,
		BytesOut: 25000,
	}

	data, err := json.Marshal(mapping)
	if err != nil {
		t.Fatalf("Failed to marshal PortMapping: %v", err)
	}

	var decoded PortMapping
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal PortMapping: %v", err)
	}

	if decoded.ID != mapping.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, mapping.ID)
	}
	if decoded.Type != mapping.Type {
		t.Errorf("Type = %q, want %q", decoded.Type, mapping.Type)
	}
	if decoded.Active != mapping.Active {
		t.Errorf("Active = %v, want %v", decoded.Active, mapping.Active)
	}
}

func TestLogEntryJSON(t *testing.T) {
	entry := LogEntry{
		Level:   "info",
		Message: "Test log message",
		Fields: map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		},
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal LogEntry: %v", err)
	}

	var decoded LogEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal LogEntry: %v", err)
	}

	if decoded.Level != entry.Level {
		t.Errorf("Level = %q, want %q", decoded.Level, entry.Level)
	}
	if decoded.Message != entry.Message {
		t.Errorf("Message = %q, want %q", decoded.Message, entry.Message)
	}
}

func TestNetworkStatsJSON(t *testing.T) {
	stats := NetworkStats{
		TotalRxBytes:  1024 * 1024 * 1024,
		TotalTxBytes:  512 * 1024 * 1024,
		RxBytesPerSec: 1024 * 100,
		TxBytesPerSec: 1024 * 50,
		PeerCount:     10,
		SessionCount:  25,
		Uptime:        86400,
		Timestamp:     time.Now().Unix(),
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Failed to marshal NetworkStats: %v", err)
	}

	var decoded NetworkStats
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal NetworkStats: %v", err)
	}

	if decoded.PeerCount != stats.PeerCount {
		t.Errorf("PeerCount = %d, want %d", decoded.PeerCount, stats.PeerCount)
	}
	if decoded.SessionCount != stats.SessionCount {
		t.Errorf("SessionCount = %d, want %d", decoded.SessionCount, stats.SessionCount)
	}
}

func TestRequestPayloadsJSON(t *testing.T) {
	t.Run("AddPeerRequest", func(t *testing.T) {
		req := AddPeerRequest{URI: "tls://example.com:443"}
		data, _ := json.Marshal(req)
		var decoded AddPeerRequest
		json.Unmarshal(data, &decoded)
		if decoded.URI != req.URI {
			t.Errorf("URI = %q, want %q", decoded.URI, req.URI)
		}
	})

	t.Run("RemovePeerRequest", func(t *testing.T) {
		req := RemovePeerRequest{URI: "tcp://192.168.1.1:12345"}
		data, _ := json.Marshal(req)
		var decoded RemovePeerRequest
		json.Unmarshal(data, &decoded)
		if decoded.URI != req.URI {
			t.Errorf("URI = %q, want %q", decoded.URI, req.URI)
		}
	})

	t.Run("AddMappingRequest", func(t *testing.T) {
		req := AddMappingRequest{
			Type:    "local-tcp",
			Source:  "127.0.0.1:8080",
			Target:  "[200::1]:80",
			Enabled: true,
		}
		data, _ := json.Marshal(req)
		var decoded AddMappingRequest
		json.Unmarshal(data, &decoded)
		if decoded.Type != req.Type {
			t.Errorf("Type = %q, want %q", decoded.Type, req.Type)
		}
		if decoded.Enabled != req.Enabled {
			t.Errorf("Enabled = %v, want %v", decoded.Enabled, req.Enabled)
		}
	})

	t.Run("RemoveMappingRequest", func(t *testing.T) {
		req := RemoveMappingRequest{ID: "mapping-123"}
		data, _ := json.Marshal(req)
		var decoded RemoveMappingRequest
		json.Unmarshal(data, &decoded)
		if decoded.ID != req.ID {
			t.Errorf("ID = %q, want %q", decoded.ID, req.ID)
		}
	})
}

func TestStateChangeEventJSON(t *testing.T) {
	event := StateChangeEvent{
		Key:       "node.status",
		Value:     "running",
		Previous:  "stopped",
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal StateChangeEvent: %v", err)
	}

	var decoded StateChangeEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal StateChangeEvent: %v", err)
	}

	if decoded.Key != event.Key {
		t.Errorf("Key = %q, want %q", decoded.Key, event.Key)
	}
}

func TestErrorEventJSON(t *testing.T) {
	event := ErrorEvent{
		Code:      "CONNECTION_FAILED",
		Message:   "Failed to connect to peer",
		Details:   "Connection refused",
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal ErrorEvent: %v", err)
	}

	var decoded ErrorEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ErrorEvent: %v", err)
	}

	if decoded.Code != event.Code {
		t.Errorf("Code = %q, want %q", decoded.Code, event.Code)
	}
	if decoded.Details != event.Details {
		t.Errorf("Details = %q, want %q", decoded.Details, event.Details)
	}
}
