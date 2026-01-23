// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"crypto/x509"
	"fmt"
	"os"
	"sync"

	"github.com/go-logr/logr"
)

// CAReloader manages CA certificate reloading for client verification.
// It provides thread-safe access to the current CA certificate pool and can be
// triggered to reload via the Reload() method, typically called by a cert watcher callback.
type CAReloader struct {
	caPath    string
	clientCAs *x509.CertPool
	mu        sync.RWMutex
	logger    logr.Logger
}

// NewCAReloader creates a new CAReloader and loads the initial CA certificate.
func NewCAReloader(caPath string, logger logr.Logger) (*CAReloader, error) {
	r := &CAReloader{
		caPath: caPath,
		logger: logger.WithName("ca-reloader"),
	}

	if err := r.Reload(); err != nil {
		return nil, err
	}

	return r, nil
}

// Reload reads the CA certificate file from disk and updates the cached certificate pool.
// This method is thread-safe and can be called concurrently.
func (r *CAReloader) Reload() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	caCert, err := os.ReadFile(r.caPath)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return fmt.Errorf("failed to parse CA certificate at %s", r.caPath)
	}

	r.clientCAs = caCertPool

	r.logger.Info("CA certificate reloaded successfully", "caPath", r.caPath)
	return nil
}

// GetClientCAs returns the current CA certificate pool for client verification.
// This method is safe for concurrent access and is called during TLS handshakes.
func (r *CAReloader) GetClientCAs() *x509.CertPool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.clientCAs
}
