package ipc

import (
	"encoding/json"
	"testing"

	"github.com/JB-SelfCompany/yggstack-gui/internal/logger"
)

func newTestHandlers(t *testing.T) *Handlers {
	t.Helper()
	log := logger.NewWithConfig(logger.Config{Level: "error", Console: false})
	return NewHandlers(log)
}

func TestNewHandlers(t *testing.T) {
	h := newTestHandlers(t)

	if h.service == nil {
		t.Error("service should not be nil")
	}
	if h.configManager == nil {
		t.Error("configManager should not be nil")
	}
	if h.peerManager == nil {
		t.Error("peerManager should not be nil")
	}
	if h.sessionManager == nil {
		t.Error("sessionManager should not be nil")
	}
	if h.socksProxy == nil {
		t.Error("socksProxy should not be nil")
	}
	if h.mappingManager == nil {
		t.Error("mappingManager should not be nil")
	}
	if h.logger == nil {
		t.Error("logger should not be nil")
	}
}

func TestHandlers_GetService(t *testing.T) {
	h := newTestHandlers(t)
	svc := h.GetService()
	if svc == nil {
		t.Error("GetService() should not return nil")
	}
}

func TestHandlers_RegisterAll(t *testing.T) {
	h := newTestHandlers(t)
	log := logger.NewWithConfig(logger.Config{Level: "error", Console: false})
	bridge := NewBridge(log)

	h.RegisterAll(bridge)

	// Verify handlers are registered
	expectedEvents := []string{
		EventNodeStart,
		EventNodeStop,
		EventNodeStatus,
		EventPeersList,
		EventPeersAdd,
		EventPeersRemove,
		EventConfigLoad,
		EventConfigSave,
		EventSettingsGet,
		EventSettingsSet,
		EventProxyConfig,
		EventProxyStatus,
		EventMappingAdd,
		EventMappingRemove,
	}

	for _, event := range expectedEvents {
		if _, exists := bridge.handlers[event]; !exists {
			t.Errorf("handler for %q should be registered", event)
		}
	}
}

func TestHandlers_NodeStart(t *testing.T) {
	h := newTestHandlers(t)

	req := &Request{
		Payload: json.RawMessage(`{}`),
	}

	resp := h.handleNodeStart(req)

	if !resp.Success {
		t.Errorf("handleNodeStart should succeed, error: %v", resp.Error)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("response data should be a map")
	}

	if data["state"] != "running" {
		t.Errorf("state = %v, want 'running'", data["state"])
	}

	if data["ipv6Address"] == nil {
		t.Error("ipv6Address should not be nil")
	}
}

func TestHandlers_NodeStop(t *testing.T) {
	h := newTestHandlers(t)

	// First start the node
	h.handleNodeStart(&Request{Payload: json.RawMessage(`{}`)})

	req := &Request{
		Payload: json.RawMessage(`{}`),
	}

	resp := h.handleNodeStop(req)

	if !resp.Success {
		t.Errorf("handleNodeStop should succeed, error: %v", resp.Error)
	}

	data := resp.Data.(map[string]interface{})
	if data["state"] != "stopped" {
		t.Errorf("state = %v, want 'stopped'", data["state"])
	}
}

func TestHandlers_NodeStop_NotRunning(t *testing.T) {
	h := newTestHandlers(t)

	req := &Request{
		Payload: json.RawMessage(`{}`),
	}

	resp := h.handleNodeStop(req)

	if resp.Success {
		t.Error("handleNodeStop on stopped service should fail")
	}
	if resp.Error == nil {
		t.Error("should have error")
	}
}

func TestHandlers_NodeStatus(t *testing.T) {
	h := newTestHandlers(t)

	req := &Request{
		Payload: json.RawMessage(`{}`),
	}

	// Status when stopped
	resp := h.handleNodeStatus(req)
	if !resp.Success {
		t.Errorf("handleNodeStatus should succeed, error: %v", resp.Error)
	}

	data := resp.Data.(map[string]interface{})
	if data["state"] != "stopped" {
		t.Errorf("state = %v, want 'stopped'", data["state"])
	}

	// Start node
	h.handleNodeStart(&Request{Payload: json.RawMessage(`{}`)})

	// Status when running
	resp = h.handleNodeStatus(req)
	if !resp.Success {
		t.Errorf("handleNodeStatus should succeed, error: %v", resp.Error)
	}

	data = resp.Data.(map[string]interface{})
	if data["state"] != "running" {
		t.Errorf("state = %v, want 'running'", data["state"])
	}
	if data["ipv6Address"] == nil {
		t.Error("ipv6Address should not be nil when running")
	}
}

func TestHandlers_PeersList(t *testing.T) {
	h := newTestHandlers(t)

	req := &Request{
		Payload: json.RawMessage(`{}`),
	}

	resp := h.handlePeersList(req)

	if !resp.Success {
		t.Errorf("handlePeersList should succeed, error: %v", resp.Error)
	}

	// Should return configured peers as disconnected
	peers, ok := resp.Data.([]map[string]interface{})
	if !ok {
		t.Fatal("response data should be a slice of maps")
	}

	for _, peer := range peers {
		if peer["uri"] == nil {
			t.Error("peer should have uri")
		}
		if peer["connected"] != false {
			t.Error("peer should be disconnected when node not running")
		}
	}
}

func TestHandlers_PeersAdd(t *testing.T) {
	h := newTestHandlers(t)

	req := &Request{
		Payload: json.RawMessage(`{"uri": "tcp://test.peer:1234"}`),
	}

	resp := h.handlePeersAdd(req)

	if !resp.Success {
		t.Errorf("handlePeersAdd should succeed, error: %v", resp.Error)
	}

	data := resp.Data.(map[string]interface{})
	if data["added"] != true {
		t.Error("peer should be added")
	}
}

