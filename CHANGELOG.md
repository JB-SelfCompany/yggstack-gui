# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.2] - 2026-01-20

### Fixed

- **System Tray Thread Stability** - Fixed tray becoming unresponsive after some time on Windows. The systray goroutine is now locked to a single OS thread using `runtime.LockOSThread()`.
- **Tray-UI State Synchronization** - Fixed service state desynchronization between system tray and main application. Added `syncInitialState()` method that synchronizes tray state with actual service state during initialization.

### Improved

- **Safe Callback Execution** - Added `executeCallbackSafely()` and `executeOnMainThreadSafely()` helper methods for tray callbacks with panic recovery and timeout detection.
- **Version Source of Truth** - Build script now reads version from `internal/version/version.go` instead of git tags.
- **Dynamic versioninfo.json** - Windows resource file is now generated during build rather than stored in repository.

### Documentation

- Simplified installation instructions (no archive extraction needed).
- Updated Node.js requirements to 20.19+ or 22.12+.

## [0.1.1] - 2026-01-19

### Fixed

- **System Tray Freezing** - Fixed application freezing when interacting with system tray menu. Handler references are now properly stored to prevent garbage collection.
- **Auto-Start Issue** - Fixed problems with the application auto-start feature.
- **Click Handlers** - Added proper single-click and double-click handlers for the tray icon.

### Improved

- **Dynamic Versioning** - Version is no longer hardcoded. Now set at build time via ldflags.
- **Build Script** - Improved `build.sh` to automatically extract and apply version from git tags.
- **Version Display** - Application version is now correctly displayed in Settings view and status bar.

### Changed

- **Default Peers Updated** - Changed default peer list to more reliable servers:
  - `tcp://ekb.itrus.su:7991`
  - `tcp://yggno.de:18226`
  - `tcp://srv.itrus.su:7991`

## [0.1.0] - 2026-01-15

### Added

- Initial release
- Desktop GUI for Yggdrasil userspace network stack
- SOCKS5 proxy support
- TCP/UDP port forwarding
- System tray integration
- Dark/light theme support
- Multi-language support (English, Russian)
- Autostart on system startup
- Peer management
- Real-time traffic statistics

[0.1.2]: https://github.com/JB-SelfCompany/yggstack-gui/compare/0.1.1...0.1.2
[0.1.1]: https://github.com/JB-SelfCompany/yggstack-gui/compare/0.1.0...0.1.1
[0.1.0]: https://github.com/JB-SelfCompany/yggstack-gui/releases/tag/0.1.0
