package app

import (
	"embed"
	"sync"

	"github.com/energye/energy/v2/cef"
	"go.uber.org/zap"

	"github.com/JB-SelfCompany/yggstack-gui/internal/config"
	"github.com/JB-SelfCompany/yggstack-gui/internal/ipc"
	"github.com/JB-SelfCompany/yggstack-gui/internal/logger"
	"github.com/JB-SelfCompany/yggstack-gui/internal/security"
	"github.com/JB-SelfCompany/yggstack-gui/internal/yggdrasil"
)

// Config holds application configuration
type Config struct {
	Version        string
	Resources      embed.FS
	Logger         *logger.Logger
	StartMinimized bool // Start minimized to tray (from --minimized flag)

	// Tray icons (embedded)
	TrayIconStopped    []byte
	TrayIconRunning    []byte
	TrayIconConnecting []byte
}

// Application represents the main application instance
type Application struct {
	mu            sync.RWMutex
	config        Config
	browserWindow cef.IBrowserWindow
	ipcBridge     *ipc.Bridge
	ipcHandlers   *ipc.Handlers
	yggService    *yggdrasil.Service
	logger        *logger.Logger
	zapLogger     *zap.Logger

	// Tray and lifecycle
	trayManager      *TrayManager
	lifecycleManager *LifecycleManager
	configStore      *config.Store

	// Security
	auditLogger      *logger.AuditLogger
	securityMW       *ipc.SecurityMiddleware
	secureStore      *security.SecureStore

	// Window state
	isVisible bool
}

// New creates a new Application instance
func New(cfg Config) *Application {
	// Get zap logger from wrapper
	zapLogger := cfg.Logger.Zap()

	app := &Application{
		config:    cfg,
		logger:    cfg.Logger,
		zapLogger: zapLogger,
	}

	app.initialize()
	return app
}

// initialize sets up the application
func (a *Application) initialize() {
	a.logger.Info("Initializing application")

	// Initialize lifecycle manager
	a.lifecycleManager = NewLifecycleManager(a.zapLogger)

	// Initialize config store
	a.configStore = config.NewStore()
	if err := a.configStore.Load(); err != nil {
		a.logger.Warn("Failed to load config, using defaults", "error", err)
	}

	// Apply settings to lifecycle manager
	settings := a.configStore.Get()
	a.lifecycleManager.SetMinimizeToTray(settings.App.MinimizeToTray)
	// Use command-line flag OR config setting for start minimized
	// Command-line flag takes priority (used by autostart)
	if a.config.StartMinimized {
		a.lifecycleManager.SetStartMinimized(true)
	} else {
		a.lifecycleManager.SetStartMinimized(settings.App.StartMinimized)
	}

	// Initialize audit logger for security events
	auditCfg := logger.DefaultAuditConfig()
	auditLogger, err := logger.NewAuditLogger(auditCfg)
	if err != nil {
		a.logger.Warn("Failed to initialize audit logger", "error", err)
	} else {
		a.auditLogger = auditLogger
		a.logger.Info("Audit logger initialized", "path", auditCfg.FilePath)
	}

	// Log application start
	if a.auditLogger != nil {
		a.auditLogger.LogSuccess(logger.AuditEventAppStart, "Application started", map[string]interface{}{
			"version": a.config.Version,
		})
	}

	// Initialize IPC bridge and handlers with Yggdrasil integration
	a.ipcBridge = ipc.NewBridge(a.logger)
	a.ipcHandlers = ipc.NewHandlers(a.logger)
	a.ipcHandlers.SetConfigStore(a.configStore) // Connect config store to handlers
	a.yggService = a.ipcHandlers.GetService()

	// Setup IPC log emitter to send logs to frontend
	a.logger.SetIPCEmitter(func(entry logger.LogEntry) {
		a.ipcBridge.Emit(ipc.EventLogEntry, map[string]interface{}{
			"level":     entry.Level,
			"message":   entry.Message,
			"source":    entry.Source,
			"fields":    entry.Fields,
			"timestamp": entry.Timestamp,
		})
	})

	// Initialize security middleware for IPC
	securityCfg := ipc.DefaultSecurityConfig()
	a.securityMW = ipc.NewSecurityMiddleware(securityCfg, a.logger, a.auditLogger)

	// Add security middleware to bridge
	a.ipcBridge.Use(a.securityMW.Middleware())

	// Add validation middleware
	validator := security.NewValidator()
	a.ipcBridge.Use(ipc.ValidationMiddleware(validator, a.logger))

	// Initialize tray manager
	a.trayManager = NewTrayManager(a, a.yggService, a.zapLogger)
	a.trayManager.SetIcons(
		a.config.TrayIconStopped,
		a.config.TrayIconRunning,
		a.config.TrayIconConnecting,
	)
	a.trayManager.SetCallbacks(
		a.showWindow,
		a.requestQuit,
	)

	// Set lifecycle shutdown callback
	a.lifecycleManager.SetOnShutdown(a.performShutdown)
}

// SetWindow sets the browser window reference (called from main after CEF init)
func (a *Application) SetWindow(window cef.IBrowserWindow) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.browserWindow = window
	a.isVisible = true

	// Register IPC handlers with Yggdrasil integration
	a.ipcHandlers.RegisterAll(a.ipcBridge)

	a.logger.Info("Browser window set")
}

