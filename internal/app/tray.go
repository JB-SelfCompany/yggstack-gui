package app

import (
	"sync"

	"github.com/energye/energy/v2/cef"
	"github.com/energye/energy/v2/pkgs/systray"
	"go.uber.org/zap"

	"github.com/JB-SelfCompany/yggstack-gui/internal/yggdrasil"
)

// TrayIcon represents different tray icon states
type TrayIcon int

const (
	TrayIconStopped TrayIcon = iota
	TrayIconRunning
	TrayIconConnecting
	TrayIconError
)

// TrayManager handles system tray functionality
type TrayManager struct {
	mu sync.RWMutex

	app     *Application
	service *yggdrasil.Service
	logger  *zap.Logger

	// Menu items
	mShow      *systray.MenuItem
	mStartStop *systray.MenuItem
	mSeparator *systray.MenuItem
	mQuit      *systray.MenuItem

	// State
	isRunning    bool
	currentIcon  TrayIcon
	isInitialized bool

	// Callbacks
	onShowWindow func()
	onQuit       func()

	// Icons (embedded)
	iconStopped    []byte
	iconRunning    []byte
	iconConnecting []byte

	// Keep references to prevent GC from collecting handlers
	clickHandler      func()
	dClickHandler     func()
	showHandler       func()
	startStopHandler  func()
	quitHandler       func()
}

// NewTrayManager creates a new tray manager instance
func NewTrayManager(app *Application, service *yggdrasil.Service, logger *zap.Logger) *TrayManager {
	return &TrayManager{
		app:         app,
		service:     service,
		logger:      logger.Named("tray"),
		currentIcon: TrayIcon(-1), // Invalid value to force first icon set
	}
}

// SetCallbacks sets the callback functions for tray actions
func (t *TrayManager) SetCallbacks(onShowWindow, onQuit func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onShowWindow = onShowWindow
	t.onQuit = onQuit
}

// SetIcons sets the tray icons for different states
func (t *TrayManager) SetIcons(stopped, running, connecting []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.iconStopped = stopped
	t.iconRunning = running
	t.iconConnecting = connecting
}

// Initialize sets up the system tray
func (t *TrayManager) Initialize() {
	t.logger.Info("Initializing system tray")

	// Run systray in a goroutine
	go systray.Run(t.onReady, t.onExit)
}

// onReady is called when the systray is ready
func (t *TrayManager) onReady() {
	t.logger.Debug("System tray ready")

	// Set initial icon and tooltip
	t.setIcon(TrayIconStopped)
	systray.SetTitle("Yggstack-GUI")
	systray.SetTooltip("Yggdrasil Network - Stopped")

	// Setup click handlers for the tray icon itself
	// Store references to prevent GC from collecting them
	t.mu.Lock()
	t.clickHandler = func() {
		t.logger.Debug("Tray icon clicked")
		t.mu.RLock()
		callback := t.onShowWindow
		t.mu.RUnlock()
		if callback != nil {
			cef.QueueAsyncCall(func(id int) {
				callback()
			})
		}
	}
	t.dClickHandler = func() {
		t.logger.Debug("Tray icon double-clicked")
		t.mu.RLock()
		callback := t.onShowWindow
		t.mu.RUnlock()
		if callback != nil {
			cef.QueueAsyncCall(func(id int) {
				callback()
			})
		}
	}
	t.mu.Unlock()

	systray.SetOnClick(t.clickHandler)
	systray.SetOnDClick(t.dClickHandler)

	// Create menu items
	t.mShow = systray.AddMenuItem("Show Window", "Show the main window")
	systray.AddSeparator()
	t.mStartStop = systray.AddMenuItem("Start Node", "Start/Stop Yggdrasil node")
	systray.AddSeparator()
	t.mQuit = systray.AddMenuItem("Exit", "Exit application")

	// Setup menu click handlers
	t.setupMenuClickHandlers()

	// Subscribe to service state changes
	if t.service != nil {
		t.service.AddStateListener(func(state yggdrasil.ServiceState, info *yggdrasil.NodeInfo) {
			t.handleStateChange(state, info)
		})
	}

	t.mu.Lock()
	t.isInitialized = true
	t.mu.Unlock()

	t.logger.Info("System tray initialized")
}

// onExit is called when the systray is about to exit
func (t *TrayManager) onExit() {
	t.logger.Info("System tray exiting")
}

// setupMenuClickHandlers sets up click handlers for menu items
func (t *TrayManager) setupMenuClickHandlers() {
	// Store handlers to prevent GC from collecting them
	t.mu.Lock()

	// Show window handler
	t.showHandler = func() {
		t.logger.Debug("Tray menu: Show clicked")
		t.mu.RLock()
		callback := t.onShowWindow
		t.mu.RUnlock()
		if callback != nil {
			// Execute on main thread
			cef.QueueAsyncCall(func(id int) {
				callback()
			})
		}
	}

	// Start/Stop handler
	t.startStopHandler = func() {
		t.logger.Debug("Tray menu: Start/Stop clicked")
		// Execute toggle on main thread to avoid potential race conditions
		cef.QueueAsyncCall(func(id int) {
			t.toggleNode()
		})
	}

	// Quit handler
	t.quitHandler = func() {
		t.logger.Debug("Tray menu: Quit clicked")
		t.mu.RLock()
		callback := t.onQuit
		t.mu.RUnlock()
		if callback != nil {
			// Execute on main thread
			cef.QueueAsyncCall(func(id int) {
				callback()
			})
		}
	}

	t.mu.Unlock()

	// Register handlers with menu items
	t.mShow.Click(t.showHandler)
	t.mStartStop.Click(t.startStopHandler)
	t.mQuit.Click(t.quitHandler)
}

