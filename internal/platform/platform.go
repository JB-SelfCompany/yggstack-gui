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

// getExecutableDir returns the directory where the executable is located
// Used for portable mode - all config/data files are stored next to the executable
func getExecutableDir() string {
	exePath, err := os.Executable()
	if err != nil {
		// Fallback to current working directory
		cwd, _ := os.Getwd()
		return cwd
	}
	// Resolve symlinks to get the real path
	realPath, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		return filepath.Dir(exePath)
	}
	return filepath.Dir(realPath)
}

// getDataDir returns the data directory for the application
// PORTABLE MODE: All files are stored in "data" subdirectory next to the executable
// This makes the application fully portable - just copy the folder to another location
func getDataDir(osType OS) string {
	// Portable mode: use directory where executable is located
	return filepath.Join(getExecutableDir(), "data")
}

// getCacheDir returns the cache directory for the application
// PORTABLE MODE: cache is stored in "data/cache" subdirectory next to the executable
func getCacheDir(osType OS) string {
	return filepath.Join(getDataDir(osType), "cache")
}

// getLogDir returns the log directory for the application
// PORTABLE MODE: logs are stored in "data/logs" subdirectory next to the executable
func getLogDir(osType OS) string {
	return filepath.Join(getDataDir(osType), "logs")
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
