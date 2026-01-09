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

	"github.com/fsnotify/fsnotify"
	"github.com/go-logr/logr"
)

// CertificateReloader watches certificate files and reloads them on change.
// It provides dynamic certificate reloading for TLS servers without restart.
type CertificateReloader struct {
	certPath  string
	keyPath   string
	caPath    string
	cert      *tls.Certificate
	clientCAs *x509.CertPool
	mu        sync.RWMutex
	logger    logr.Logger
}

// NewCertificateReloader creates a new CertificateReloader and loads the initial certificates.
func NewCertificateReloader(certPath, keyPath, caPath string, logger logr.Logger) (*CertificateReloader, error) {
	r := &CertificateReloader{
		certPath: certPath,
		keyPath:  keyPath,
		caPath:   caPath,
		logger:   logger.WithName("cert-reloader"),
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

// Watch starts watching the certificate files for changes and reloads them when modified.
// It blocks until the context is cancelled.
func (r *CertificateReloader) Watch(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	// Watch the directory containing the certificates.
	// In Kubernetes, secrets are mounted as symlinks that get updated atomically,
	// so we need to watch the directory for changes.
	certDir := filepath.Dir(r.certPath)
	if err := watcher.Add(certDir); err != nil {
		return err
	}
	r.logger.Info("Watching certificate directory for changes", "directory", certDir)

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
				if err := r.Reload(); err != nil {
					r.logger.Error(err, "Failed to reload certificates")
					// Continue watching, don't exit on reload failure
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			r.logger.Error(err, "Certificate watcher error")
		}
	}
}
