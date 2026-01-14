// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTestCertificate creates a self-signed certificate and private key for testing.
func generateTestCertificate(t *testing.T) (certPEM, keyPEM []byte) {
	t.Helper()

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Create self-signed certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	// Encode certificate to PEM
	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key to PEM
	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return certPEM, keyPEM
}

func TestNewCertificateReloader(t *testing.T) {
	tmpDir := t.TempDir()

	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")
	caPath := filepath.Join(tmpDir, "ca.pem")

	// Generate test certificates
	certPEM, keyPEM := generateTestCertificate(t)
	caPEM, _ := generateTestCertificate(t)

	// Write test certificates to files
	require.NoError(t, os.WriteFile(certPath, certPEM, 0600))
	require.NoError(t, os.WriteFile(keyPath, keyPEM, 0600))
	require.NoError(t, os.WriteFile(caPath, caPEM, 0600))

	logger := logr.Discard()

	reloader, err := NewCertificateReloader(certPath, keyPath, caPath, logger)
	require.NoError(t, err)
	require.NotNil(t, reloader)

	assert.Equal(t, certPath, reloader.certPath)
	assert.Equal(t, keyPath, reloader.keyPath)
	assert.Equal(t, caPath, reloader.caPath)
	assert.NotNil(t, reloader.cert)
	assert.NotNil(t, reloader.clientCAs)
}

func TestNewCertificateReloader_InvalidCertPath(t *testing.T) {
	tmpDir := t.TempDir()

	keyPath := filepath.Join(tmpDir, "key.pem")
	caPath := filepath.Join(tmpDir, "ca.pem")

	// Generate test certificates
	_, keyPEM := generateTestCertificate(t)
	caPEM, _ := generateTestCertificate(t)

	require.NoError(t, os.WriteFile(keyPath, keyPEM, 0600))
	require.NoError(t, os.WriteFile(caPath, caPEM, 0600))

	logger := logr.Discard()

	reloader, err := NewCertificateReloader("/nonexistent/cert.pem", keyPath, caPath, logger)
	assert.Error(t, err)
	assert.Nil(t, reloader)
}

func TestNewCertificateReloader_InvalidKeyPath(t *testing.T) {
	tmpDir := t.TempDir()

	certPath := filepath.Join(tmpDir, "cert.pem")
	caPath := filepath.Join(tmpDir, "ca.pem")

	// Generate test certificates
	certPEM, _ := generateTestCertificate(t)
	caPEM, _ := generateTestCertificate(t)

	require.NoError(t, os.WriteFile(certPath, certPEM, 0600))
	require.NoError(t, os.WriteFile(caPath, caPEM, 0600))

	logger := logr.Discard()

	reloader, err := NewCertificateReloader(certPath, "/nonexistent/key.pem", caPath, logger)
	assert.Error(t, err)
	assert.Nil(t, reloader)
}

func TestNewCertificateReloader_InvalidCAPath(t *testing.T) {
	tmpDir := t.TempDir()

	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")

	// Generate test certificates
	certPEM, keyPEM := generateTestCertificate(t)

	require.NoError(t, os.WriteFile(certPath, certPEM, 0600))
	require.NoError(t, os.WriteFile(keyPath, keyPEM, 0600))

	logger := logr.Discard()

	reloader, err := NewCertificateReloader(certPath, keyPath, "/nonexistent/ca.pem", logger)
	assert.Error(t, err)
	assert.Nil(t, reloader)
}

