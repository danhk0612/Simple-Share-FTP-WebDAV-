# Simple Share (FTP/WebDAV)

Simple Share (FTP/WebDAV) is a lightweight Windows tray utility for sharing a local folder over FTP or WebDAV.

## Features

- FTP / WebDAV server selection
- Configurable listening port
- Shared root folder selection on first run
- Anonymous access or username/password authentication
- Windows Firewall rule check and registration
- System tray operation
- Settings backup, restore, and reset
- GitHub Releases update check
- Korean (default) and English UI
- Start with Windows toggle
- Single executable distribution target

## Quick start

1. Download `SimpleShare.exe` from GitHub Releases or build it yourself.
2. Run the executable.
3. On first launch, choose FTP or WebDAV, the port, and the root folder to share.
4. Configure anonymous access or a username/password.
5. Use the tray menu to start/stop the server, open settings, check the firewall, back up settings, restore settings, or check for updates.

### Default ports

- WebDAV: `8080`
- FTP: `2121`

> FTP uses additional data connections. Simple Share registers a Windows Firewall **program rule** rather than only opening the control port, so negotiated FTP data connections can use the same application permission.

## Configuration

Configuration is stored under the current user's roaming application data directory:

```text
%APPDATA%\SimpleShareFTPWebDAV\config.json
```

Settings backups are JSON files selected by the user. Backups can contain the configured password, so store backup files in a trusted location.

## Build

Requirements:

- Windows 10/11
- Go 1.25 or newer

```bat
build.bat
```

Or:

```bat
go mod download
go build -trimpath -ldflags="-s -w -H windowsgui" -o SimpleShare.exe .
```

GitHub Actions also builds a Windows x64 artifact on pushes and pull requests.

## Update check

The application checks the latest GitHub Release only when requested from the tray menu. The current application version is compared with the release tag (for example, `v0.2.0`). If a newer version exists, Simple Share can open the release page in the default browser.

## Security notes

- FTP is not encrypted. Use it only on a trusted network unless you add a protected network layer such as a VPN.
- The initial WebDAV implementation uses HTTP. Do not expose it directly to the public Internet without TLS termination or another secure tunnel.
- Anonymous access allows clients to access the configured root folder without credentials.
- The configured root folder is the filesystem boundary exposed by the selected server.

## Project status

Initial development version: `0.1.0`

The first milestone focuses on a compact Windows tray application and the requested core sharing features.

## License

MIT License. See [LICENSE](LICENSE).
