package yggdrasil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/JB-SelfCompany/yggstack-gui/internal/logger"
)

func newTestConfigManager(t *testing.T) *ConfigManager {
	t.Helper()
	log := logger.NewWithConfig(logger.Config{Level: "error", Console: false})
	cm := NewConfigManager(log)
	// Use temp directory for tests
	cm.path = filepath.Join(t.TempDir(), "yggdrasil.json")
	return cm
}

func TestDefaultPeers(t *testing.T) {
	peers := DefaultPeers()
	if len(peers) == 0 {
		t.Error("DefaultPeers() should not be empty")
	}

	// Check that all peers have valid URI format
	for _, peer := range peers {
		if peer == "" {
			t.Error("peer URI should not be empty")
		}
	}
}

func TestNewConfigManager(t *testing.T) {
	cm := newTestConfigManager(t)

	if cm.config == nil {
		t.Error("config should not be nil")
	}

	peers := cm.GetPeers()
	if len(peers) == 0 {
		t.Error("should have default peers")
	}
}

func TestConfigManager_Load_NotExists(t *testing.T) {
	cm := newTestConfigManager(t)

	err := cm.Load()
	if err != nil {
		t.Errorf("Load() for non-existent file should not return error, got %v", err)
	}
}

func TestConfigManager_Load(t *testing.T) {
	cm := newTestConfigManager(t)

	// Create test config file
	cfg := Config{
		Peers:  []string{"tcp://test1:1234", "tcp://test2:5678"},
		Listen: []string{"tcp://0.0.0.0:9999"},
	}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(cm.path, data, 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	err := cm.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	peers := cm.GetPeers()
	if len(peers) != 2 {
		t.Errorf("expected 2 peers, got %d", len(peers))
	}
}

func TestConfigManager_Load_InvalidJSON(t *testing.T) {
	cm := newTestConfigManager(t)

	// Write invalid JSON
	if err := os.WriteFile(cm.path, []byte("not json"), 0600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	err := cm.Load()
	if err == nil {
		t.Error("Load() should return error for invalid JSON")
	}
}

func TestConfigManager_Save(t *testing.T) {
	cm := newTestConfigManager(t)

	cm.SetPeers([]string{"tcp://saved:1234"})

	err := cm.Save()
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file was created
	data, err := os.ReadFile(cm.path)
	if err != nil {
		t.Fatalf("failed to read saved config: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("failed to parse saved config: %v", err)
	}

	if len(cfg.Peers) != 1 || cfg.Peers[0] != "tcp://saved:1234" {
		t.Errorf("saved config peers = %v, want [tcp://saved:1234]", cfg.Peers)
	}
}

func TestConfigManager_Generate(t *testing.T) {
	cm := newTestConfigManager(t)

	cm.SetPeers([]string{"tcp://old:1234"})

	err := cm.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Should have default peers after generate
	peers := cm.GetPeers()
	defaults := DefaultPeers()
	if len(peers) != len(defaults) {
		t.Errorf("peers count = %d, want %d", len(peers), len(defaults))
	}
}

func TestConfigManager_GetPeers(t *testing.T) {
	cm := newTestConfigManager(t)

	peers := cm.GetPeers()
	if peers == nil {
		t.Error("GetPeers() should not return nil")
	}

	// Verify it returns a copy
	peers[0] = "modified"
	original := cm.GetPeers()
	if original[0] == "modified" {
		t.Error("GetPeers() should return a copy")
	}
}

func TestConfigManager_SetPeers(t *testing.T) {
	cm := newTestConfigManager(t)

	newPeers := []string{"tcp://peer1:1", "tcp://peer2:2"}
	cm.SetPeers(newPeers)

	peers := cm.GetPeers()
	if len(peers) != 2 {
		t.Errorf("peers count = %d, want 2", len(peers))
	}
}

func TestConfigManager_AddPeer(t *testing.T) {
	cm := newTestConfigManager(t)

	initialCount := len(cm.GetPeers())

	err := cm.AddPeer("tcp://newpeer:1234")
	if err != nil {
		t.Fatalf("AddPeer() error = %v", err)
	}

	peers := cm.GetPeers()
	if len(peers) != initialCount+1 {
		t.Errorf("peers count = %d, want %d", len(peers), initialCount+1)
	}
}

func TestConfigManager_AddPeer_Duplicate(t *testing.T) {
	cm := newTestConfigManager(t)

	cm.SetPeers([]string{"tcp://existing:1234"})

	err := cm.AddPeer("tcp://existing:1234")
	if err != nil {
		t.Fatalf("AddPeer() error = %v", err)
	}

	// Should not add duplicate
	peers := cm.GetPeers()
	if len(peers) != 1 {
		t.Errorf("peers count = %d, want 1 (no duplicate)", len(peers))
	}
}

func TestConfigManager_RemovePeer(t *testing.T) {
	cm := newTestConfigManager(t)

	cm.SetPeers([]string{"tcp://peer1:1", "tcp://peer2:2", "tcp://peer3:3"})

	err := cm.RemovePeer("tcp://peer2:2")
	if err != nil {
		t.Fatalf("RemovePeer() error = %v", err)
	}

	peers := cm.GetPeers()
	if len(peers) != 2 {
		t.Errorf("peers count = %d, want 2", len(peers))
	}

	for _, p := range peers {
		if p == "tcp://peer2:2" {
			t.Error("peer2 should have been removed")
		}
	}
}

func TestConfigManager_RemovePeer_NotExists(t *testing.T) {
	cm := newTestConfigManager(t)

	initialCount := len(cm.GetPeers())

	err := cm.RemovePeer("tcp://nonexistent:1234")
	if err != nil {
		t.Fatalf("RemovePeer() for non-existent peer should not error: %v", err)
	}

	// Count should be unchanged
	peers := cm.GetPeers()
	if len(peers) != initialCount {
		t.Errorf("peers count changed unexpectedly")
	}
}

func TestConfigManager_Multicast(t *testing.T) {
	cm := newTestConfigManager(t)

	// Initially disabled
	if cm.IsMulticastEnabled() {
		t.Error("multicast should be disabled by default")
	}

	cm.SetMulticastEnabled(true)
	if !cm.IsMulticastEnabled() {
		t.Error("multicast should be enabled")
	}

	cm.SetMulticastEnabled(false)
	if cm.IsMulticastEnabled() {
		t.Error("multicast should be disabled")
	}
}

func TestConfigManager_GetConfigInfo(t *testing.T) {
	cm := newTestConfigManager(t)

	cm.SetPeers([]string{"tcp://info:1234"})

	info := cm.GetConfigInfo()
	if info == nil {
		t.Fatal("GetConfigInfo() returned nil")
	}

	if info.ConfigPath != cm.path {
		t.Errorf("ConfigPath = %q, want %q", info.ConfigPath, cm.path)
	}

	if len(info.Peers) != 1 {
		t.Errorf("Peers count = %d, want 1", len(info.Peers))
	}
}

func TestConfigManager_GetConfig(t *testing.T) {
	cm := newTestConfigManager(t)

	cfg := cm.GetConfig()
	if cfg == nil {
		t.Error("GetConfig() should not return nil")
	}
}

func TestConfigManager_GetPath(t *testing.T) {
	cm := newTestConfigManager(t)

	path := cm.GetPath()
	if path == "" {
		t.Error("GetPath() should not return empty string")
	}
}

func TestConfigManager_ConcurrentAccess(t *testing.T) {
	cm := newTestConfigManager(t)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(3)

		// Concurrent reads
		go func() {
			defer wg.Done()
			_ = cm.GetPeers()
		}()

		go func() {
			defer wg.Done()
			_ = cm.GetConfigInfo()
		}()

		// Concurrent writes
		go func(idx int) {
			defer wg.Done()
			_ = cm.AddPeer("tcp://concurrent:" + string(rune(idx)))
		}(i)
	}
	wg.Wait()
}