func TestCertificateReloader_DebounceMultipleEvents(t *testing.T) {
	tmpDir := t.TempDir()

	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")
	caPath := filepath.Join(tmpDir, "ca.pem")

	// Generate and write test certificates
	certPEM, keyPEM := generateTestCertificate(t)
	caPEM, _ := generateTestCertificate(t)

	require.NoError(t, os.WriteFile(certPath, certPEM, 0600))
	require.NoError(t, os.WriteFile(keyPath, keyPEM, 0600))
	require.NoError(t, os.WriteFile(caPath, caPEM, 0600))

	logger := logr.Discard()
	reloader, err := NewCertificateReloader(certPath, keyPath, caPath, logger)
	require.NoError(t, err)

	// Track reload count
	var reloadCount atomic.Int32
	reloader.testReloadCallback = func() {
		reloadCount.Add(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start watcher in background
	watcherDone := make(chan error, 1)
	go func() {
		watcherDone <- reloader.Watch(ctx)
	}()

	// Wait for watcher to start
	time.Sleep(50 * time.Millisecond)

	// Trigger multiple rapid events (simulating Kubernetes atomic update)
	for i := 0; i < 4; i++ {
		reloader.scheduleReload()
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for debounce delay + buffer
	time.Sleep(200 * time.Millisecond)

	// Should have only 1 reload despite 4 events
	assert.Equal(t, int32(1), reloadCount.Load(), "Expected exactly 1 reload for multiple rapid events")

	cancel()
	<-watcherDone
}

func TestCertificateReloader_DebounceResetTimer(t *testing.T) {
	tmpDir := t.TempDir()

	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")
	caPath := filepath.Join(tmpDir, "ca.pem")

	// Generate and write test certificates
	certPEM, keyPEM := generateTestCertificate(t)
	caPEM, _ := generateTestCertificate(t)

	require.NoError(t, os.WriteFile(certPath, certPEM, 0600))
	require.NoError(t, os.WriteFile(keyPath, keyPEM, 0600))
	require.NoError(t, os.WriteFile(caPath, caPEM, 0600))

	logger := logr.Discard()
	reloader, err := NewCertificateReloader(certPath, keyPath, caPath, logger)
	require.NoError(t, err)

	// Set shorter debounce for faster test
	reloader.debounceDelay = 100 * time.Millisecond

	var reloadCount atomic.Int32
	var reloadTimes []time.Time
	var timesMu sync.Mutex

	reloader.testReloadCallback = func() {
		reloadCount.Add(1)
		timesMu.Lock()
		reloadTimes = append(reloadTimes, time.Now())
		timesMu.Unlock()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcherDone := make(chan error, 1)
	go func() {
		watcherDone <- reloader.Watch(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// First event
	startTime := time.Now()
	reloader.scheduleReload()

	// Wait 50ms (half the debounce delay)
	time.Sleep(50 * time.Millisecond)

	// Second event should reset the timer
	reloader.scheduleReload()

	// Wait for reload to happen
	time.Sleep(200 * time.Millisecond)

	// Should have only 1 reload, happening ~100ms after the second event
	assert.Equal(t, int32(1), reloadCount.Load())

	timesMu.Lock()
	if len(reloadTimes) > 0 {
		elapsed := reloadTimes[0].Sub(startTime)
		// Should be closer to 150ms (50ms + 100ms) than 100ms
		assert.Greater(t, elapsed, 120*time.Millisecond, "Timer should have been reset by second event")
	}
	timesMu.Unlock()

	cancel()
	<-watcherDone
}

func TestCertificateReloader_SimulateKubernetesAtomicUpdate(t *testing.T) {
	tmpDir := t.TempDir()

	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")
	caPath := filepath.Join(tmpDir, "ca.pem")

	// Generate and write initial certificates
	certPEM1, keyPEM1 := generateTestCertificate(t)
	caPEM, _ := generateTestCertificate(t)

	require.NoError(t, os.WriteFile(certPath, certPEM1, 0600))
	require.NoError(t, os.WriteFile(keyPath, keyPEM1, 0600))
	require.NoError(t, os.WriteFile(caPath, caPEM, 0600))

	logger := logr.Discard()
	reloader, err := NewCertificateReloader(certPath, keyPath, caPath, logger)
	require.NoError(t, err)

	var reloadCount atomic.Int32
	reloader.testReloadCallback = func() {
		reloadCount.Add(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcherDone := make(chan error, 1)
	go func() {
		watcherDone <- reloader.Watch(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Simulate Kubernetes atomic update pattern
	// 1. Create timestamped directory
	timestampedDir := filepath.Join(tmpDir, "..2026_01_14_22_49_52.97371130")
	require.NoError(t, os.Mkdir(timestampedDir, 0755))
	time.Sleep(20 * time.Millisecond)

	// 2. Create temporary symlink
	tmpSymlink := filepath.Join(tmpDir, "..data_tmp")
	require.NoError(t, os.Symlink(timestampedDir, tmpSymlink))
	time.Sleep(20 * time.Millisecond)

	// 3. Generate new certificates and write to timestamped dir
	certPEM2, keyPEM2 := generateTestCertificate(t)
	require.NoError(t, os.WriteFile(filepath.Join(timestampedDir, "cert.pem"), certPEM2, 0600))
	require.NoError(t, os.WriteFile(filepath.Join(timestampedDir, "key.pem"), keyPEM2, 0600))
	require.NoError(t, os.WriteFile(filepath.Join(timestampedDir, "ca.pem"), caPEM, 0600))

	// 4. Rename temporary symlink to final symlink (atomic operation)
	finalSymlink := filepath.Join(tmpDir, "..data")
	require.NoError(t, os.Rename(tmpSymlink, finalSymlink))

	// Wait for debounce + buffer
	time.Sleep(250 * time.Millisecond)

	// Should have exactly 1 reload despite multiple filesystem events
	assert.Equal(t, int32(1), reloadCount.Load(), "Expected exactly 1 reload for Kubernetes atomic update")

	cancel()
	<-watcherDone
}

func TestCertificateReloader_SingleEventStillWorks(t *testing.T) {
	tmpDir := t.TempDir()

	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")
	caPath := filepath.Join(tmpDir, "ca.pem")

	// Generate and write test certificates
	certPEM, keyPEM := generateTestCertificate(t)
	caPEM, _ := generateTestCertificate(t)

	require.NoError(t, os.WriteFile(certPath, certPEM, 0600))
	require.NoError(t, os.WriteFile(keyPath, keyPEM, 0600))
	require.NoError(t, os.WriteFile(caPath, caPEM, 0600))

	logger := logr.Discard()
	reloader, err := NewCertificateReloader(certPath, keyPath, caPath, logger)
	require.NoError(t, err)

	var reloadCount atomic.Int32
	reloader.testReloadCallback = func() {
		reloadCount.Add(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcherDone := make(chan error, 1)
	go func() {
		watcherDone <- reloader.Watch(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Trigger single event
	reloader.scheduleReload()

	// Wait for debounce delay + buffer
	time.Sleep(200 * time.Millisecond)

	// Should have exactly 1 reload
	assert.Equal(t, int32(1), reloadCount.Load(), "Expected exactly 1 reload for single event")

	cancel()
	<-watcherDone
}

func TestCertificateReloader_ReloadFailureDoesntStopWatching(t *testing.T) {
	tmpDir := t.TempDir()

	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")
	caPath := filepath.Join(tmpDir, "ca.pem")

	// Generate and write test certificates
	certPEM, keyPEM := generateTestCertificate(t)
	caPEM, _ := generateTestCertificate(t)

	require.NoError(t, os.WriteFile(certPath, certPEM, 0600))
	require.NoError(t, os.WriteFile(keyPath, keyPEM, 0600))
	require.NoError(t, os.WriteFile(caPath, caPEM, 0600))

	logger := logr.Discard()
	reloader, err := NewCertificateReloader(certPath, keyPath, caPath, logger)
	require.NoError(t, err)

	var successCount atomic.Int32

	reloader.testReloadCallback = func() {
		successCount.Add(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcherDone := make(chan error, 1)
	go func() {
		watcherDone <- reloader.Watch(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// First reload: corrupt the key file to make reload fail
	require.NoError(t, os.WriteFile(keyPath, []byte("invalid key"), 0600))
	reloader.scheduleReload()
	time.Sleep(200 * time.Millisecond)

	// Reload should have been attempted but failed
	assert.Equal(t, int32(0), successCount.Load(), "Reload should have failed")

	// Second reload: restore valid key file
	require.NoError(t, os.WriteFile(keyPath, keyPEM, 0600))
	reloader.scheduleReload()
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, int32(1), successCount.Load(), "Watcher should continue after reload failure")

	cancel()
	<-watcherDone
}

func TestCertificateReloader_ContextCancellationDuringDebounce(t *testing.T) {
	tmpDir := t.TempDir()

	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")
	caPath := filepath.Join(tmpDir, "ca.pem")

	// Generate and write test certificates
	certPEM, keyPEM := generateTestCertificate(t)
	caPEM, _ := generateTestCertificate(t)

	require.NoError(t, os.WriteFile(certPath, certPEM, 0600))
	require.NoError(t, os.WriteFile(keyPath, keyPEM, 0600))
	require.NoError(t, os.WriteFile(caPath, caPEM, 0600))

	logger := logr.Discard()
	reloader, err := NewCertificateReloader(certPath, keyPath, caPath, logger)
	require.NoError(t, err)

	var reloadCount atomic.Int32
	reloader.testReloadCallback = func() {
		reloadCount.Add(1)
	}

	ctx, cancel := context.WithCancel(context.Background())

	watcherDone := make(chan error, 1)
	go func() {
		watcherDone <- reloader.Watch(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Schedule reload
	reloader.scheduleReload()

	// Cancel context immediately (before timer fires)
	cancel()

	// Wait for watcher to exit
	err = <-watcherDone
	assert.Equal(t, context.Canceled, err)

	// Reload should not have happened
	assert.Equal(t, int32(0), reloadCount.Load(), "Reload should not happen after context cancellation")
}

func TestCertificateReloader_ConcurrentEvents(t *testing.T) {
	tmpDir := t.TempDir()

	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")
	caPath := filepath.Join(tmpDir, "ca.pem")

	// Generate and write test certificates
	certPEM, keyPEM := generateTestCertificate(t)
	caPEM, _ := generateTestCertificate(t)

	require.NoError(t, os.WriteFile(certPath, certPEM, 0600))
	require.NoError(t, os.WriteFile(keyPath, keyPEM, 0600))
	require.NoError(t, os.WriteFile(caPath, caPEM, 0600))

	logger := logr.Discard()
	reloader, err := NewCertificateReloader(certPath, keyPath, caPath, logger)
	require.NoError(t, err)

	var reloadCount atomic.Int32
	reloader.testReloadCallback = func() {
		reloadCount.Add(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcherDone := make(chan error, 1)
	go func() {
		watcherDone <- reloader.Watch(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Trigger events from multiple goroutines concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reloader.scheduleReload()
		}()
	}
	wg.Wait()

	// Wait for debounce + buffer
	time.Sleep(200 * time.Millisecond)

	// Should have exactly 1 reload despite concurrent events
	assert.Equal(t, int32(1), reloadCount.Load(), "Expected exactly 1 reload despite concurrent events")

	cancel()
	<-watcherDone
}