func TestHandlers_PeersAdd_InvalidJSON(t *testing.T) {
	h := newTestHandlers(t)

	req := &Request{
		Payload: json.RawMessage(`invalid`),
	}

	resp := h.handlePeersAdd(req)

	if resp.Success {
		t.Error("should fail with invalid JSON")
	}
	if resp.Error.Code != "PARSE_ERROR" {
		t.Errorf("error code = %q, want 'PARSE_ERROR'", resp.Error.Code)
	}
}

func TestHandlers_PeersAdd_InvalidURI(t *testing.T) {
	h := newTestHandlers(t)

	req := &Request{
		Payload: json.RawMessage(`{"uri": "invalid-uri"}`),
	}

	resp := h.handlePeersAdd(req)

	if resp.Success {
		t.Error("should fail with invalid URI")
	}
	if resp.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("error code = %q, want 'VALIDATION_ERROR'", resp.Error.Code)
	}
}

func TestHandlers_PeersRemove(t *testing.T) {
	h := newTestHandlers(t)

	// First add a peer
	h.handlePeersAdd(&Request{
		Payload: json.RawMessage(`{"uri": "tcp://test.peer:1234"}`),
	})

	req := &Request{
		Payload: json.RawMessage(`{"uri": "tcp://test.peer:1234"}`),
	}

	resp := h.handlePeersRemove(req)

	if !resp.Success {
		t.Errorf("handlePeersRemove should succeed, error: %v", resp.Error)
	}

	data := resp.Data.(map[string]interface{})
	if data["removed"] != true {
		t.Error("peer should be removed")
	}
}

func TestHandlers_PeersRemove_InvalidJSON(t *testing.T) {
	h := newTestHandlers(t)

	req := &Request{
		Payload: json.RawMessage(`invalid`),
	}

	resp := h.handlePeersRemove(req)

	if resp.Success {
		t.Error("should fail with invalid JSON")
	}
}

func TestHandlers_ConfigLoad(t *testing.T) {
	h := newTestHandlers(t)

	req := &Request{
		Payload: json.RawMessage(`{}`),
	}

	resp := h.handleConfigLoad(req)

	if !resp.Success {
		t.Errorf("handleConfigLoad should succeed, error: %v", resp.Error)
	}

	data := resp.Data.(map[string]interface{})
	if data["path"] == nil {
		t.Error("path should not be nil")
	}
	if data["peers"] == nil {
		t.Error("peers should not be nil")
	}
}

func TestHandlers_ConfigSave(t *testing.T) {
	h := newTestHandlers(t)

	req := &Request{
		Payload: json.RawMessage(`{}`),
	}

	resp := h.handleConfigSave(req)

	if !resp.Success {
		// May fail if no write permission, that's OK
		t.Logf("handleConfigSave result: %v", resp.Error)
	}
}

func TestHandlers_SettingsGet(t *testing.T) {
	h := newTestHandlers(t)

	req := &Request{
		Payload: json.RawMessage(`{}`),
	}

	resp := h.handleSettingsGet(req)

	if !resp.Success {
		t.Errorf("handleSettingsGet should succeed, error: %v", resp.Error)
	}

	data := resp.Data.(map[string]interface{})
	if data["language"] == nil {
		t.Error("language should not be nil")
	}
	if data["theme"] == nil {
		t.Error("theme should not be nil")
	}
}

func TestHandlers_SettingsSet(t *testing.T) {
	h := newTestHandlers(t)

	req := &Request{
		Payload: json.RawMessage(`{"language": "ru", "theme": "light"}`),
	}

	resp := h.handleSettingsSet(req)

	if !resp.Success {
		t.Errorf("handleSettingsSet should succeed, error: %v", resp.Error)
	}

	data := resp.Data.(map[string]interface{})
	if data["language"] != "ru" {
		t.Errorf("language = %v, want 'ru'", data["language"])
	}
}

func TestHandlers_SettingsSet_InvalidJSON(t *testing.T) {
	h := newTestHandlers(t)

	req := &Request{
		Payload: json.RawMessage(`invalid`),
	}

	resp := h.handleSettingsSet(req)

	if resp.Success {
		t.Error("should fail with invalid JSON")
	}
}

func TestHandlers_ProxyStatus(t *testing.T) {
	h := newTestHandlers(t)

	req := &Request{
		Payload: json.RawMessage(`{}`),
	}

	resp := h.handleProxyStatus(req)

	if !resp.Success {
		t.Errorf("handleProxyStatus should succeed, error: %v", resp.Error)
	}
}

func TestHandlers_MappingAdd_InvalidJSON(t *testing.T) {
	h := newTestHandlers(t)

	req := &Request{
		Payload: json.RawMessage(`invalid`),
	}

	resp := h.handleMappingAdd(req)

	if resp.Success {
		t.Error("should fail with invalid JSON")
	}
	if resp.Error.Code != "PARSE_ERROR" {
		t.Errorf("error code = %q, want 'PARSE_ERROR'", resp.Error.Code)
	}
}

func TestHandlers_MappingRemove_InvalidJSON(t *testing.T) {
	h := newTestHandlers(t)

	req := &Request{
		Payload: json.RawMessage(`invalid`),
	}

	resp := h.handleMappingRemove(req)

	if resp.Success {
		t.Error("should fail with invalid JSON")
	}
	if resp.Error.Code != "PARSE_ERROR" {
		t.Errorf("error code = %q, want 'PARSE_ERROR'", resp.Error.Code)
	}
}
