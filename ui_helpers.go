package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lxn/walk"
)

func handleUpdate(owner walk.Form, cfg Config) {
	info, err := fetchLatestRelease()
	if err != nil {
		showErr(owner, cfg, err)
		return
	}
	latest := latestVersion(info)
	if latest == "" {
		showErr(owner, cfg, fmt.Errorf("latest release version is empty"))
		return
	}
	if compareVersion(latest, Version) <= 0 {
		msg := fmt.Sprintf("현재 최신 버전입니다.\n\n현재 버전: %s", Version)
		if cfg.Language == "en" {
			msg = fmt.Sprintf("You are using the latest version.\n\nCurrent version: %s", Version)
		}
		walk.MsgBox(owner, tr(cfg.Language, "app"), msg, walk.MsgBoxIconInformation)
		return
	}

	msg := fmt.Sprintf("새 버전 %s이 있습니다.\n현재 버전: %s\n\n지금 업데이트하시겠습니까?\n업데이트가 완료되면 프로그램이 자동으로 재시작됩니다.", latest, Version)
	if cfg.Language == "en" {
		msg = fmt.Sprintf("Version %s is available.\nCurrent version: %s\n\nUpdate now?\nThe application will restart automatically when the update is complete.", latest, Version)
	}
	if walk.MsgBox(owner, tr(cfg.Language, "app"), msg, walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) != walk.DlgCmdYes {
		return
	}

	downloading := "업데이트 파일을 다운로드하고 검증합니다. 완료 후 프로그램이 자동으로 재시작됩니다."
	if cfg.Language == "en" {
		downloading = "The update will be downloaded and verified. The application will restart automatically when complete."
	}
	walk.MsgBox(owner, tr(cfg.Language, "app"), downloading, walk.MsgBoxIconInformation)

	update, err := prepareUpdate(info)
	if err != nil {
		showErr(owner, cfg, err)
		return
	}
	if err := startPreparedUpdate(update); err != nil {
		showErr(owner, cfg, err)
		return
	}
	_ = manager.Stop()
	walk.App().Exit(0)
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

func firewallNotAllowed(lang string) string {
	if lang == "en" { return "No enabled Windows Firewall rule was found for this executable." }
	return "Windows 방화벽에서 이 실행 파일에 대한 활성 허용 규칙이 없습니다."
}

func firewallAdded(lang string) string {
	if lang == "en" { return "The Windows Firewall rule was added." }
	return "Windows 방화벽 규칙을 추가했습니다."
}

func firewallRemoved(lang string) string {
	if lang == "en" { return "The Windows Firewall allow rule was removed." }
	return "Windows 방화벽 허용 규칙을 제거했습니다."
}

func resetAsk(lang string) string {
	if lang == "en" { return "Reset all settings and run initial setup again?" }
	return "모든 설정을 초기화하고 최초 설정을 다시 진행하시겠습니까?"
}
