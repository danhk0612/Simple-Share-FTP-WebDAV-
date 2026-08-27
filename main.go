package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

//go:embed assets/app-ftp.ico
var ftpIconBytes []byte

//go:embed assets/app-ftp-stopped.ico
var ftpStoppedIconBytes []byte

//go:embed assets/app-webdav.ico
var webdavIconBytes []byte

//go:embed assets/app-webdav-stopped.ico
var webdavStoppedIconBytes []byte

var manager ServerManager

var statusIcons struct {
	ftpRunning    *walk.Icon
	ftpStopped    *walk.Icon
	webdavRunning *walk.Icon
	webdavStopped *walk.Icon
}

func main() {
	if len(os.Args) == 4 && os.Args[1] == "--apply-update" {
		if err := runApplyUpdate(os.Args[2], os.Args[3]); err != nil {
			walk.MsgBox(nil, "Simple Share (FTP/WebDAV)", fmt.Sprintf("업데이트 적용에 실패했습니다.\n\n%v", err), walk.MsgBoxIconError)
		}
		return
	}

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
	loadStatusIcons()

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
	if icon := protocolStatusIcon(cfg.Protocol, manager.IsRunning()); icon != nil {
		_ = ni.SetIcon(icon)
	}
	_ = ni.SetToolTip(tr(cfg.Language, "app"))
	_ = ni.SetVisible(true)

	var rebuild func()
	openSettings := func() {
		wasRunning := manager.IsRunning()
		showSettingsAsync(mw, &cfg, func() {
			if wasRunning {
				_ = manager.Stop()
			}
			if err := setAutoStart(cfg.AutoStart); err != nil {
				showErr(mw, cfg, err)
			}
			if wasRunning {
				if err := manager.Start(cfg); err != nil {
					showErr(mw, cfg, err)
				}
			}
			_ = ni.SetToolTip(tr(cfg.Language, "app"))
			rebuild()
		})
	}

	var lastTrayLeftClick time.Time
	ni.MouseUp().Attach(func(_, _ int, button walk.MouseButton) {
		if button != walk.LeftButton {
			return
		}
		now := time.Now()
		if !lastTrayLeftClick.IsZero() && now.Sub(lastTrayLeftClick) <= 500*time.Millisecond {
			lastTrayLeftClick = time.Time{}
			openSettings()
			return
		}
		lastTrayLeftClick = now
	})

	rebuild = func() {
		if icon := protocolStatusIcon(cfg.Protocol, manager.IsRunning()); icon != nil {
			_ = ni.SetIcon(icon)
		}

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
			if manager.IsRunning() {
				return tr(cfg.Language, "stop")
			}
			return tr(cfg.Language, "start")
		}(), func() {
			if manager.IsRunning() {
				if err := manager.Stop(); err != nil {
					showErr(mw, cfg, err)
				}
			} else if err := manager.Start(cfg); err != nil {
				showErr(mw, cfg, err)
			}
			rebuild()
		})
		_ = actions.Add(toggle)

		_ = actions.Add(newAction(tr(cfg.Language, "openRoot"), func() {
			if cfg.Root != "" {
				_ = openPath(cfg.Root)
			}
		}))
		_ = actions.Add(walk.NewSeparatorAction())

		firewallMenu, _ := walk.NewMenu()
		_ = firewallMenu.Actions().Add(newAction(tr(cfg.Language, "firewallCheck"), func() {
			_ = ni.ShowInfo(tr(cfg.Language, "app"), tr(cfg.Language, "firewallChecking"))
			allowed, err := firewallAllowed()
			if err != nil {
				showErr(mw, cfg, err)
				return
			}
			if allowed {
				walk.MsgBox(mw, tr(cfg.Language, "app"), firewallOK(cfg.Language), walk.MsgBoxIconInformation)
			} else {
				walk.MsgBox(mw, tr(cfg.Language, "app"), firewallNotAllowed(cfg.Language), walk.MsgBoxIconInformation)
			}
		}))
		_ = firewallMenu.Actions().Add(newAction(tr(cfg.Language, "firewallAllow"), func() {
			_ = ni.ShowInfo(tr(cfg.Language, "app"), tr(cfg.Language, "firewallAdding"))
			if err := addFirewallRule(); err != nil {
				showErr(mw, cfg, err)
				return
			}
			walk.MsgBox(mw, tr(cfg.Language, "app"), firewallAdded(cfg.Language), walk.MsgBoxIconInformation)
		}))
		_ = firewallMenu.Actions().Add(newAction(tr(cfg.Language, "firewallRemove"), func() {
			_ = ni.ShowInfo(tr(cfg.Language, "app"), tr(cfg.Language, "firewallRemoving"))
			if err := removeFirewallRule(); err != nil {
				showErr(mw, cfg, err)
				return
			}
			walk.MsgBox(mw, tr(cfg.Language, "app"), firewallRemoved(cfg.Language), walk.MsgBoxIconInformation)
		}))
		firewallAction := walk.NewMenuAction(firewallMenu)
		_ = firewallAction.SetText(tr(cfg.Language, "firewallManage"))
		_ = actions.Add(firewallAction)

		_ = actions.Add(newAction(tr(cfg.Language, "update"), func() {
			if handleUpdate(mw, cfg) {
				_ = manager.Stop()
				walk.App().Exit(0)
			}
		}))
		_ = actions.Add(walk.NewSeparatorAction())

		settingsMenu, _ := walk.NewMenu()
		_ = settingsMenu.Actions().Add(newAction(tr(cfg.Language, "settings"), openSettings))
		_ = settingsMenu.Actions().Add(walk.NewSeparatorAction())
		_ = settingsMenu.Actions().Add(newAction(tr(cfg.Language, "backup"), func() {
			backupSettings(mw, cfg)
		}))
		_ = settingsMenu.Actions().Add(newAction(tr(cfg.Language, "restore"), func() {
			if restored, ok := restoreSettings(mw, cfg); ok {
				wasRunning := manager.IsRunning()
				if wasRunning {
					_ = manager.Stop()
				}
				cfg = restored
				_ = saveConfig(cfg)
				_ = setAutoStart(cfg.AutoStart)
				if wasRunning {
					if err := manager.Start(cfg); err != nil {
						showErr(mw, cfg, err)
					}
				}
				_ = ni.SetToolTip(tr(cfg.Language, "app"))
				rebuild()
			}
		}))
		_ = settingsMenu.Actions().Add(newAction(tr(cfg.Language, "reset"), func() {
			if walk.MsgBox(mw, tr(cfg.Language, "app"), resetAsk(cfg.Language), walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
				return
			}
			_ = manager.Stop()
			_ = setAutoStart(false)
			_ = resetConfig()
			cfg = defaultConfig()
			showSettingsAsync(mw, &cfg, func() {
				_ = setAutoStart(cfg.AutoStart)
				if err := manager.Start(cfg); err != nil {
					showErr(mw, cfg, err)
				}
				_ = ni.SetToolTip(tr(cfg.Language, "app"))
				rebuild()
			})
		}))
		settingsAction := walk.NewMenuAction(settingsMenu)
		_ = settingsAction.SetText(tr(cfg.Language, "settingsManage"))
		_ = actions.Add(settingsAction)

		languageMenu, _ := walk.NewMenu()
		koAction := newAction(tr(cfg.Language, "korean"), func() {
			if cfg.Language == "ko" {
				return
			}
			cfg.Language = "ko"
			_ = saveConfig(cfg)
			_ = ni.SetToolTip(tr(cfg.Language, "app"))
			rebuild()
		})
		koAction.SetCheckable(true)
		koAction.SetExclusive(true)
		koAction.SetChecked(cfg.Language == "ko")
		_ = languageMenu.Actions().Add(koAction)

		enAction := newAction(tr(cfg.Language, "english"), func() {
			if cfg.Language == "en" {
				return
			}
			cfg.Language = "en"
			_ = saveConfig(cfg)
			_ = ni.SetToolTip(tr(cfg.Language, "app"))
			rebuild()
		})
		enAction.SetCheckable(true)
		enAction.SetExclusive(true)
		enAction.SetChecked(cfg.Language == "en")
		_ = languageMenu.Actions().Add(enAction)

		languageAction := walk.NewMenuAction(languageMenu)
		_ = languageAction.SetText(tr(cfg.Language, "language"))
		_ = actions.Add(languageAction)

		auto := newAction(tr(cfg.Language, "autostart"), func() {
			cfg.AutoStart = !cfg.AutoStart
			if err := setAutoStart(cfg.AutoStart); err != nil {
				cfg.AutoStart = !cfg.AutoStart
				showErr(mw, cfg, err)
				return
			}
			_ = saveConfig(cfg)
			rebuild()
		})
		auto.SetCheckable(true)
		auto.SetChecked(cfg.AutoStart)
		_ = actions.Add(auto)

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
	dlg, accepted, err := createSettingsDialog(owner, cfg, nil)
	if err != nil {
		return false
	}
	dlg.Run()
	return *accepted
}