// InitializeTray initializes the system tray
func (a *Application) InitializeTray() {
	a.trayManager.Initialize()
}

// MarkReady marks the application as ready
func (a *Application) MarkReady() {
	// Check if we should start minimized
	if a.lifecycleManager.ShouldStartMinimized() {
		a.hideWindow()
		// Auto-start the node when starting minimized
		a.autoStartNode()
	}

	a.lifecycleManager.Ready()
	a.logger.Info("Application ready")
}

// autoStartNode automatically starts the Yggdrasil node and SOCKS proxy
func (a *Application) autoStartNode() {
	if a.yggService == nil {
		a.logger.Warn("Cannot auto-start: Yggdrasil service not initialized")
		return
	}

	// Load configuration
	if err := a.yggService.ConfigManager().Load(); err != nil {
		a.logger.Warn("Failed to load configuration for auto-start", "error", err)
		return
	}

	cfg := a.yggService.ConfigManager().GetConfig()

	// Start the service
	if err := a.yggService.Start(cfg); err != nil {
		a.logger.Warn("Failed to auto-start Yggdrasil node", "error", err)
		return
	}

	a.logger.Info("Yggdrasil node auto-started (start minimized)")

	// Auto-start SOCKS proxy with settings from config store
	if a.configStore != nil && a.ipcHandlers != nil {
		appSettings := a.configStore.Get()
		socksConfig := yggdrasil.SOCKSConfig{
			Enabled:       true,
			ListenAddress: appSettings.Proxy.ListenAddress,
			Nameserver:    appSettings.Proxy.Nameserver,
		}
		// Get SOCKS proxy from handlers
		if socksProxy := a.ipcHandlers.GetSOCKSProxy(); socksProxy != nil {
			if err := socksProxy.Start(socksConfig); err != nil {
				a.logger.Warn("Failed to auto-start SOCKS proxy", "error", err)
			} else {
				a.logger.Info("SOCKS proxy auto-started", "address", socksConfig.ListenAddress)
			}
		}
	}
}

// BrowserWindow returns the browser window
func (a *Application) BrowserWindow() cef.IBrowserWindow {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.browserWindow
}

// Shutdown gracefully shuts down the application
func (a *Application) Shutdown() {
	a.logger.Info("Shutting down application")
	a.lifecycleManager.RequestShutdown()
}

// performShutdown is called by lifecycle manager during shutdown
func (a *Application) performShutdown() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Log application stop
	if a.auditLogger != nil {
		a.auditLogger.LogSuccess(logger.AuditEventAppStop, "Application stopping", nil)
	}

	// Stop Yggdrasil service if running
	if a.yggService != nil && a.yggService.IsRunning() {
		a.logger.Info("Stopping Yggdrasil service")
		if err := a.yggService.Stop(); err != nil {
			a.logger.Warn("Error stopping Yggdrasil service", "error", err)
		}
	}

	// Quit system tray
	if a.trayManager != nil {
		a.trayManager.Quit()
	}

	// Save config
	if a.configStore != nil {
		if err := a.configStore.Save(); err != nil {
			a.logger.Warn("Error saving config", "error", err)
		}
	}

	// Close secure store
	if a.secureStore != nil {
		if err := a.secureStore.Close(); err != nil {
			a.logger.Warn("Error closing secure store", "error", err)
		}
	}

	// Close audit logger
	if a.auditLogger != nil {
		a.auditLogger.Flush()
		if err := a.auditLogger.Close(); err != nil {
			a.logger.Warn("Error closing audit logger", "error", err)
		}
	}

	// Close window
	if a.browserWindow != nil {
		a.browserWindow.Close()
	}
}

// showWindow shows the main window (called from tray)
func (a *Application) showWindow() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.browserWindow != nil {
		a.browserWindow.Show()
		a.isVisible = true
		a.lifecycleManager.HandleWindowShow()
	}
}

// hideWindow hides the main window to tray
func (a *Application) hideWindow() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.browserWindow != nil {
		a.browserWindow.Hide()
		a.isVisible = false
		a.lifecycleManager.HandleWindowHide()
	}
}

// requestQuit requests application quit (called from tray)
func (a *Application) requestQuit() {
	a.lifecycleManager.RequestShutdown()
	// Close the browser window which will exit the application
	a.mu.RLock()
	window := a.browserWindow
	a.mu.RUnlock()
	if window != nil {
		window.Close()
	}
}

// HandleWindowClose handles window close event
// Returns true if window should be hidden (minimize to tray)
func (a *Application) HandleWindowClose() bool {
	return a.lifecycleManager.HandleWindowClose()
}

// YggdrasilService returns the Yggdrasil service
func (a *Application) YggdrasilService() *yggdrasil.Service {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.yggService
}

// TrayManager returns the tray manager
func (a *Application) TrayManager() *TrayManager {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.trayManager
}

// LifecycleManager returns the lifecycle manager
func (a *Application) LifecycleManager() *LifecycleManager {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lifecycleManager
}

// ConfigStore returns the config store
func (a *Application) ConfigStore() *config.Store {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.configStore
}

// AuditLogger returns the audit logger
func (a *Application) AuditLogger() *logger.AuditLogger {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.auditLogger
}

// SecureStore returns the secure storage
func (a *Application) SecureStore() *security.SecureStore {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.secureStore
}
