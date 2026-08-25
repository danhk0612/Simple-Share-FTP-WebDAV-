package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

//go:embed assets/app.ico
var appIconBytes []byte

var manager ServerManager

func main() {
	cfg, exists, err := loadConfig()
	if err != nil {
		cfg = defaultConfig()
		exists = false
	}

	mw, err := walk.NewMainWindow()
	if err != nil {
		return
	}
	defer mw.Dispose()

	if !exists {
		if !showSettings(mw, &cfg) {
			return
		}
	}

	_ = setAutoStart(cfg.AutoStart)
	if err := manager.Start(cfg); err != nil {
		walk.MsgBox(mw, tr(cfg.Language, "app"), err.Error(), walk.MsgBoxIconError)
	}

	ni, err := walk.NewNotifyIcon(mw)
	if err != nil {
		return
	}
	defer ni.Dispose()
	if icon := loadTrayIcon(); icon != nil {
		_ = ni.SetIcon(icon)
	}
	_ = ni.SetToolTip(tr(cfg.Language, "app"))
	_ = ni.SetVisible(true)

	var rebuild func()
	rebuild = func() {
		actions := ni.ContextMenu().Actions()
		actions.Clear()

		status := walk.NewAction()
		if manager.IsRunning() {
			_ = status.SetText(fmt.Sprintf("%s · %s : %d", tr(cfg.Language, "running"), protocolLabel(cfg.Protocol), cfg.Port))
		} else {
			_ = status.SetText(tr(cfg.Language, "stopped"))
		}
		status.SetEnabled(false)
		_ = actions.Add(status)

		toggle := newAction(func() string {
			if manager.IsRunning() { return tr(cfg.Language, "stop") }
			return tr(cfg.Language, "start")
		}(), func() {
			if manager.IsRunning() {
				if err := manager.Stop(); err != nil { showErr(mw, cfg, err) }
			} else if err := manager.Start(cfg); err != nil {
				showErr(mw, cfg, err)
			}
			rebuild()
		})
		_ = actions.Add(toggle)

		_ = actions.Add(newAction(tr(cfg.Language, "settings"), func() {
			wasRunning := manager.IsRunning()
			if showSettings(mw, &cfg) {
				if wasRunning { _ = manager.Stop() }
				if err := setAutoStart(cfg.AutoStart); err != nil { showErr(mw, cfg, err) }
				if wasRunning {
					if err := manager.Start(cfg); err != nil { showErr(mw, cfg, err) }
				}
			}
			rebuild()
		}))
		_ = actions.Add(newAction(tr(cfg.Language, "openRoot"), func() {
			if cfg.Root != "" { _ = openPath(cfg.Root) }
		}))
		_ = actions.Add(walk.NewSeparatorAction())

		_ = actions.Add(newAction(tr(cfg.Language, "firewall"), func() {
			allowed, err := firewallAllowed()
			if err != nil { showErr(mw, cfg, err); return }
			if allowed {
				walk.MsgBox(mw, tr(cfg.Language, "app"), firewallOK(cfg.Language), walk.MsgBoxIconInformation)
				return
			}
			if walk.MsgBox(mw, tr(cfg.Language, "app"), firewallAsk(cfg.Language), walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) == walk.DlgCmdYes {
				if err := addFirewallRule(); err != nil { showErr(mw, cfg, err); return }
				walk.MsgBox(mw, tr(cfg.Language, "app"), firewallAdded(cfg.Language), walk.MsgBoxIconInformation)
			}
		}))
		_ = actions.Add(newAction(tr(cfg.Language, "update"), func() { handleUpdate(mw, cfg) }))
		_ = actions.Add(walk.NewSeparatorAction())

		_ = actions.Add(newAction(tr(cfg.Language, "backup"), func() { backupSettings(mw, cfg) }))
		_ = actions.Add(newAction(tr(cfg.Language, "restore"), func() {
			if restored, ok := restoreSettings(mw, cfg); ok {
				wasRunning := manager.IsRunning()
				if wasRunning { _ = manager.Stop() }
				cfg = restored
				_ = saveConfig(cfg)
				_ = setAutoStart(cfg.AutoStart)
				if wasRunning {
					if err := manager.Start(cfg); err != nil { showErr(mw, cfg, err) }
				}
				rebuild()
			}
		}))
		_ = actions.Add(newAction(tr(cfg.Language, "reset"), func() {
			if walk.MsgBox(mw, tr(cfg.Language, "app"), resetAsk(cfg.Language), walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes { return }
			_ = manager.Stop()
			_ = setAutoStart(false)
			_ = resetConfig()
			cfg = defaultConfig()
			if showSettings(mw, &cfg) {
				_ = setAutoStart(cfg.AutoStart)
				if err := manager.Start(cfg); err != nil { showErr(mw, cfg, err) }
			}
			rebuild()
		}))
		_ = actions.Add(walk.NewSeparatorAction())

		auto := newAction(tr(cfg.Language, "autostart"), func() {
			cfg.AutoStart = !cfg.AutoStart
			if err := setAutoStart(cfg.AutoStart); err != nil { cfg.AutoStart = !cfg.AutoStart; showErr(mw, cfg, err); return }
			_ = saveConfig(cfg)
			rebuild()
		})
		auto.SetCheckable(true)
		auto.SetChecked(cfg.AutoStart)
		_ = actions.Add(auto)

		_ = actions.Add(newAction(tr(cfg.Language, "language"), func() {
			if cfg.Language == "en" { cfg.Language = "ko" } else { cfg.Language = "en" }
			_ = saveConfig(cfg)
			_ = ni.SetToolTip(tr(cfg.Language, "app"))
			rebuild()
		}))
		_ = actions.Add(walk.NewSeparatorAction())
		_ = actions.Add(newAction(tr(cfg.Language, "exit"), func() {
			_ = manager.Stop()
			walk.App().Exit(0)
		}))
	}

	rebuild()
	mw.Run()
}

func newAction(text string, fn func()) *walk.Action {
	a := walk.NewAction()
	_ = a.SetText(text)
	a.Triggered().Attach(fn)
	return a
}

func showSettings(owner walk.Form, cfg *Config) bool {
	var dlg *walk.Dialog
	var protocolCB, languageCB *walk.ComboBox
	var portLE, rootLE, userLE, passLE *walk.LineEdit
	var anonymousCB, autoCB *walk.CheckBox
	var savePB, cancelPB *walk.PushButton
	accepted := false

	protocolIndex := 0
	if cfg.Protocol == "ftp" { protocolIndex = 1 }
	languageIndex := 0
	if cfg.Language == "en" { languageIndex = 1 }

	_, err := (Dialog{
		AssignTo: &dlg,
		Title: tr(cfg.Language, "app"),
		MinSize: Size{520, 330},
		Layout: VBox{},
		AcceptButton: &savePB,
		CancelButton: &cancelPB,
		Children: []Widget{
			Composite{Layout: Grid{Columns: 2}, Children: []Widget{
				Label{Text: tr(cfg.Language, "protocol")},
				ComboBox{AssignTo: &protocolCB, Model: []string{"WebDAV", "FTP"}, CurrentIndex: protocolIndex},
				Label{Text: tr(cfg.Language, "port")},
				LineEdit{AssignTo: &portLE, Text: strconv.Itoa(cfg.Port)},
				Label{Text: tr(cfg.Language, "root")},
				Composite{Layout: HBox{}, Children: []Widget{
					LineEdit{AssignTo: &rootLE, Text: cfg.Root},
					PushButton{Text: tr(cfg.Language, "browse"), OnClicked: func() {
						fd := walk.FileDialog{Title: tr(cfg.Language, "root"), InitialDirPath: rootLE.Text()}
						if ok, _ := fd.ShowBrowseFolder(dlg); ok { _ = rootLE.SetText(fd.FilePath) }
					}},
				}},
				Label{Text: ""}, CheckBox{AssignTo: &anonymousCB, Text: tr(cfg.Language, "anonymous"), Checked: cfg.Anonymous},
				Label{Text: tr(cfg.Language, "username")}, LineEdit{AssignTo: &userLE, Text: cfg.Username},
				Label{Text: tr(cfg.Language, "password")}, LineEdit{AssignTo: &passLE, Text: cfg.Password, PasswordMode: true},
				Label{Text: tr(cfg.Language, "autostart")}, CheckBox{AssignTo: &autoCB, Checked: cfg.AutoStart},
				Label{Text: "Language"}, ComboBox{AssignTo: &languageCB, Model: []string{tr(cfg.Language, "korean"), tr(cfg.Language, "english")}, CurrentIndex: languageIndex},
			}},
			Composite{Layout: HBox{}, Children: []Widget{
				HSpacer{},
				PushButton{AssignTo: &savePB, Text: tr(cfg.Language, "save"), OnClicked: func() {
					port, err := strconv.Atoi(portLE.Text())
					if err != nil || port < 1 || port > 65535 { walk.MsgBox(dlg, tr(cfg.Language, "app"), tr(cfg.Language, "invalidPort"), walk.MsgBoxIconWarning); return }
					if rootLE.Text() == "" { walk.MsgBox(dlg, tr(cfg.Language, "app"), tr(cfg.Language, "invalidRoot"), walk.MsgBoxIconWarning); return }
					if !anonymousCB.Checked() && (userLE.Text() == "" || passLE.Text() == "") { walk.MsgBox(dlg, tr(cfg.Language, "app"), tr(cfg.Language, "needCredential"), walk.MsgBoxIconWarning); return }
					cfg.Protocol = "webdav"
					if protocolCB.CurrentIndex() == 1 { cfg.Protocol = "ftp" }
					cfg.Port, cfg.Root, cfg.Anonymous = port, rootLE.Text(), anonymousCB.Checked()
					cfg.Username, cfg.Password, cfg.AutoStart = userLE.Text(), passLE.Text(), autoCB.Checked()
					cfg.Language = "ko"
					if languageCB.CurrentIndex() == 1 { cfg.Language = "en" }
					if err := saveConfig(*cfg); err != nil { showErr(dlg, *cfg, err); return }
					accepted = true
					dlg.Accept()
				}},
				PushButton{AssignTo: &cancelPB, Text: tr(cfg.Language, "cancel"), OnClicked: func() { dlg.Cancel() }},
			}},
		},
	}).Run(owner)
	return err == nil && accepted
}

func loadTrayIcon() *walk.Icon {
	dir := filepath.Join(os.TempDir(), "SimpleShareFTPWebDAV")
	if os.MkdirAll(dir, 0700) != nil { return walk.IconApplication() }
	path := filepath.Join(dir, "app.ico")
	if os.WriteFile(path, appIconBytes, 0600) != nil { return walk.IconApplication() }
	icon, err := walk.NewIconFromFile(path)
	if err != nil { return walk.IconApplication() }
	return icon
}

func protocolLabel(p string) string {
	if p == "ftp" { return "FTP" }
	return "WebDAV"
}

func showErr(owner walk.Form, cfg Config, err error) {
	walk.MsgBox(owner, tr(cfg.Language, "app"), err.Error(), walk.MsgBoxIconError)
}
