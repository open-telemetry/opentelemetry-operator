// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
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

// getCertificateBytes extracts the raw certificate bytes from a CertificateReloader for comparison.
func getCertificateBytes(t *testing.T, reloader *CertificateReloader) []byte {
	t.Helper()
	cert, err := reloader.GetCertificate(nil)
	require.NoError(t, err)
	require.NotNil(t, cert)
	require.NotEmpty(t, cert.Certificate)
	return cert.Certificate[0]
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

	// Capture initial certificate
	initialCertBytes := getCertificateBytes(t, reloader)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start watcher in background
	watcherDone := make(chan error, 1)
	go func() {
		watcherDone <- reloader.Watch(ctx)
	}()

	// Trigger multiple rapid filesystem events (simulating Kubernetes atomic update)
	// Create 4 temporary files in rapid succession
	for i := range 4 {
		tmpFile := filepath.Join(tmpDir, fmt.Sprintf("..temp-%d", i))
		require.NoError(t, os.WriteFile(tmpFile, []byte("temporary"), 0600))
		time.Sleep(10 * time.Millisecond)
	}

	// Write new certificates to trigger actual reload
	newCertPEM, newKeyPEM := generateTestCertificate(t)
	require.NoError(t, os.WriteFile(certPath, newCertPEM, 0600))
	require.NoError(t, os.WriteFile(keyPath, newKeyPEM, 0600))

	// Wait for debounce delay + buffer
	time.Sleep(200 * time.Millisecond)

	// Verify certificate changed despite multiple rapid events
	currentCertBytes := getCertificateBytes(t, reloader)
	assert.False(t, bytes.Equal(initialCertBytes, currentCertBytes), "Certificate should have changed after reload")

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

	// Capture initial certificate
	initialCertBytes := getCertificateBytes(t, reloader)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcherDone := make(chan error, 1)
	go func() {
		watcherDone <- reloader.Watch(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// First filesystem event
	tmpFile1 := filepath.Join(tmpDir, "..data_tmp")
	require.NoError(t, os.WriteFile(tmpFile1, []byte("temp1"), 0600))

	// Wait 50ms (half the debounce delay)
	time.Sleep(50 * time.Millisecond)

	// Second filesystem event should reset the timer
	tmpFile2 := filepath.Join(tmpDir, "..data")
	require.NoError(t, os.WriteFile(tmpFile2, []byte("temp2"), 0600))

	// Write new certificates to trigger actual reload
	newCertPEM, newKeyPEM := generateTestCertificate(t)
	require.NoError(t, os.WriteFile(certPath, newCertPEM, 0600))
	require.NoError(t, os.WriteFile(keyPath, newKeyPEM, 0600))

	// Wait for reload to happen (timer should fire after debounce delay)
	time.Sleep(200 * time.Millisecond)

	// Verify certificate changed (timer reset worked and reload happened)
	currentCertBytes := getCertificateBytes(t, reloader)
	assert.False(t, bytes.Equal(initialCertBytes, currentCertBytes), "Certificate should have changed after debounced reload")

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

	// Capture initial certificate
	initialCertBytes := getCertificateBytes(t, reloader)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcherDone := make(chan error, 1)
	go func() {
		watcherDone <- reloader.Watch(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Simulate Kubernetes atomic update pattern by creating the sequence of
	// filesystem events that Kubernetes generates when updating a Secret/ConfigMap:

	// 1. Create timestamped directory (triggers CREATE event)
	timestampedDir := filepath.Join(tmpDir, "..2026_01_14_22_49_52.97371130")
	require.NoError(t, os.Mkdir(timestampedDir, 0755))
	time.Sleep(20 * time.Millisecond)

	// 2. Create temporary symlink (triggers CREATE event)
	tmpSymlink := filepath.Join(tmpDir, "..data_tmp")
	require.NoError(t, os.Symlink(timestampedDir, tmpSymlink))
	time.Sleep(20 * time.Millisecond)

	// 3. Rename temporary symlink to final symlink (triggers CREATE event for ..data)
	finalSymlink := filepath.Join(tmpDir, "..data")
	require.NoError(t, os.Rename(tmpSymlink, finalSymlink))
	time.Sleep(20 * time.Millisecond)

	// 4. Write new certificate files (these would be in the timestamped dir in real K8s,
	//    but we write to the actual paths for testing to ensure reload works)
	certPEM2, keyPEM2 := generateTestCertificate(t)
	require.NoError(t, os.WriteFile(certPath, certPEM2, 0600))
	require.NoError(t, os.WriteFile(keyPath, keyPEM2, 0600))

	// Wait for debounce + buffer
	time.Sleep(250 * time.Millisecond)

	// Verify certificate changed to the new value after Kubernetes atomic update
	currentCertBytes := getCertificateBytes(t, reloader)
	assert.False(t, bytes.Equal(initialCertBytes, currentCertBytes), "Certificate should have changed after Kubernetes atomic update")

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

	// Capture initial certificate
	initialCertBytes := getCertificateBytes(t, reloader)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcherDone := make(chan error, 1)
	go func() {
		watcherDone <- reloader.Watch(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Write new certificates to trigger actual reload
	newCertPEM, newKeyPEM := generateTestCertificate(t)
	require.NoError(t, os.WriteFile(certPath, newCertPEM, 0600))
	require.NoError(t, os.WriteFile(keyPath, newKeyPEM, 0600))

	// Trigger filesystem event
	tmpFile := filepath.Join(tmpDir, "..data")
	require.NoError(t, os.WriteFile(tmpFile, []byte("temporary"), 0600))

	// Wait for debounce delay + buffer
	time.Sleep(200 * time.Millisecond)

	// Verify certificate changed
	currentCertBytes := getCertificateBytes(t, reloader)
	assert.False(t, bytes.Equal(initialCertBytes, currentCertBytes), "Certificate should have changed")

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

	// Capture initial certificate
	initialCertBytes := getCertificateBytes(t, reloader)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcherDone := make(chan error, 1)
	go func() {
		watcherDone <- reloader.Watch(ctx)
	}()

	// First reload: corrupt the key file to make reload fail
	// Writing to keyPath will trigger a fsnotify WRITE event
	require.NoError(t, os.WriteFile(keyPath, []byte("invalid key"), 0600))
	time.Sleep(200 * time.Millisecond)

	// Verify certificate didn't change (reload failed)
	afterFailureCertBytes := getCertificateBytes(t, reloader)
	assert.True(t, bytes.Equal(initialCertBytes, afterFailureCertBytes), "Certificate should not change when reload fails")

	// Second reload: write valid NEW certificate
	newCertPEM, newKeyPEM := generateTestCertificate(t)
	require.NoError(t, os.WriteFile(certPath, newCertPEM, 0600))
	require.NoError(t, os.WriteFile(keyPath, newKeyPEM, 0600))
	time.Sleep(200 * time.Millisecond)

	// Verify certificate changed (watcher recovered and reloaded)
	finalCertBytes := getCertificateBytes(t, reloader)
	assert.False(t, bytes.Equal(initialCertBytes, finalCertBytes), "Watcher should continue after reload failure")

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

	// Capture initial certificate
	initialCertBytes := getCertificateBytes(t, reloader)

	ctx, cancel := context.WithCancel(context.Background())

	watcherDone := make(chan error, 1)
	go func() {
		watcherDone <- reloader.Watch(ctx)
	}()

	// Generate new certificates and write to files
	newCertPEM, newKeyPEM := generateTestCertificate(t)
	require.NoError(t, os.WriteFile(certPath, newCertPEM, 0600))
	require.NoError(t, os.WriteFile(keyPath, newKeyPEM, 0600))

	// Trigger a filesystem event to schedule a reload
	tmpFile := filepath.Join(tmpDir, "..data")
	require.NoError(t, os.WriteFile(tmpFile, []byte("temp"), 0600))

	// Cancel context immediately (before debounce timer fires)
	time.Sleep(10 * time.Millisecond) // Small delay to ensure event is received
	cancel()

	// Wait for watcher to exit
	err = <-watcherDone
	assert.Equal(t, context.Canceled, err)

	// Verify certificate did NOT change (reload didn't happen before cancellation)
	currentCertBytes := getCertificateBytes(t, reloader)
	assert.True(t, bytes.Equal(initialCertBytes, currentCertBytes), "Certificate should not have changed after context cancellation")
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

	// Capture initial certificate
	initialCertBytes := getCertificateBytes(t, reloader)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcherDone := make(chan error, 1)
	go func() {
		watcherDone <- reloader.Watch(ctx)
	}()

	// Trigger filesystem events from multiple goroutines concurrently
	// This tests thread safety of the debouncing mechanism
	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			tmpFile := filepath.Join(tmpDir, fmt.Sprintf("..temp-concurrent-%d", id))
			_ = os.WriteFile(tmpFile, []byte("concurrent"), 0600)
		}(i)
	}
	wg.Wait()

	// Write new certificates to trigger actual reload
	newCertPEM, newKeyPEM := generateTestCertificate(t)
	require.NoError(t, os.WriteFile(certPath, newCertPEM, 0600))
	require.NoError(t, os.WriteFile(keyPath, newKeyPEM, 0600))

	// Wait for debounce + buffer
	time.Sleep(200 * time.Millisecond)

	// Verify certificate changed despite concurrent filesystem events
	currentCertBytes := getCertificateBytes(t, reloader)
	assert.False(t, bytes.Equal(initialCertBytes, currentCertBytes), "Certificate should have changed despite concurrent events")

	cancel()
	<-watcherDone
}

func TestCertificateReloader_DifferentDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create separate directories for cert, key, and CA
	certDir := filepath.Join(tmpDir, "certs")
	keyDir := filepath.Join(tmpDir, "keys")
	caDir := filepath.Join(tmpDir, "ca")

	require.NoError(t, os.Mkdir(certDir, 0755))
	require.NoError(t, os.Mkdir(keyDir, 0755))
	require.NoError(t, os.Mkdir(caDir, 0755))

	certPath := filepath.Join(certDir, "tls.crt")
	keyPath := filepath.Join(keyDir, "tls.key")
	caPath := filepath.Join(caDir, "ca.crt")

	// Generate and write test certificates to different directories
	certPEM, keyPEM := generateTestCertificate(t)
	caPEM, _ := generateTestCertificate(t)

	require.NoError(t, os.WriteFile(certPath, certPEM, 0600))
	require.NoError(t, os.WriteFile(keyPath, keyPEM, 0600))
	require.NoError(t, os.WriteFile(caPath, caPEM, 0600))

	logger := logr.Discard()
	reloader, err := NewCertificateReloader(certPath, keyPath, caPath, logger)
	require.NoError(t, err)

	// Capture initial certificate
	initialCertBytes := getCertificateBytes(t, reloader)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watcherDone := make(chan error, 1)
	go func() {
		watcherDone <- reloader.Watch(ctx)
	}()

	// Trigger events in different directories
	// 1. Create a temp file in cert directory
	tmpCertFile := filepath.Join(certDir, "..data_tmp")
	require.NoError(t, os.WriteFile(tmpCertFile, []byte("temp"), 0600))
	time.Sleep(20 * time.Millisecond)

	// 2. Create a temp file in key directory
	tmpKeyFile := filepath.Join(keyDir, "..data_tmp")
	require.NoError(t, os.WriteFile(tmpKeyFile, []byte("temp"), 0600))
	time.Sleep(20 * time.Millisecond)

	// 3. Create a temp file in CA directory
	tmpCAFile := filepath.Join(caDir, "..data_tmp")
	require.NoError(t, os.WriteFile(tmpCAFile, []byte("temp"), 0600))

	// Write new certificates to trigger actual reload
	newCertPEM1, newKeyPEM1 := generateTestCertificate(t)
	require.NoError(t, os.WriteFile(certPath, newCertPEM1, 0600))
	require.NoError(t, os.WriteFile(keyPath, newKeyPEM1, 0600))

	// Wait for debounce + buffer
	time.Sleep(200 * time.Millisecond)

	// Verify certificate changed after events across multiple directories
	afterFirstReloadBytes := getCertificateBytes(t, reloader)
	assert.False(t, bytes.Equal(initialCertBytes, afterFirstReloadBytes), "Certificate should have changed after events across multiple directories")

	// Verify individual directory updates still work
	// Write another set of new certificates
	newCertPEM2, newKeyPEM2 := generateTestCertificate(t)
	require.NoError(t, os.WriteFile(certPath, newCertPEM2, 0600))
	require.NoError(t, os.WriteFile(keyPath, newKeyPEM2, 0600))

	// Simulate Kubernetes-style update in key directory
	// Create timestamped directory and symlink
	timestampedKeyDir := filepath.Join(keyDir, "..2026_01_15_01_00_00")
	require.NoError(t, os.Mkdir(timestampedKeyDir, 0755))
	time.Sleep(150 * time.Millisecond)

	// Verify certificate changed again
	afterSecondReloadBytes := getCertificateBytes(t, reloader)
	assert.False(t, bytes.Equal(afterFirstReloadBytes, afterSecondReloadBytes), "Certificate should have changed when directory created in watched key directory")

	cancel()
	<-watcherDone
}
