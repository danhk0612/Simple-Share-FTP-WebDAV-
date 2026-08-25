package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	ftpserver "github.com/fclairamb/ftpserverlib"
	"github.com/spf13/afero"
	"golang.org/x/net/webdav"
)

type ServerManager struct {
	mu      sync.Mutex
	ftp     *ftpserver.FtpServer
	web     *http.Server
	running bool
}

func (m *ServerManager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

func (m *ServerManager) Start(cfg Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}
	if cfg.Root == "" {
		return errors.New("root folder is not configured")
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return errors.New("invalid port")
	}

	if cfg.Protocol == "ftp" {
		driver := &ftpDriver{cfg: cfg}
		srv := ftpserver.NewFtpServer(driver)
		if err := srv.Listen(); err != nil {
			return err
		}
		m.ftp = srv
		m.running = true
		go func() {
			_ = srv.Serve()
		}()
		return nil
	}

	h := &webdav.Handler{
		Prefix:     "/",
		FileSystem: webdav.Dir(cfg.Root),
		LockSystem: webdav.NewMemLS(),
	}
	var handler http.Handler = h
	if !cfg.Anonymous {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			if !ok || user != cfg.Username || pass != cfg.Password {
				w.Header().Set("WWW-Authenticate", `Basic realm="Simple Share"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			h.ServeHTTP(w, r)
		})
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", cfg.Port))
	if err != nil {
		return err
	}
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	m.web = srv
	m.running = true
	go func() {
		_ = srv.Serve(ln)
	}()
	return nil
}

func (m *ServerManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}
	var err error
	if m.ftp != nil {
		err = m.ftp.Stop()
		m.ftp = nil
	}
	if m.web != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if shutdownErr := m.web.Shutdown(ctx); shutdownErr != nil && err == nil {
			err = shutdownErr
		}
		m.web = nil
	}
	m.running = false
	return err
}

type ftpDriver struct {
	cfg Config
}

func (d *ftpDriver) GetSettings() (*ftpserver.Settings, error) {
	return &ftpserver.Settings{
		ListenAddr:  fmt.Sprintf("0.0.0.0:%d", d.cfg.Port),
		Banner:      "Simple Share (FTP/WebDAV)",
		IdleTimeout: 900,
	}, nil
}

func (d *ftpDriver) ClientConnected(ftpserver.ClientContext) (string, error) {
	return "Simple Share (FTP/WebDAV)", nil
}

func (d *ftpDriver) ClientDisconnected(ftpserver.ClientContext) {}

func (d *ftpDriver) AuthUser(_ ftpserver.ClientContext, user, pass string) (ftpserver.ClientDriver, error) {
	if d.cfg.Anonymous {
		if user != "anonymous" && user != "ftp" {
			return nil, errors.New("anonymous login required")
		}
	} else if user != d.cfg.Username || pass != d.cfg.Password {
		return nil, errors.New("invalid username or password")
	}
	return afero.NewBasePathFs(afero.NewOsFs(), d.cfg.Root), nil
}

func (d *ftpDriver) GetTLSConfig() (*tls.Config, error) {
	return nil, nil
}
