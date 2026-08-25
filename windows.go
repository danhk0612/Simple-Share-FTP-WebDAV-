package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

const (
	autoStartValueName = "Simple Share (FTP-WebDAV)"
	firewallRuleName   = "Simple Share (FTP-WebDAV)"
)

func setAutoStart(enabled bool) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	if !enabled {
		err := key.DeleteValue(autoStartValueName)
		if err == registry.ErrNotExist {
			return nil
		}
		return err
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return key.SetStringValue(autoStartValueName, `"`+exe+`"`)
}

func firewallAllowed() (bool, error) {
	exe, err := os.Executable()
	if err != nil {
		return false, err
	}
	quotedName := strings.ReplaceAll(firewallRuleName, "'", "''")
	quotedExe := strings.ReplaceAll(exe, "'", "''")
	script := fmt.Sprintf(`$r = Get-NetFirewallRule -DisplayName '%s' -ErrorAction SilentlyContinue | Where-Object {$_.Enabled -eq 'True'} | Get-NetFirewallApplicationFilter | Where-Object {$_.Program -ieq '%s'}; if ($r) { exit 0 } else { exit 1 }`, quotedName, quotedExe)
	cmd := hiddenCommand("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	err = cmd.Run()
	if err == nil {
		return true, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

func addFirewallRule() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	nameArg := `name="` + firewallRuleName + `"`
	programArg := `program="` + exe + `"`
	script := fmt.Sprintf(`Start-Process -FilePath 'netsh.exe' -Verb RunAs -Wait -WindowStyle Hidden -ArgumentList @('advfirewall','firewall','delete','rule','%s'); Start-Process -FilePath 'netsh.exe' -Verb RunAs -Wait -WindowStyle Hidden -ArgumentList @('advfirewall','firewall','add','rule','%s','dir=in','action=allow','%s','enable=yes','profile=any')`, escapePS(nameArg), escapePS(nameArg), escapePS(programArg))
	return hiddenCommand("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script).Run()
}

func removeFirewallRule() error {
	nameArg := `name="` + firewallRuleName + `"`
	script := fmt.Sprintf(`Start-Process -FilePath 'netsh.exe' -Verb RunAs -Wait -WindowStyle Hidden -ArgumentList @('advfirewall','firewall','delete','rule','%s')`, escapePS(nameArg))
	return hiddenCommand("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script).Run()
}

func hiddenCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}

func escapePS(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func openPath(path string) error {
	return exec.Command("explorer.exe", path).Start()
}

func openURL(url string) error {
	return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url).Start()
}
