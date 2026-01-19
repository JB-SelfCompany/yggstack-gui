package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"runtime"

	"github.com/energye/energy/v2/cef"
	"github.com/energye/golcl/lcl"
	"github.com/JB-SelfCompany/yggstack-gui/internal/app"
	"github.com/JB-SelfCompany/yggstack-gui/internal/logger"
	"github.com/JB-SelfCompany/yggstack-gui/internal/version"
	"github.com/JB-SelfCompany/yggstack-gui/internal/web"
)

//go:embed resources
var resources embed.FS

func main() {
	// Check if app should start minimized (from autostart)
	startMinimized := false
	for _, arg := range os.Args[1:] {
		if arg == "--minimized" {
			startMinimized = true
			break
		}
	}

	// Initialize logger with file output
	log := logger.NewWithConfig(logger.Config{
		Level:      "info",
		FilePath:   logger.GetLogFilePath(),
		Console:    true,
		Production: false,
	})
	defer log.Sync()

	if startMinimized {
		log.Info("Starting Yggstack-GUI in minimized mode (from autostart)", "version", version.Version)
	} else {
		log.Info("Starting Yggstack-GUI", "version", version.Version)
	}

	// Determine URL: use dev server or embedded assets
	appURL := os.Getenv("YGGSTACK_DEV_URL")
	if appURL == "" {
		// Production mode: serve embedded assets
		serverURL, err := startAssetServer(log)
		if err != nil {
			log.Error("Failed to start asset server", "error", err)
			os.Exit(1)
		}
		appURL = serverURL
		log.Info("Serving embedded assets", "url", appURL)
	} else {
		log.Info("Using dev server", "url", appURL)
	}

	// Load app icon from embedded resources (platform-specific)
	var appIcon []byte
	var iconPath string
	var iconErr error
	if runtime.GOOS == "windows" {
		// Windows requires ICO format for system tray
		appIcon, iconErr = resources.ReadFile("resources/appicon.ico")
		iconPath = "resources/appicon.ico"
	} else {
		// Linux/macOS use PNG
		appIcon, iconErr = resources.ReadFile("resources/appicon.png")
		iconPath = "resources/appicon.png"
	}
	if iconErr != nil {
		log.Warn("Failed to load app icon", "error", iconErr)
	}

	// Initialize CEF with embedded resources for icon support
	cef.GlobalInit(nil, &resources)

	// Create CEF application
	cefApp := cef.NewApplication()

	// Configure browser window
	cef.BrowserWindow.Config.Url = appURL
	cef.BrowserWindow.Config.Title = "Yggstack-GUI"
	cef.BrowserWindow.Config.Width = 1200
	cef.BrowserWindow.Config.Height = 800
	cef.BrowserWindow.Config.EnableCenterWindow = true
	cef.BrowserWindow.Config.EnableResize = true
	cef.BrowserWindow.Config.Icon = iconPath

	// Create application instance for services
	application := app.New(app.Config{
		Version:            version.Version,
		Resources:          resources,
		Logger:             log,
		StartMinimized:     startMinimized,
		TrayIconStopped:    appIcon,
		TrayIconRunning:    appIcon,
		TrayIconConnecting: appIcon,
	})

	// Setup browser initialization callback
	cef.BrowserWindow.SetBrowserInit(func(event *cef.BrowserEvent, window cef.IBrowserWindow) {
		// Set window reference in application
		application.SetWindow(window)

		// Initialize system tray
		application.InitializeTray()

		// Handle window close event - minimize to tray if enabled
		window.AsLCLBrowserWindow().BrowserWindow().SetOnCloseQuery(func(sender lcl.IObject, canClose *bool) bool {
			if application.HandleWindowClose() {
				// Minimize to tray instead of closing
				*canClose = false
				window.Hide()
				return true // Event handled
			}
			// Allow normal close
			*canClose = true
			return false
		})

		// Mark as ready
		application.MarkReady()
	})

	// Run application
	cef.Run(cefApp)
}

// startAssetServer starts an HTTP server to serve embedded frontend assets
func startAssetServer(log *logger.Logger) (string, error) {
	// Get the dist subdirectory from embedded assets
	distFS, err := fs.Sub(web.Assets, "dist")
	if err != nil {
		return "", fmt.Errorf("failed to get dist subdirectory: %w", err)
	}

	// Create file server with SPA support
	fileServer := http.FileServer(http.FS(distFS))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Check if file exists
		if _, err := fs.Stat(distFS, path[1:]); err != nil {
			// File not found, serve index.html for SPA routing
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})

	// Find available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to listen: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d", port)

	// Start server in background
	go func() {
		server := &http.Server{Handler: handler}
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Error("Asset server error", "error", err)
		}
	}()

	return url, nil
}
