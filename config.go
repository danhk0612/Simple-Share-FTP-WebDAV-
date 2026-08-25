package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	Protocol  string `json:"protocol"`
	Port      int    `json:"port"`
	Root      string `json:"root"`
	Anonymous bool   `json:"anonymous"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Language  string `json:"language"`
	AutoStart bool   `json:"autoStart"`
}

func defaultConfig() Config {
	return Config{
		Protocol: "webdav",
		Port:     8080,
		Language: "ko",
		Username: "share",
	}
}

func configDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "SimpleShareFTPWebDAV"), nil
}

func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func loadConfig() (Config, bool, error) {
	cfg := defaultConfig()
	path, err := configPath()
	if err != nil {
		return cfg, false, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, false, nil
	}
	if err != nil {
		return cfg, false, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, true, err
	}
	if cfg.Language != "en" {
		cfg.Language = "ko"
	}
	if cfg.Protocol != "ftp" {
		cfg.Protocol = "webdav"
	}
	return cfg, true, nil
}

func saveConfig(cfg Config) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	path, err := configPath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}

func resetConfig() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
