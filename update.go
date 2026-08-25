package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	Version       = "1.0.0"
	releasesAPI   = "https://api.github.com/repos/danhk0612/Simple-Share-FTP-WebDAV-/releases/latest"
	releasesPage  = "https://github.com/danhk0612/Simple-Share-FTP-WebDAV-/releases"
)

type releaseInfo struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

func checkUpdate() (bool, string, string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, releasesAPI, nil)
	if err != nil {
		return false, "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Simple-Share-FTP-WebDAV/"+Version)
	resp, err := client.Do(req)
	if err != nil {
		return false, "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return false, "", releasesPage, nil
	}
	if resp.StatusCode != http.StatusOK {
		return false, "", "", fmt.Errorf("GitHub API returned %s", resp.Status)
	}
	var info releaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return false, "", "", err
	}
	if info.TagName == "" {
		return false, "", info.HTMLURL, errors.New("release tag is empty")
	}
	latest := strings.TrimPrefix(strings.TrimSpace(info.TagName), "v")
	return compareVersion(latest, Version) > 0, latest, info.HTMLURL, nil
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
