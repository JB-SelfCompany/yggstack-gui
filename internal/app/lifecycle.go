package app

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// LifecycleState represents the application lifecycle state
type LifecycleState int

const (
	StateInitializing LifecycleState = iota
	StateReady
	StateBackground
	StateForeground
	StateShuttingDown
	StateTerminated
)

func (s LifecycleState) String() string {
	switch s {
	case StateInitializing:
		return "initializing"
	case StateReady:
		return "ready"
	case StateBackground:
		return "background"
	case StateForeground:
		return "foreground"
	case StateShuttingDown:
		return "shutting_down"
	case StateTerminated:
		return "terminated"
	default:
		return "unknown"
	}
}

// LifecycleListener is a callback for lifecycle state changes
type LifecycleListener func(state LifecycleState, previousState LifecycleState)

// LifecycleManager manages the application lifecycle
type LifecycleManager struct {
	mu sync.RWMutex

	logger    *zap.Logger
	state     LifecycleState
	listeners []LifecycleListener

	// Settings
	minimizeToTray bool
	startMinimized bool

	// Callbacks
	onShutdown func()

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc

	// Shutdown timeout
	shutdownTimeout time.Duration
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager(logger *zap.Logger) *LifecycleManager {
	ctx, cancel := context.WithCancel(context.Background())

	lm := &LifecycleManager{
		logger:          logger.Named("lifecycle"),
		state:           StateInitializing,
		minimizeToTray:  true,
		startMinimized:  false,
		shutdownTimeout: 10 * time.Second,
		ctx:             ctx,
		cancel:          cancel,
	}

	// Setup signal handlers
	lm.setupSignalHandlers()

	return lm
}

// setupSignalHandlers sets up OS signal handlers for graceful shutdown
func (lm *LifecycleManager) setupSignalHandlers() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		lm.logger.Info("Received signal", zap.String("signal", sig.String()))
		lm.RequestShutdown()
	}()
}

// SetMinimizeToTray sets whether the app should minimize to tray on close
func (lm *LifecycleManager) SetMinimizeToTray(enabled bool) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.minimizeToTray = enabled
}

// SetStartMinimized sets whether the app should start minimized
func (lm *LifecycleManager) SetStartMinimized(enabled bool) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.startMinimized = enabled
}

// ShouldMinimizeToTray returns whether the app should minimize to tray
func (lm *LifecycleManager) ShouldMinimizeToTray() bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.minimizeToTray
}

// ShouldStartMinimized returns whether the app should start minimized
func (lm *LifecycleManager) ShouldStartMinimized() bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.startMinimized
}

// SetOnShutdown sets the shutdown callback
func (lm *LifecycleManager) SetOnShutdown(callback func()) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.onShutdown = callback
}

// AddListener adds a lifecycle state listener
func (lm *LifecycleManager) AddListener(listener LifecycleListener) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.listeners = append(lm.listeners, listener)
}

// GetState returns the current lifecycle state
func (lm *LifecycleManager) GetState() LifecycleState {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	return lm.state
}

// SetState transitions to a new state
func (lm *LifecycleManager) SetState(newState LifecycleState) {
	lm.mu.Lock()
	previousState := lm.state
	lm.state = newState
	listeners := make([]LifecycleListener, len(lm.listeners))
	copy(listeners, lm.listeners)
	lm.mu.Unlock()

	lm.logger.Info("Lifecycle state changed",
		zap.String("previous", previousState.String()),
		zap.String("current", newState.String()))

	// Notify listeners
	for _, listener := range listeners {
		go listener(newState, previousState)
	}
}

// Ready marks the application as ready
func (lm *LifecycleManager) Ready() {
	lm.SetState(StateReady)
	if lm.ShouldStartMinimized() {
		lm.SetState(StateBackground)
	} else {
		lm.SetState(StateForeground)
	}
}

// EnterBackground transitions to background state
func (lm *LifecycleManager) EnterBackground() {
	state := lm.GetState()
	if state == StateForeground || state == StateReady {
		lm.SetState(StateBackground)
	}
}

// EnterForeground transitions to foreground state
func (lm *LifecycleManager) EnterForeground() {
	state := lm.GetState()
	if state == StateBackground || state == StateReady {
		lm.SetState(StateForeground)
	}
}

// IsBackground returns whether the app is in background state
func (lm *LifecycleManager) IsBackground() bool {
	return lm.GetState() == StateBackground
}

// IsForeground returns whether the app is in foreground state
func (lm *LifecycleManager) IsForeground() bool {
	return lm.GetState() == StateForeground
}

// IsShuttingDown returns whether the app is shutting down
func (lm *LifecycleManager) IsShuttingDown() bool {
	state := lm.GetState()
	return state == StateShuttingDown || state == StateTerminated
}

// RequestShutdown initiates graceful shutdown
func (lm *LifecycleManager) RequestShutdown() {
	if lm.IsShuttingDown() {
		return
	}

	lm.logger.Info("Shutdown requested")
	lm.SetState(StateShuttingDown)

	// Cancel context to signal all goroutines
	lm.cancel()

	// Get shutdown callback
	lm.mu.RLock()
	callback := lm.onShutdown
	lm.mu.RUnlock()

	// Execute shutdown with timeout
	done := make(chan struct{})
	go func() {
		if callback != nil {
			callback()
		}
		close(done)
	}()

	select {
	case <-done:
		lm.logger.Info("Graceful shutdown completed")
	case <-time.After(lm.shutdownTimeout):
		lm.logger.Warn("Shutdown timeout exceeded, forcing exit")
	}

	lm.SetState(StateTerminated)
}

// Context returns the lifecycle context (cancelled on shutdown)
func (lm *LifecycleManager) Context() context.Context {
	return lm.ctx
}

// HandleWindowClose handles the window close event
// Returns true if the window should be hidden (minimize to tray)
// Returns false if the application should exit
func (lm *LifecycleManager) HandleWindowClose() bool {
	if lm.ShouldMinimizeToTray() && !lm.IsShuttingDown() {
		lm.logger.Debug("Minimizing to tray instead of closing")
		lm.EnterBackground()
		return true
	}

	lm.logger.Debug("Window close will exit application")
	return false
}

// HandleWindowShow handles when the window is shown
func (lm *LifecycleManager) HandleWindowShow() {
	if lm.IsBackground() {
		lm.EnterForeground()
	}
}

// HandleWindowHide handles when the window is hidden
func (lm *LifecycleManager) HandleWindowHide() {
	if lm.IsForeground() && lm.ShouldMinimizeToTray() {
		lm.EnterBackground()
	}
}

// SetShutdownTimeout sets the shutdown timeout
func (lm *LifecycleManager) SetShutdownTimeout(timeout time.Duration) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.shutdownTimeout = timeout
}
