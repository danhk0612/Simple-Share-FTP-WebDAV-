package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/lxn/walk"
)

func init() {
	if len(os.Args) == 4 && os.Args[1] == "--apply-update" {
		if err := runApplyUpdate(os.Args[2], os.Args[3]); err != nil {
			walk.MsgBox(nil, "Simple Share (FTP/WebDAV)", fmt.Sprintf("업데이트 적용에 실패했습니다.\n\n%v", err), walk.MsgBoxIconError)
		}
		os.Exit(0)
	}
}

func runApplyUpdate(targetPath, newExePath string) error {
	time.Sleep(700 * time.Millisecond)
	backup := targetPath + ".update-backup"
	_ = os.Remove(backup)
	var lastErr error
	for i := 0; i < 80; i++ {
		if err := os.Rename(targetPath, backup); err != nil {
			lastErr = err
			time.Sleep(250 * time.Millisecond)
			continue
		}
		if err := copyUpdateFile(newExePath, targetPath); err != nil {
			_ = os.Rename(backup, targetPath)
			return err
		}
		_ = os.Remove(backup)
		if err := exec.Command(targetPath).Start(); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("could not replace executable: %w", lastErr)
}

func copyUpdateFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
