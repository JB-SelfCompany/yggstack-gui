<div align="center">

# 🖥️ Yggstack-GUI

Desktop GUI for Yggdrasil Userspace Network Stack

[![License](https://img.shields.io/github/license/JB-SelfCompany/yggstack-gui)](LICENSE)
![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8.svg)
![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux-lightgrey.svg) 
![Downloads](https://img.shields.io/github/downloads/JB-SelfCompany/yggstack-gui/total)
[![Visitors](https://visitor-badge.laobi.icu/badge?page_id=JB-SelfCompany.yggstack-gui)](https://github.com/JB-SelfCompany/yggstack-gui)

**[English](#) | [Русский](README.ru.md)**

[Features](#-features) • [Installation](#-installation) • [Quick Start](#-quick-start) • [Documentation](#-documentation)

</div>

---

## Overview

Yggstack-GUI is a cross-platform desktop application that provides a graphical interface for [yggstack](https://github.com/yggdrasil-network/yggstack) - a userspace network stack for the Yggdrasil mesh network. It allows you to connect to Yggdrasil without requiring TUN interface configuration or administrator privileges.

The application runs entirely in userspace using gVisor's netstack, making it perfect for:
- Users without admin/root access
- Environments where TUN interfaces are not available
- Quick connections without system-wide network changes
- Port forwarding and SOCKS5 proxy access to Yggdrasil

---

## Features

- **No TUN Required** - Runs entirely in userspace without kernel modules
- **No Admin Privileges** - Works without root/administrator access
- **SOCKS5 Proxy** - Route any SOCKS5-capable application through Yggdrasil
- **Port Forwarding** - TCP/UDP port mappings between local and Yggdrasil networks
- **Modern UI** - Vue.js-based interface with dark/light theme support
- **System Tray** - Minimize to tray, quick access controls
- **Autostart** - Launch on system startup (minimized mode)
- **Peer Management** - Add, remove, and monitor peer connections
- **Traffic Statistics** - Real-time bandwidth and connection monitoring
- **Cross-Platform** - Windows and Linux support

---

## Requirements

- **Windows 10/11** or **Linux** (x64)
- No external dependencies required

> **Note:** Unlike the standard Yggdrasil daemon, Yggstack-GUI does NOT require administrator privileges or TUN interface support.

---

## Installation

### From Binary (Recommended)

Download the latest release from [GitHub Releases](https://github.com/JB-SelfCompany/yggstack-gui/releases/latest) for your platform.

**Windows:**
1. Download `yggstack-gui-x.x.x-windows-amd64.7z`
2. Extract **all files** to any folder (e.g., `C:\Programs\yggstack-gui\`)
3. Run `yggstack-gui-windows-amd64.exe`

> **Important:** The application requires CEF libraries (DLL files, locales folder) to be in the same directory as the executable. Do not move only the .exe file - always keep all files together.

**Linux:**
1. Download `yggstack-gui-x.x.x-linux-amd64.tar.xz`
2. Extract all files:
   ```bash
   tar -xJf yggstack-gui-*.tar.xz
   ```
3. Run: `./yggstack-gui-linux-amd64`

> **Important:** Keep all files together (libraries, locales folder). The application requires CEF libraries to be in the same directory as the executable.

### From Source

**Prerequisites:**
- Go 1.22+
- Node.js 20.19+ or 22.12+
- CEF framework and liblcl library

```bash
# Clone repository
git clone https://github.com/JB-SelfCompany/yggstack-gui.git
cd yggstack-gui
```

**Install CEF framework:**

<details>
<summary><b>Windows</b></summary>

```bash
# Use Energy CLI
./energy-cli/energy-windows64.exe install
```
</details>

<details>
<summary><b>Linux (manual installation)</b></summary>

```bash
# Install dependencies
sudo apt install libgtk-3-0 libnss3 libatk-bridge2.0-0 libxkbcommon0 libgbm1 libxcomposite1 libxdamage1 libxrandr2 libcups2 libasound2

# Create CEF directory
mkdir -p energy/CEF-109_LINUX_64
cd energy/CEF-109_LINUX_64

# Download CEF 109 from Spotify CDN
wget "https://cef-builds.spotifycdn.com/cef_binary_109.1.18%2Bgf1c41e4%2Bchromium-109.0.5414.120_linux64_minimal.tar.bz2" -O cef.tar.bz2
tar -xjf cef.tar.bz2 --strip-components=1
rm cef.tar.bz2

# Download liblcl-109
wget "https://github.com/energye/liblcl/releases/download/v2.5.4/liblcl-109-GTK3.Linux64.zip"
unzip liblcl-109-GTK3.Linux64.zip
rm liblcl-109-GTK3.Linux64.zip

cd ../..
```
</details>

**Build:**

```bash
./build.sh
```

The build script will:
- Build the Vue.js frontend
- Compile the Go backend with `-tags prod` for portable mode
- Copy CEF framework files to `bin/`
- Compress binaries with UPX (if available)
- Create distribution archive in `dist/`

---

## Quick Start

1. **Launch the application**
   ```bash
   ./yggstack-gui
   ```

2. **Add peers** - Go to Settings and add Yggdrasil peers (e.g., from [public peers list](https://publicpeers.neilalexander.dev/))

3. **Start the node** - Click "Start" on the Dashboard

4. **Configure access:**
   - **SOCKS5 Proxy:** Enable in Proxy tab, configure your browser/apps to use `127.0.0.1:1080`
   - **Port Forwarding:** Set up TCP/UDP mappings in Forwarding tab

---

## Documentation

### Dashboard

The main dashboard shows:
- **Node Status** - Current state (running/stopped)
- **IPv6 Address** - Your Yggdrasil address (200::/7 range)
- **Subnet** - Your /64 subnet for routing
- **Public Key** - Node's Ed25519 public key
- **Statistics** - Connected peers, sessions, traffic, uptime

### SOCKS5 Proxy

Enable a SOCKS5 proxy server to route traffic through Yggdrasil:

| Setting | Default | Description |
|---------|---------|-------------|
| Listen Address | `127.0.0.1:1080` | Local address for SOCKS5 server |
| Nameserver | (optional) | DNS server for .ygg domain resolution |

**Browser Configuration:**
- Firefox: Settings → Network Settings → Manual proxy → SOCKS Host: `127.0.0.1`, Port: `1080`
- Chrome: Use extensions like SwitchyOmega or system proxy settings

### Port Forwarding

Four types of port mappings are supported:

| Type | Direction | Use Case |
|------|-----------|----------|
| **Local TCP** | Local → Yggdrasil | Access Yggdrasil services from local apps |
| **Remote TCP** | Yggdrasil → Local | Expose local services to Yggdrasil network |
| **Local UDP** | Local → Yggdrasil | UDP traffic to Yggdrasil |
| **Remote UDP** | Yggdrasil → Local | Receive UDP from Yggdrasil |

**Example - Access SSH on Yggdrasil node:**
```
Type: Local TCP
Source: 127.0.0.1:2222
Target: [200:xxxx::1]:22
```
Then connect: `ssh -p 2222 user@127.0.0.1`

**Example - Expose local web server:**
```
Type: Remote TCP
Source: [your-ygg-address]:8080
Target: 127.0.0.1:80
```

### Peer Management

- **Add Peers** - Enter peer URIs in standard Yggdrasil format:
  - `tls://hostname:port`
  - `tcp://hostname:port`
  - `quic://hostname:port`

- **Monitor Peers** - View connection status, latency, and traffic per peer

- **Auto-discovery** - Multicast peer discovery on local network (if configured)

### Settings

| Setting | Description |
|---------|-------------|
| **Theme** | Light, Dark, or System |
| **Language** | Interface language |
| **Autostart** | Launch on system startup |
| **Minimize to Tray** | Close button minimizes instead of exiting |
| **Listen Addresses** | Yggdrasil listen addresses |

### System Tray

When minimized to tray:
- **Left-click** - Show/hide window
- **Right-click** - Context menu with quick actions:
  - Start/Stop node
  - Show window
  - Exit application

---

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                   Yggstack-GUI                       │
├─────────────────────────────────────────────────────┤
│  Frontend (Vue.js + CEF)                            │
│  ├── Dashboard, Peers, Proxy, Forwarding, Settings  │
│  └── IPC Communication                              │
├─────────────────────────────────────────────────────┤
│  Backend (Go)                                        │
│  ├── Yggdrasil Service (Core + Netstack)            │
│  ├── SOCKS5 Proxy Server                            │
│  ├── Port Mapping Manager                           │
│  └── Config & Security (OS Keychain)                │
├─────────────────────────────────────────────────────┤
│  gVisor Netstack                                     │
│  └── Userspace TCP/IP stack (no TUN required)       │
└─────────────────────────────────────────────────────┘
```

**Key Components:**

- **Energy/CEF** - Chromium Embedded Framework for desktop UI
- **gVisor Netstack** - Userspace network stack implementation
- **Yggdrasil Core** - Mesh routing and encryption
- **IPC Bridge** - Frontend-backend communication

---

## Development

### Project Structure

```
yggstack-gui/
├── cmd/yggstack-gui/      # Application entry point
├── internal/
│   ├── app/               # Application lifecycle, tray
│   ├── config/            # Configuration management
│   ├── ipc/               # IPC handlers and events
│   ├── logger/            # Logging system
│   ├── platform/          # Platform-specific (autostart)
│   ├── security/          # Keychain, crypto, validation
│   ├── web/               # Embedded frontend assets
│   └── yggdrasil/         # Yggdrasil service, SOCKS, mappings
│       └── netstack/      # gVisor netstack integration
├── frontend/              # Vue.js frontend
│   ├── src/
│   │   ├── components/    # UI components
│   │   ├── views/         # Page views
│   │   ├── store/         # Pinia stores
│   │   └── utils/         # IPC utilities
│   └── package.json
├── resources/             # App icons
└── energy-cli/            # CEF framework tools
```

### Build Commands

```bash
# Development (with hot reload)
YGGSTACK_DEV_URL=http://localhost:5173 go run ./cmd/yggstack-gui
cd frontend && npm run dev

# Production build
cd frontend && npm run build
go build -ldflags "-s -w" -o yggstack-gui ./cmd/yggstack-gui

# Run tests
go test -v ./...
```

### Frontend Development

```bash
cd frontend
npm install
npm run dev          # Dev server at localhost:5173
npm run build        # Production build
npm run type-check   # TypeScript checking
```

---

## Troubleshooting

<details>
<summary><b>Cannot connect to peers</b></summary>

- Verify peer URIs are correct and accessible
- Check if your firewall allows outbound connections on peer ports
- Try different peers from the [public peers list](https://publicpeers.neilalexander.dev/)

</details>

<details>
<summary><b>SOCKS5 proxy not working</b></summary>

- Ensure the Yggdrasil node is running (green status)
- Verify proxy is enabled in the Proxy tab
- Check listen address (default: `127.0.0.1:1080`)
- Confirm your application is configured to use SOCKS5 (not HTTP proxy)

</details>

<details>
<summary><b>Port forwarding fails</b></summary>

- Verify source/target addresses are correctly formatted
- For remote mappings, ensure Yggdrasil is running
- Check if the local port is not already in use
- Verify the target service is running and accessible

</details>

<details>
<summary><b>Application doesn't start on Linux</b></summary>

- Install required libraries: `libgtk-3-0`, `libnss3`, `libatk-bridge2.0-0`
- Run from terminal to see error messages
- Check if CEF libraries are present in the application directory

</details>

---

## License

This project is licensed under the **GNU General Public License v3.0** - see the [LICENSE](LICENSE) file for details.

---

## Acknowledgments

- **Yggdrasil Network** - [yggdrasil-network.github.io](https://yggdrasil-network.github.io/)
- **yggstack** - Userspace Yggdrasil implementation
- **Energy Framework** - [energye.github.io](https://energye.github.io/)
- **gVisor** - Userspace network stack
- **Vue.js** - Frontend framework
- **go-socks5** - SOCKS5 server library

---

## Support

- **Issues**: [GitHub Issues](https://github.com/JB-SelfCompany/yggstack-gui/issues)
- **Yggdrasil Network**: [yggdrasil-network.github.io](https://yggdrasil-network.github.io/)
- **Public Peers**: [publicpeers.neilalexander.dev](https://publicpeers.neilalexander.dev/)

---

<div align="center">

Made with ❤️ by <a href="https://github.com/JB-SelfCompany">JB-SelfCompany</a>

</div>