// toggleNode starts or stops the Yggdrasil node
func (t *TrayManager) toggleNode() {
	if t.service == nil {
		t.logger.Warn("Service not available")
		return
	}

	state := t.service.GetState()
	switch state {
	case yggdrasil.StateStopped:
		t.logger.Info("Starting node from tray")
		// Update menu immediately
		t.mStartStop.SetTitle("Starting...")
		t.mStartStop.Disable()
		t.setIcon(TrayIconConnecting)

		go func() {
			// Load configuration before starting (same as handleNodeStart)
			if err := t.service.ConfigManager().Load(); err != nil {
				t.logger.Error("Failed to load configuration", zap.Error(err))
				cef.QueueAsyncCall(func(id int) {
					t.mStartStop.SetTitle("Start Node")
					t.mStartStop.Enable()
					t.setIcon(TrayIconError)
				})
				return
			}

			if err := t.service.Start(nil); err != nil {
				t.logger.Error("Failed to start node", zap.Error(err))
				cef.QueueAsyncCall(func(id int) {
					t.mStartStop.SetTitle("Start Node")
					t.mStartStop.Enable()
					t.setIcon(TrayIconError)
				})
			}
		}()

	case yggdrasil.StateRunning:
		t.logger.Info("Stopping node from tray")
		// Update menu immediately
		t.mStartStop.SetTitle("Stopping...")
		t.mStartStop.Disable()

		go func() {
			if err := t.service.Stop(); err != nil {
				t.logger.Error("Failed to stop node", zap.Error(err))
				cef.QueueAsyncCall(func(id int) {
					t.mStartStop.SetTitle("Stop Node")
					t.mStartStop.Enable()
				})
			}
		}()

	default:
		t.logger.Debug("Node is in transitional state, ignoring toggle")
	}
}

// handleStateChange updates the tray based on service state changes
func (t *TrayManager) handleStateChange(state yggdrasil.ServiceState, info *yggdrasil.NodeInfo) {
	t.logger.Debug("Service state changed", zap.Int("state", int(state)))

	cef.QueueAsyncCall(func(id int) {
		t.mu.Lock()
		defer t.mu.Unlock()

		switch state {
		case yggdrasil.StateStopped:
			t.isRunning = false
			t.mStartStop.SetTitle("Start Node")
			t.mStartStop.Enable()
			t.setIconLocked(TrayIconStopped)
			systray.SetTooltip("Yggdrasil Network - Stopped")

		case yggdrasil.StateStarting:
			t.mStartStop.SetTitle("Starting...")
			t.mStartStop.Disable()
			t.setIconLocked(TrayIconConnecting)
			systray.SetTooltip("Yggdrasil Network - Starting...")

		case yggdrasil.StateRunning:
			t.isRunning = true
			t.mStartStop.SetTitle("Stop Node")
			t.mStartStop.Enable()
			t.setIconLocked(TrayIconRunning)
			tooltip := "Yggdrasil Network - Running"
			if info != nil && info.IPv6Address != "" {
				tooltip = "Yggdrasil: " + info.IPv6Address
			}
			systray.SetTooltip(tooltip)

		case yggdrasil.StateStopping:
			t.mStartStop.SetTitle("Stopping...")
			t.mStartStop.Disable()
			t.setIconLocked(TrayIconConnecting)
			systray.SetTooltip("Yggdrasil Network - Stopping...")
		}
	})
}

// setIcon sets the tray icon (thread-safe)
func (t *TrayManager) setIcon(icon TrayIcon) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.setIconLocked(icon)
}

// setIconLocked sets the tray icon (must be called with lock held)
func (t *TrayManager) setIconLocked(icon TrayIcon) {
	if t.currentIcon == icon {
		return
	}
	t.currentIcon = icon

	var iconData []byte
	switch icon {
	case TrayIconStopped:
		iconData = t.iconStopped
	case TrayIconRunning:
		iconData = t.iconRunning
	case TrayIconConnecting:
		iconData = t.iconConnecting
	case TrayIconError:
		iconData = t.iconStopped // Use stopped icon for error state
	}

	t.logger.Debug("Setting tray icon", zap.Int("icon", int(icon)), zap.Int("dataSize", len(iconData)))
	if len(iconData) > 0 {
		systray.SetIcon(iconData)
	} else {
		t.logger.Warn("No icon data available for tray")
	}
}

// UpdateStatus updates the tray tooltip with current status
func (t *TrayManager) UpdateStatus(status string) {
	systray.SetTooltip(status)
}

// ShowNotification shows a system notification (if supported)
func (t *TrayManager) ShowNotification(title, message string) {
	// Note: systray doesn't support notifications directly
	// This would need platform-specific implementation
	t.logger.Info("Notification", zap.String("title", title), zap.String("message", message))
}

// Quit gracefully shuts down the system tray
func (t *TrayManager) Quit() {
	t.logger.Info("Quitting system tray")
	systray.Quit()
}

// IsNodeRunning returns whether the node is currently running
func (t *TrayManager) IsNodeRunning() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.isRunning
}
