package platform

import (
	"os"
	"path/filepath"
	"runtime"
)

// OS represents the operating system type
type OS string

const (
	Windows OS = "windows"
	Darwin  OS = "darwin"
	Linux   OS = "linux"
	Unknown OS = "unknown"
)

// Info contains information about the current platform
type Info struct {
	OS       OS
	Arch     string
	Version  string
	HomeDir  string
	DataDir  string
	CacheDir string
	LogDir   string
}

// Current returns information about the current platform
func Current() *Info {
	info := &Info{
		OS:      GetOS(),
		Arch:    runtime.GOARCH,
		HomeDir: getHomeDir(),
	}

	info.DataDir = getDataDir(info.OS)
	info.CacheDir = getCacheDir(info.OS)
	info.LogDir = getLogDir(info.OS)

	return info
}

// GetOS returns the current operating system
func GetOS() OS {
	switch runtime.GOOS {
	case "windows":
		return Windows
	case "darwin":
		return Darwin
	case "linux":
		return Linux
	default:
		return Unknown
	}
}

// IsWindows returns true if running on Windows
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// IsDarwin returns true if running on macOS
func IsDarwin() bool {
	return runtime.GOOS == "darwin"
}

// IsLinux returns true if running on Linux
func IsLinux() bool {
	return runtime.GOOS == "linux"
}

// getHomeDir returns the user's home directory
func getHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

// getDataDir returns the platform-specific data directory
// Windows: %APPDATA%\Yggstack-GUI (Roaming AppData, like Tyr Desktop)
// macOS: ~/Library/Application Support/Yggstack-GUI
// Linux: ~/.config/yggstack-gui (XDG_CONFIG_HOME)
func getDataDir(osType OS) string {
	switch osType {
	case Windows:
		// %APPDATA%\Yggstack-GUI (Roaming AppData)
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
		return filepath.Join(appData, "Yggstack-GUI")

	case Darwin:
		// ~/Library/Application Support/Yggstack-GUI
		return filepath.Join(getHomeDir(), "Library", "Application Support", "Yggstack-GUI")

	case Linux:
		// ~/.config/yggstack-gui (XDG_CONFIG_HOME)
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			configHome = filepath.Join(getHomeDir(), ".config")
		}
		return filepath.Join(configHome, "yggstack-gui")

	default:
		return filepath.Join(getHomeDir(), ".yggstack-gui")
	}
}

// getCacheDir returns the platform-specific cache directory
func getCacheDir(osType OS) string {
	switch osType {
	case Windows:
		// %APPDATA%\Yggstack-GUI\cache
		return filepath.Join(getDataDir(osType), "cache")

	case Darwin:
		// ~/Library/Caches/Yggstack-GUI
		return filepath.Join(getHomeDir(), "Library", "Caches", "Yggstack-GUI")

	case Linux:
		// ~/.cache/yggstack-gui (XDG_CACHE_HOME)
		cacheHome := os.Getenv("XDG_CACHE_HOME")
		if cacheHome == "" {
			cacheHome = filepath.Join(getHomeDir(), ".cache")
		}
		return filepath.Join(cacheHome, "yggstack-gui")

	default:
		return filepath.Join(getHomeDir(), ".yggstack-gui", "cache")
	}
}

// getLogDir returns the platform-specific log directory
func getLogDir(osType OS) string {
	switch osType {
	case Windows:
		// %APPDATA%\Yggstack-GUI\logs
		return filepath.Join(getDataDir(osType), "logs")

	case Darwin:
		// ~/Library/Logs/Yggstack-GUI
		return filepath.Join(getHomeDir(), "Library", "Logs", "Yggstack-GUI")

	case Linux:
		// ~/.config/yggstack-gui/logs (in config dir)
		return filepath.Join(getDataDir(osType), "logs")

	default:
		return filepath.Join(getHomeDir(), ".yggstack-gui", "logs")
	}
}

// EnsureDirectories creates all necessary application directories
func EnsureDirectories() error {
	info := Current()

	dirs := []string{
		info.DataDir,
		info.CacheDir,
		info.LogDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

// GetConfigPath returns the path to the configuration file
func GetConfigPath() string {
	return filepath.Join(getDataDir(GetOS()), "config.json")
}

// GetYggdrasilConfigPath returns the default path for Yggdrasil configuration
func GetYggdrasilConfigPath() string {
	return filepath.Join(getDataDir(GetOS()), "yggdrasil.conf")
}
