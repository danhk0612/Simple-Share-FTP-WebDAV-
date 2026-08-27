package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// Self-update with verified in-place replacement is available from v1.0.2.
	Version             = "1.0.2"
	releasesAPI         = "https://api.github.com/repos/danhk0612/Simple-Share-FTP-WebDAV-/releases/latest"
	updateExeAssetName  = "SimpleShare.exe"
	updateHashAssetName = "SimpleShare.exe.sha256"
)

type releaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type releaseInfo struct {
	TagName string         `json:"tag_name"`
	HTMLURL string         `json:"html_url"`
	Assets  []releaseAsset `json:"assets"`
}

type preparedUpdate struct {
	updaterPath string
	newExePath  string
	targetPath  string
}

func fetchLatestRelease() (releaseInfo, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, releasesAPI, nil)
	if err != nil {
		return releaseInfo{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Simple-Share-FTP-WebDAV/"+Version)
	resp, err := client.Do(req)
	if err != nil {
		return releaseInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return releaseInfo{}, fmt.Errorf("GitHub API returned %s", resp.Status)
	}
	var info releaseInfo
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&info); err != nil {
		return releaseInfo{}, err
	}
	if strings.TrimSpace(info.TagName) == "" {
		return releaseInfo{}, errors.New("release tag is empty")
	}
	return info, nil
}

func latestVersion(info releaseInfo) string {
	return strings.TrimPrefix(strings.TrimSpace(info.TagName), "v")
}

func prepareUpdate(info releaseInfo) (preparedUpdate, error) {
	var exeURL, hashURL string
	for _, asset := range info.Assets {
		switch asset.Name {
		case updateExeAssetName:
			exeURL = asset.BrowserDownloadURL
		case updateHashAssetName:
			hashURL = asset.BrowserDownloadURL
		}
	}
	if exeURL == "" || hashURL == "" {
		return preparedUpdate{}, errors.New("release does not contain SimpleShare.exe and its SHA-256 file")
	}

	target, err := os.Executable()
	if err != nil {
		return preparedUpdate{}, err
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return preparedUpdate{}, err
	}

	tempDir, err := os.MkdirTemp("", "SimpleShare-update-*")
	if err != nil {
		return preparedUpdate{}, err
	}
	newExe := filepath.Join(tempDir, "SimpleShare.new.exe")
	hashFile := filepath.Join(tempDir, updateHashAssetName)
	updater := filepath.Join(tempDir, "SimpleShare.updater.exe")

	if err := downloadUpdateFile(exeURL, newExe); err != nil {
		return preparedUpdate{}, err
	}
	if err := downloadUpdateFile(hashURL, hashFile); err != nil {
		return preparedUpdate{}, err
	}
	if err := verifySHA256File(newExe, hashFile); err != nil {
		return preparedUpdate{}, err
	}
	if err := copyUpdateFile(target, updater); err != nil {
		return preparedUpdate{}, err
	}

	return preparedUpdate{updaterPath: updater, newExePath: newExe, targetPath: target}, nil
}

func startPreparedUpdate(update preparedUpdate) error {
	return exec.Command(update.updaterPath, "--apply-update", update.targetPath, update.newExePath).Start()
}

func downloadUpdateFile(url, path string) error {
	client := &http.Client{Timeout: 90 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Simple-Share-FTP-WebDAV/"+Version)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, io.LimitReader(resp.Body, 100<<20))
	return err
}

func verifySHA256File(exePath, hashPath string) error {
	f, err := os.Open(hashPath)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(f)
	var expected string
	if scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) > 0 {
			expected = strings.ToLower(strings.TrimSpace(fields[0]))
		}
	}
	f.Close()
	if err := scanner.Err(); err != nil {
		return err
	}
	if len(expected) != 64 {
		return errors.New("release SHA-256 file is invalid")
	}
	if _, err := hex.DecodeString(expected); err != nil {
		return errors.New("release SHA-256 file is invalid")
	}

	exe, err := os.Open(exePath)
	if err != nil {
		return err
	}
	h := sha256.New()
	_, err = io.Copy(h, exe)
	exe.Close()
	if err != nil {
		return err
	}
	if hex.EncodeToString(h.Sum(nil)) != expected {
		return errors.New("downloaded executable SHA-256 does not match release")
	}
	return nil
}

func compareVersion(a, b string) int {
	pa := strings.Split(a, ".")
	pb := strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		av, bv := 0, 0
		if i < len(pa) {
			av, _ = strconv.Atoi(numericPrefix(pa[i]))
		}
		if i < len(pb) {
			bv, _ = strconv.Atoi(numericPrefix(pb[i]))
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

func numericPrefix(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		b.WriteRune(r)
	}
	return b.String()
}
