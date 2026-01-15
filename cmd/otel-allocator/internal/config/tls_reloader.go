// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-logr/logr"
)

// CertificateReloader watches certificate files and reloads them on change.
// It provides dynamic certificate reloading for TLS servers without restart.
type CertificateReloader struct {
	certPath      string
	keyPath       string
	caPath        string
	cert          *tls.Certificate
	clientCAs     *x509.CertPool
	mu               sync.RWMutex
	logger           logr.Logger
	debounceDelay    time.Duration
	maxDebounceWait  time.Duration
	reloadTimer      *time.Timer
	firstEventTime   *time.Time
	timerMu          sync.Mutex
	reloadNotify     chan struct{}
}

const defaultDebounceDelay = 100 * time.Millisecond
const defaultMaxDebounceWait = 1 * time.Second

// NewCertificateReloader creates a new CertificateReloader and loads the initial certificates.
func NewCertificateReloader(certPath, keyPath, caPath string, logger logr.Logger) (*CertificateReloader, error) {
	r := &CertificateReloader{
		certPath:        certPath,
		keyPath:         keyPath,
		caPath:          caPath,
		logger:          logger.WithName("cert-reloader"),
		debounceDelay:   defaultDebounceDelay,
		maxDebounceWait: defaultMaxDebounceWait,
		reloadNotify:    make(chan struct{}, 1),
		firstEventTime:  nil,
	}

	if err := r.Reload(); err != nil {
		return nil, err
	}

	return r, nil
}

// Reload reads the certificate files from disk and updates the cached certificates.
func (r *CertificateReloader) Reload() error {
	cert, err := tls.LoadX509KeyPair(r.certPath, r.keyPath)
	if err != nil {
		return err
	}

	caCert, err := os.ReadFile(r.caPath)
	if err != nil {
		return err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	r.mu.Lock()
	r.cert = &cert
	r.clientCAs = caCertPool
	r.mu.Unlock()

	r.logger.Info("Certificates reloaded successfully",
		"certPath", r.certPath,
		"keyPath", r.keyPath,
		"caPath", r.caPath)

	return nil
}

// GetCertificate returns the current server certificate for TLS handshakes.
// This is called by the TLS stack for each new connection.
func (r *CertificateReloader) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cert, nil
}

// GetClientCAs returns the current CA certificate pool for client verification.
func (r *CertificateReloader) GetClientCAs() *x509.CertPool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.clientCAs
}

// scheduleReload schedules a certificate reload after the debounce delay.
// If a reload is already scheduled, it resets the timer.
//
// To prevent timer starvation from continuous events, this implements a
// maximum debounce wait time. Even if events keep arriving, a reload is
// guaranteed to happen within maxDebounceWait from the first event.
func (r *CertificateReloader) scheduleReload() {
	r.timerMu.Lock()
	defer r.timerMu.Unlock()

	now := time.Now()

	// Track first event time if this is the start of a new debounce window
	if r.firstEventTime == nil {
		r.firstEventTime = &now
	}

	// Calculate how long until we must reload (max wait constraint)
	timeSinceFirstEvent := now.Sub(*r.firstEventTime)
	timeUntilMaxWait := r.maxDebounceWait - timeSinceFirstEvent

	// Determine actual delay: use debounce delay, but cap at max wait time
	var actualDelay time.Duration
	if timeUntilMaxWait <= 0 {
		// We've already waited the maximum time, reload immediately
		actualDelay = 0
	} else if timeUntilMaxWait < r.debounceDelay {
		// We're close to the max wait time, use remaining time
		actualDelay = timeUntilMaxWait
	} else {
		// Normal case: use standard debounce delay
		actualDelay = r.debounceDelay
	}

	// Stop existing timer if present
	if r.reloadTimer != nil {
		// Stop existing timer and drain channel if it already fired
		if !r.reloadTimer.Stop() {
			select {
			case <-r.reloadTimer.C:
			default:
			}
		}
	}

	// Schedule reload with calculated delay
	r.reloadTimer = time.AfterFunc(actualDelay, func() {
		// Send non-blocking notification
		select {
		case r.reloadNotify <- struct{}{}:
		default:
			// Channel already has a pending reload notification
		}

		// Reset first event time for next debounce window
		r.timerMu.Lock()
		r.firstEventTime = nil
		r.timerMu.Unlock()
	})
}

// Watch starts watching the certificate files for changes and reloads them when modified.
// It blocks until the context is cancelled.
func (r *CertificateReloader) Watch(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	// Collect all unique directories containing certificate files.
	// In Kubernetes, secrets are mounted as symlinks that get updated atomically,
	// so we need to watch the directories for changes.
	// Certificate files may be in different directories.
	dirs := make(map[string]struct{})
	dirs[filepath.Dir(r.certPath)] = struct{}{}
	dirs[filepath.Dir(r.keyPath)] = struct{}{}
	dirs[filepath.Dir(r.caPath)] = struct{}{}

	// Add each unique directory to the watcher
	for dir := range dirs {
		if err := watcher.Add(dir); err != nil {
			return err
		}
		r.logger.Info("Watching certificate directory for changes", "directory", dir)
	}

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Certificate watcher stopped")
			return ctx.Err()

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			// In Kubernetes, secret updates create a new symlink target.
			// We look for Create or Write events on any file in the directory.
			if event.Op&(fsnotify.Create|fsnotify.Write) != 0 {
				r.logger.Info("Certificate file change detected", "event", event)
				r.scheduleReload()
			}

		case <-r.reloadNotify:
			if err := r.Reload(); err != nil {
				r.logger.Error(err, "Failed to reload certificates")
				// Continue watching, don't exit on reload failure
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			r.logger.Error(err, "Certificate watcher error")
		}
	}
}
