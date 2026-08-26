# Simple Share (FTP/WebDAV)

[한국어](README.ko.md) | English

Simple Share (FTP/WebDAV) is a lightweight Windows tray utility for sharing a local folder over FTP or WebDAV.

Current stable release: **v1.0.1**

## Features

- FTP / WebDAV server selection
- Configurable listening port
- Shared root folder selection on first run
- Anonymous access or username/password authentication
- Windows Firewall management from the tray menu
  - Check current rule status
  - Allow this application
  - Remove the allow rule
- Protocol/status-aware tray icons
  - FTP running / stopped icons
  - WebDAV running / stopped icons
- Protocol icon on the Settings window title bar
- System tray operation
- Start / stop server from the tray
- Open the shared root folder from the tray
- Settings management submenu
  - Edit settings
  - Back up settings
  - Restore settings
  - Reset settings
- GitHub Releases update check
- Korean (default) and English UI
- Start with Windows toggle
- Single executable distribution for Windows x64

## Download

Download the latest Windows build from the GitHub Releases page:

- `SimpleShare.exe`

Release page: https://github.com/danhk0612/Simple-Share-FTP-WebDAV-/releases

## Quick start

1. Download and run `SimpleShare.exe`.
2. On first launch, choose FTP or WebDAV, the listening port, and the root folder to share.
3. Choose whether to allow anonymous access. If anonymous access is disabled, configure a username and password.
4. The selected server starts after the initial configuration is saved.
5. Use the tray icon for server control, firewall management, settings, language selection, update checks, and exit.

### Default ports

- WebDAV: `8080`
- FTP: `2121`

You can change the port in Settings.

### FTP note

FTP uses a control connection and additional data connections. Simple Share creates a Windows Firewall **program rule** for `SimpleShare.exe` instead of opening only one port, so FTP data connections created by the application can use the same permission.

## Tray menu

The tray menu includes:

- Server status
- Start / Stop server
- Open root folder
- Firewall management
  - Check status
  - Allow
  - Remove allow rule
- Check for updates
- Settings management
  - Settings
  - Back up settings
  - Restore settings
  - Reset settings
- Language
  - Korean
  - English
- Start with Windows
- Exit

The tray icon changes automatically according to the selected protocol and server state. FTP and WebDAV use distinct icons, and stopped servers use a gray variant.

## Configuration

Configuration is stored under the current user's roaming application data directory:

```text
%APPDATA%\SimpleShareFTPWebDAV\config.json
```

The configuration includes the selected protocol, port, root folder, authentication settings, language, and startup preference.

Settings backups are JSON files selected by the user. Backups can contain the configured password, so store them in a trusted location.

## Authentication

### FTP

When anonymous access is disabled, FTP authentication uses the configured username and password.

When anonymous access is enabled, the accepted anonymous usernames are:

- `anonymous`
- `ftp`

### WebDAV

When anonymous access is disabled, WebDAV uses HTTP Basic authentication with the configured username and password.

## Firewall management

From the tray menu, open **Firewall management** to:

- Check whether an enabled Windows Firewall rule exists for the current `SimpleShare.exe`
- Add an inbound allow rule for the executable
- Remove the Simple Share firewall rule

Adding or removing the rule requires Windows administrator approval.

## Update check

Simple Share checks the latest GitHub Release only when **Check for updates** is selected from the tray menu.

The application compares its built-in version with the latest release tag, such as `v1.0.1`. If a newer version is available, it offers to open the GitHub Releases page in the default browser.

Simple Share does not currently replace or update the executable automatically.

## Build

Requirements:

- Windows 10/11
- Go 1.25 or newer

Recommended local build method:

```bat
build.bat
```

`build.bat` prepares Go modules, embeds the Windows manifest and application icon, and builds `SimpleShare.exe` as a Windows GUI application.

GitHub Actions builds the Windows x64 executable on pushes and pull requests. Release commits are also used to publish versioned GitHub Releases.

## Security notes

- FTP traffic is not encrypted. Use FTP only on a trusted network or through a protected network layer such as a VPN.
- WebDAV currently uses HTTP, not HTTPS. Do not expose it directly to the public Internet without TLS termination, VPN, or another secure tunnel.
- Anonymous access allows clients to access the configured root folder without credentials.
- The configured root folder is the filesystem boundary exposed by the selected server.
- The configured password is currently stored as plain text in the local JSON configuration file.
- Exported settings backups can also contain the password.
- Protect the Windows account, configuration file, and backup files accordingly.

## Version

Current stable version: **1.0.1**

## License

MIT License. See [LICENSE](LICENSE).
