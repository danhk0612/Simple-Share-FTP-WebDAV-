package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lxn/walk"
)

func handleUpdate(owner walk.Form, cfg Config) {
	available, latest, url, err := checkUpdate()
	if err != nil {
		showErr(owner, cfg, err)
		return
	}
	if !available {
		msg := "현재 최신 버전입니다."
		if cfg.Language == "en" { msg = "You are using the latest version." }
		walk.MsgBox(owner, tr(cfg.Language, "app"), msg+"\n\n"+Version, walk.MsgBoxIconInformation)
		return
	}
	msg := fmt.Sprintf("새 버전 %s이 있습니다.\n현재 버전: %s\n\nGitHub Releases를 여시겠습니까?", latest, Version)
	if cfg.Language == "en" { msg = fmt.Sprintf("Version %s is available.\nCurrent version: %s\n\nOpen GitHub Releases?", latest, Version) }
	if walk.MsgBox(owner, tr(cfg.Language, "app"), msg, walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) == walk.DlgCmdYes {
		if url == "" { url = releasesPage }
		_ = openURL(url)
	}
}

func backupSettings(owner walk.Form, cfg Config) {
	path, err := configPath()
	if err != nil { showErr(owner, cfg, err); return }
	fd := walk.FileDialog{
		Title: "Simple Share settings backup",
		FilePath: "SimpleShare-settings.json",
		Filter: "JSON (*.json)|*.json|All files (*.*)|*.*",
	}
	ok, err := fd.ShowSave(owner)
	if err != nil { showErr(owner, cfg, err); return }
	if !ok { return }
	b, err := os.ReadFile(path)
	if err != nil { showErr(owner, cfg, err); return }
	if filepath.Ext(fd.FilePath) == "" { fd.FilePath += ".json" }
	if err := os.WriteFile(fd.FilePath, b, 0600); err != nil { showErr(owner, cfg, err); return }
	msg := "설정을 백업했습니다."
	if cfg.Language == "en" { msg = "Settings were backed up." }
	walk.MsgBox(owner, tr(cfg.Language, "app"), msg, walk.MsgBoxIconInformation)
}

func restoreSettings(owner walk.Form, current Config) (Config, bool) {
	fd := walk.FileDialog{Title: "Simple Share settings restore", Filter: "JSON (*.json)|*.json|All files (*.*)|*.*"}
	ok, err := fd.ShowOpen(owner)
	if err != nil { showErr(owner, current, err); return current, false }
	if !ok { return current, false }
	b, err := os.ReadFile(fd.FilePath)
	if err != nil { showErr(owner, current, err); return current, false }
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil { showErr(owner, current, err); return current, false }
	if cfg.Protocol != "ftp" { cfg.Protocol = "webdav" }
	if cfg.Language != "en" { cfg.Language = "ko" }
	if cfg.Port < 1 || cfg.Port > 65535 || cfg.Root == "" {
		err := fmt.Errorf("invalid settings backup")
		showErr(owner, current, err)
		return current, false
	}
	msg := "설정을 복원했습니다."
	if cfg.Language == "en" { msg = "Settings were restored." }
	walk.MsgBox(owner, tr(cfg.Language, "app"), msg, walk.MsgBoxIconInformation)
	return cfg, true
}

func firewallOK(lang string) string {
	if lang == "en" { return "Windows Firewall allows this executable." }
	return "Windows 방화벽에서 이 실행 파일이 허용되어 있습니다."
}

func firewallAsk(lang string) string {
	if lang == "en" { return "No enabled firewall rule was found for this executable. Add one now? Administrator approval is required." }
	return "이 실행 파일에 대한 활성 방화벽 규칙이 없습니다. 지금 추가하시겠습니까? 관리자 승인이 필요합니다."
}

func firewallAdded(lang string) string {
	if lang == "en" { return "The Windows Firewall rule was added." }
	return "Windows 방화벽 규칙을 추가했습니다."
}

func resetAsk(lang string) string {
	if lang == "en" { return "Reset all settings and run initial setup again?" }
	return "모든 설정을 초기화하고 최초 설정을 다시 진행하시겠습니까?"
}