func showSettingsAsync(owner walk.Form, cfg *Config, onSaved func()) {
	dlg, _, err := createSettingsDialog(owner, cfg, onSaved)
	if err != nil {
		showErr(owner, *cfg, err)
		return
	}
	dlg.Show()
	_ = dlg.Activate()
}

func createSettingsDialog(owner walk.Form, cfg *Config, onSaved func()) (*walk.Dialog, *bool, error) {
	var dlg *walk.Dialog
	var protocolCB, languageCB *walk.ComboBox
	var portLE, rootLE, userLE, passLE *walk.LineEdit
	var anonymousCB, autoCB *walk.CheckBox
	var savePB, cancelPB *walk.PushButton
	accepted := false
	async := onSaved != nil

	protocolIndex := 0
	if cfg.Protocol == "ftp" {
		protocolIndex = 1
	}
	languageIndex := 0
	if cfg.Language == "en" {
		languageIndex = 1
	}

	definition := Dialog{
		AssignTo:      &dlg,
		Title:         tr(cfg.Language, "app"),
		MinSize:       Size{520, 330},
		Layout:        VBox{},
		DefaultButton: &savePB,
		CancelButton:  &cancelPB,
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
						if ok, _ := fd.ShowBrowseFolder(dlg); ok {
							_ = rootLE.SetText(fd.FilePath)
						}
					}},
				}},
				Label{Text: ""}, CheckBox{AssignTo: &anonymousCB, Text: tr(cfg.Language, "anonymous"), Checked: cfg.Anonymous},
				Label{Text: tr(cfg.Language, "username")}, LineEdit{AssignTo: &userLE, Text: cfg.Username},
				Label{Text: tr(cfg.Language, "password")}, LineEdit{AssignTo: &passLE, Text: cfg.Password, PasswordMode: true},
				Label{Text: tr(cfg.Language, "autostart")}, CheckBox{AssignTo: &autoCB, Checked: cfg.AutoStart},
				Label{Text: tr(cfg.Language, "language")}, ComboBox{AssignTo: &languageCB, Model: []string{tr(cfg.Language, "korean"), tr(cfg.Language, "english")}, CurrentIndex: languageIndex},
			}},
			Composite{Layout: HBox{}, Children: []Widget{
				HSpacer{},
				PushButton{AssignTo: &savePB, Text: tr(cfg.Language, "save"), OnClicked: func() {
					port, err := strconv.Atoi(portLE.Text())
					if err != nil || port < 1 || port > 65535 {
						walk.MsgBox(dlg, tr(cfg.Language, "app"), tr(cfg.Language, "invalidPort"), walk.MsgBoxIconWarning)
						return
					}
					if rootLE.Text() == "" {
						walk.MsgBox(dlg, tr(cfg.Language, "app"), tr(cfg.Language, "invalidRoot"), walk.MsgBoxIconWarning)
						return
					}
					if !anonymousCB.Checked() && (userLE.Text() == "" || passLE.Text() == "") {
						walk.MsgBox(dlg, tr(cfg.Language, "app"), tr(cfg.Language, "needCredential"), walk.MsgBoxIconWarning)
						return
					}
					cfg.Protocol = "webdav"
					if protocolCB.CurrentIndex() == 1 {
						cfg.Protocol = "ftp"
					}
					cfg.Port, cfg.Root, cfg.Anonymous = port, rootLE.Text(), anonymousCB.Checked()
					cfg.Username, cfg.Password, cfg.AutoStart = userLE.Text(), passLE.Text(), autoCB.Checked()
					cfg.Language = "ko"
					if languageCB.CurrentIndex() == 1 {
						cfg.Language = "en"
					}
					if err := saveConfig(*cfg); err != nil {
						showErr(dlg, *cfg, err)
						return
					}
					accepted = true
					if async {
						dlg.Hide()
						onSaved()
						return
					}
					dlg.Accept()
				}},
				PushButton{AssignTo: &cancelPB, Text: tr(cfg.Language, "cancel"), OnClicked: func() {
					if async {
						dlg.Hide()
						return
					}
					dlg.Cancel()
				}},
			}},
		},
	}

	if err := definition.Create(owner); err != nil {
		return nil, &accepted, err
	}
	if icon := protocolStatusIcon(cfg.Protocol, true); icon != nil {
		_ = dlg.SetIcon(icon)
	}
	if async {
		dlg.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
			*canceled = true
			dlg.Hide()
		})
	}
	return dlg, &accepted, nil
}

