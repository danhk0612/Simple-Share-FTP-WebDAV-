package main

import (
	"bufio"
	"context"
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
	Version             = "1.0.3"
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
	updaterPath  string
	newExePath   string
	targetPath   string
	latestVersion string
}

type updateProgressFunc func(status, detail string, percent int, marquee, cancelable bool)

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

func prepareUpdate(ctx context.Context, info releaseInfo, report updateProgressFunc, lang string) (result preparedUpdate, err error) {
	latest := latestVersion(info)
	result.latestVersion = latest
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
		return result, errors.New("release does not contain SimpleShare.exe and its SHA-256 file")
	}

	target, err := os.Executable()
	if err != nil {
		return result, err
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return result, err
	}

	tempDir, err := os.MkdirTemp("", "SimpleShare-update-*")
	if err != nil {
		return result, err
	}
	success := false
	defer func() {
		if !success {
			_ = os.RemoveAll(tempDir)
		}
	}()

	newExe := filepath.Join(tempDir, "SimpleShare.new.exe")
	hashFile := filepath.Join(tempDir, updateHashAssetName)

	if report != nil {
		report(updateText(lang, "업데이트 파일을 다운로드하는 중...", "Downloading update file..."), "0%", 0, false, true)
	}
	if err := downloadUpdateFile(ctx, exeURL, newExe, true, report, lang); err != nil {
		return result, err
	}
	if report != nil {
		report(updateText(lang, "검증 정보를 다운로드하는 중...", "Downloading verification data..."), updateText(lang, "SHA-256 체크섬을 가져오고 있습니다.", "Retrieving the SHA-256 checksum."), 100, true, true)
	}
	if err := downloadUpdateFile(ctx, hashURL, hashFile, false, report, lang); err != nil {
		return result, err
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if report != nil {
		report(updateText(lang, "다운로드한 파일을 확인하는 중...", "Verifying the downloaded file..."), "SHA-256", 100, true, false)
	}
	if err := verifySHA256File(newExe, hashFile); err != nil {
		return result, err
	}

	result.updaterPath = newExe
	result.newExePath = newExe
	result.targetPath = target
	success = true
	return result, nil
}

func startPreparedUpdate(update preparedUpdate, lang string) error {
	return exec.Command(update.updaterPath, "--apply-update", update.targetPath, update.newExePath, Version, update.latestVersion, lang).Start()
}

func downloadUpdateFile(ctx context.Context, url, path string, reportProgress bool, report updateProgressFunc, lang string) error {
	client := &http.Client{Timeout: 90 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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

	const maxSize = int64(100 << 20)
	if resp.ContentLength > maxSize {
		return errors.New("download exceeds size limit")
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := io.LimitReader(resp.Body, maxSize+1)
	buf := make([]byte, 64*1024)
	var written int64
	total := resp.ContentLength
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, readErr := reader.Read(buf)
		if n > 0 {
			written += int64(n)
			if written > maxSize {
				return errors.New("download exceeds size limit")
			}
			if _, err := f.Write(buf[:n]); err != nil {
				return err
			}
			if reportProgress && report != nil {
				percent := 0
				marquee := total <= 0
				detail := fmt.Sprintf("%.1f MB", float64(written)/(1024*1024))
				if total > 0 {
					percent = int(written * 100 / total)
					if percent > 100 {
						percent = 100
					}
					detail = fmt.Sprintf("%.1f MB / %.1f MB  (%d%%)", float64(written)/(1024*1024), float64(total)/(1024*1024), percent)
				}
				report(updateText(lang, "업데이트 파일을 다운로드하는 중...", "Downloading update file..."), detail, percent, marquee, true)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	return nil
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
