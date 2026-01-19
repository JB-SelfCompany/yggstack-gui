package yggdrasil

import (
	"sync"
	"testing"
	"time"

	"github.com/JB-SelfCompany/yggstack-gui/internal/logger"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	log := logger.NewWithConfig(logger.Config{Level: "error", Console: false})
	return NewService(log)
}

func TestServiceState_String(t *testing.T) {
	tests := []struct {
		state    ServiceState
		expected string
	}{
		{StateStopped, "stopped"},
		{StateStarting, "starting"},
		{StateRunning, "running"},
		{StateStopping, "stopping"},
		{ServiceState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("ServiceState.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNewService(t *testing.T) {
	svc := newTestService(t)

	if svc.state != StateStopped {
		t.Errorf("initial state = %v, want %v", svc.state, StateStopped)
	}

	if svc.configManager == nil {
		t.Error("configManager should not be nil")
	}

	if svc.listeners == nil {
		t.Error("listeners should not be nil")
	}
}

func TestService_Start(t *testing.T) {
	svc := newTestService(t)

	err := svc.Start(nil)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if svc.GetState() != StateRunning {
		t.Errorf("state = %v, want %v", svc.GetState(), StateRunning)
	}

	info := svc.GetNodeInfo()
	if info == nil {
		t.Fatal("GetNodeInfo() returned nil")
	}

	if info.IPv6Address == "" {
		t.Error("IPv6Address should not be empty")
	}
}

func TestService_Start_AlreadyRunning(t *testing.T) {
	svc := newTestService(t)

	if err := svc.Start(nil); err != nil {
		t.Fatalf("first Start() error = %v", err)
	}

	err := svc.Start(nil)
	if err == nil {
		t.Error("second Start() should return error")
	}
}

func TestService_Stop(t *testing.T) {
	svc := newTestService(t)

	if err := svc.Start(nil); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	err := svc.Stop()
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if svc.GetState() != StateStopped {
		t.Errorf("state = %v, want %v", svc.GetState(), StateStopped)
	}

	if svc.GetNodeInfo() != nil {
		t.Error("GetNodeInfo() should return nil after stop")
	}
}

func TestService_Stop_NotRunning(t *testing.T) {
	svc := newTestService(t)

	err := svc.Stop()
	if err == nil {
		t.Error("Stop() on stopped service should return error")
	}
}

func TestService_Restart(t *testing.T) {
	svc := newTestService(t)

	if err := svc.Start(nil); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	err := svc.Restart()
	if err != nil {
		t.Fatalf("Restart() error = %v", err)
	}

	if svc.GetState() != StateRunning {
		t.Errorf("state after restart = %v, want %v", svc.GetState(), StateRunning)
	}
}

func TestService_IsRunning(t *testing.T) {
	svc := newTestService(t)

	if svc.IsRunning() {
		t.Error("IsRunning() should be false initially")
	}

	if err := svc.Start(nil); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !svc.IsRunning() {
		t.Error("IsRunning() should be true after start")
	}

	if err := svc.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if svc.IsRunning() {
		t.Error("IsRunning() should be false after stop")
	}
}

func TestService_GetUptime(t *testing.T) {
	svc := newTestService(t)

	if uptime := svc.GetUptime(); uptime != 0 {
		t.Errorf("GetUptime() on stopped service = %v, want 0", uptime)
	}

	if err := svc.Start(nil); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	uptime := svc.GetUptime()
	if uptime < 10*time.Millisecond {
		t.Errorf("GetUptime() = %v, expected at least 10ms", uptime)
	}
}

func TestService_StateListener(t *testing.T) {
	svc := newTestService(t)

	var mu sync.Mutex
	var states []ServiceState

	svc.AddStateListener(func(state ServiceState, info *NodeInfo) {
		mu.Lock()
		states = append(states, state)
		mu.Unlock()
	})

	if err := svc.Start(nil); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Wait for async listener notification
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(states) < 2 {
		t.Errorf("expected at least 2 state changes, got %d", len(states))
	}

	// Should have starting -> running
	hasStarting := false
	hasRunning := false
	for _, s := range states {
		if s == StateStarting {
			hasStarting = true
		}
		if s == StateRunning {
			hasRunning = true
		}
	}

	if !hasStarting {
		t.Error("should have received StateStarting")
	}
	if !hasRunning {
		t.Error("should have received StateRunning")
	}
}

func TestService_AddPeer(t *testing.T) {
	svc := newTestService(t)

	// Should fail when not running
	err := svc.AddPeer("tcp://test:1234")
	if err == nil {
		t.Error("AddPeer() on stopped service should return error")
	}

	if err := svc.Start(nil); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	err = svc.AddPeer("tcp://test:1234")
	if err != nil {
		t.Errorf("AddPeer() error = %v", err)
	}
}

func TestService_RemovePeer(t *testing.T) {
	svc := newTestService(t)

	// Should fail when not running
	err := svc.RemovePeer("tcp://test:1234")
	if err == nil {
		t.Error("RemovePeer() on stopped service should return error")
	}

	if err := svc.Start(nil); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	err = svc.RemovePeer("tcp://test:1234")
	if err != nil {
		t.Errorf("RemovePeer() error = %v", err)
	}
}

func TestService_ConfigManager(t *testing.T) {
	svc := newTestService(t)

	cm := svc.ConfigManager()
	if cm == nil {
		t.Error("ConfigManager() should not return nil")
	}
}

func TestService_ConcurrentAccess(t *testing.T) {
	svc := newTestService(t)

	if err := svc.Start(nil); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer svc.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = svc.GetState()
			_ = svc.GetNodeInfo()
			_ = svc.GetUptime()
			_ = svc.IsRunning()
		}()
	}
	wg.Wait()
}