func loadStatusIcons() {
	statusIcons.ftpRunning = loadEmbeddedIcon("app-ftp.ico", ftpIconBytes)
	statusIcons.ftpStopped = loadEmbeddedIcon("app-ftp-stopped.ico", ftpStoppedIconBytes)
	statusIcons.webdavRunning = loadEmbeddedIcon("app-webdav.ico", webdavIconBytes)
	statusIcons.webdavStopped = loadEmbeddedIcon("app-webdav-stopped.ico", webdavStoppedIconBytes)
}

func loadEmbeddedIcon(name string, data []byte) *walk.Icon {
	dir := filepath.Join(os.TempDir(), "SimpleShareFTPWebDAV")
	if os.MkdirAll(dir, 0700) != nil {
		return walk.IconApplication()
	}
	path := filepath.Join(dir, name)
	if os.WriteFile(path, data, 0600) != nil {
		return walk.IconApplication()
	}
	icon, err := walk.NewIconFromFile(path)
	if err != nil {
		return walk.IconApplication()
	}
	return icon
}

func protocolStatusIcon(protocol string, running bool) *walk.Icon {
	if protocol == "ftp" {
		if running {
			return statusIcons.ftpRunning
		}
		return statusIcons.ftpStopped
	}
	if running {
		return statusIcons.webdavRunning
	}
	return statusIcons.webdavStopped
}

func protocolLabel(p string) string {
	if p == "ftp" {
		return "FTP"
	}
	return "WebDAV"
}

func showErr(owner walk.Form, cfg Config, err error) {
	walk.MsgBox(owner, tr(cfg.Language, "app"), err.Error(), walk.MsgBoxIconError)
}
