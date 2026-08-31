package main

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type updateProgressWindow struct {
	dlg        *walk.Dialog
	status     *walk.Label
	detail     *walk.Label
	progress   *walk.ProgressBar
	cancelBtn  *walk.PushButton
	cancel     context.CancelFunc
	lang       string
	cancelable bool
	closed     bool
	mu         sync.Mutex
}

func updateText(lang, ko, en string) string {
	if lang == "en" {
		return en
	}
	return ko
}

func openUpdateProgressWindow(owner walk.Form, cfg Config, latest string, cancel context.CancelFunc) (*updateProgressWindow, error) {
	w := &updateProgressWindow{cancel: cancel, lang: cfg.Language, cancelable: true}
	var cancelBtn *walk.PushButton

	definition := Dialog{
		AssignTo: &w.dlg,
		Title:    updateText(cfg.Language, "Simple Share 업데이트", "Simple Share Update"),
		MinSize:  Size{520, 250},
		Size:     Size{520, 250},
		Layout:   VBox{Margins: Margins{24, 20, 24, 20}, Spacing: 10},
		Children: []Widget{
			Label{Text: fmt.Sprintf(updateText(cfg.Language, "새 버전 v%s을(를) 설치하고 있습니다.", "Installing v%s."), latest)},
			Label{AssignTo: &w.status, Text: updateText(cfg.Language, "업데이트 파일 다운로드 준비 중...", "Preparing update download...")},
			ProgressBar{AssignTo: &w.progress, MinValue: 0, MaxValue: 100, Value: 0},
			Label{AssignTo: &w.detail, Text: "0%"},
			Composite{Layout: HBox{}, Children: []Widget{
				HSpacer{},
				PushButton{AssignTo: &cancelBtn, Text: updateText(cfg.Language, "업데이트 중단", "Cancel Update")},
			}},
		},
	}
	if err := definition.Create(owner); err != nil {
		return nil, err
	}
	w.cancelBtn = cancelBtn
	if icon := protocolStatusIcon(cfg.Protocol, manager.IsRunning()); icon != nil {
		_ = w.dlg.SetIcon(icon)
	}
	w.cancelBtn.Clicked().Attach(w.requestCancel)
	w.dlg.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		w.mu.Lock()
		cancelable := w.cancelable
		closed := w.closed
		w.mu.Unlock()
		if closed {
			return
		}
		*canceled = true
		if cancelable {
			w.requestCancel()
		}
	})
	w.dlg.Show()
	_ = w.dlg.Activate()
	return w, nil
}

func (w *updateProgressWindow) requestCancel() {
	w.mu.Lock()
	if w.closed || !w.cancelable {
		w.mu.Unlock()
		return
	}
	w.cancelable = false
	cancel := w.cancel
	w.mu.Unlock()

	w.cancelBtn.SetEnabled(false)
	_ = w.status.SetText(updateText(w.lang, "업데이트를 중단하는 중...", "Cancelling update..."))
	if cancel != nil {
		cancel()
	}
}

func (w *updateProgressWindow) set(status, detail string, percent int, marquee, cancelable bool) {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return
	}
	w.cancelable = cancelable
	w.mu.Unlock()

	w.dlg.Synchronize(func() {
		w.mu.Lock()
		closed := w.closed
		w.mu.Unlock()
		if closed {
			return
		}
		_ = w.status.SetText(status)
		_ = w.detail.SetText(detail)
		_ = w.progress.SetMarqueeMode(marquee)
		if !marquee {
			if percent < 0 {
				percent = 0
			}
			if percent > 100 {
				percent = 100
			}
			w.progress.SetValue(percent)
		}
		w.cancelBtn.SetEnabled(cancelable)
	})
}

func (w *updateProgressWindow) close() {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return
	}
	w.closed = true
	w.cancelable = false
	w.mu.Unlock()
	w.dlg.Hide()
	w.dlg.Dispose()
}

func (w *updateProgressWindow) synchronize(fn func()) {
	w.dlg.Synchronize(fn)
}

func runUpdateCompleteCountdown(targetPath, oldVersion, newVersion, lang string) error {
	var dlg *walk.Dialog
	var countdownLabel *walk.Label
	var restartBtn *walk.PushButton
	seconds := 5
	restarting := false

	refresh := func() {}
	restart := func() {}

	definition := Dialog{
		AssignTo: &dlg,
		Title:    updateText(lang, "Simple Share 업데이트", "Simple Share Update"),
		MinSize:  Size{470, 230},
		Size:     Size{470, 230},
		Layout:   VBox{Margins: Margins{24, 22, 24, 20}, Spacing: 12},
		Children: []Widget{
			Label{Text: updateText(lang, "업데이트가 완료되었습니다.", "The update is complete.")},
			Label{Text: fmt.Sprintf("v%s → v%s", oldVersion, newVersion)},
			Label{AssignTo: &countdownLabel},
			Composite{Layout: HBox{}, Children: []Widget{
				HSpacer{},
				PushButton{AssignTo: &restartBtn, Text: updateText(lang, "지금 재시작", "Restart Now")},
			}},
		},
	}
	if err := definition.Create(nil); err != nil {
		return err
	}
	_ = dlg.SetIcon(walk.IconApplication())

	refresh = func() {
		_ = countdownLabel.SetText(fmt.Sprintf(updateText(lang, "자동 재시작까지 %d초", "Restarting automatically in %d seconds"), seconds))
	}
	restart = func() {
		if restarting {
			return
		}
		restarting = true
		if err := exec.Command(targetPath).Start(); err != nil {
			restarting = false
			walk.MsgBox(dlg, updateText(lang, "Simple Share 업데이트", "Simple Share Update"), err.Error(), walk.MsgBoxIconError)
			return
		}
		dlg.Accept()
	}
	restartBtn.Clicked().Attach(restart)
	dlg.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		if restarting {
			return
		}
		*canceled = true
		restart()
	})
	refresh()

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				dlg.Synchronize(func() {
					if restarting {
						return
					}
					seconds--
					if seconds <= 0 {
						restart()
						return
					}
					refresh()
				})
			case <-done:
				return
			}
		}
	}()
	dlg.Run()
	close(done)
	return nil
}
