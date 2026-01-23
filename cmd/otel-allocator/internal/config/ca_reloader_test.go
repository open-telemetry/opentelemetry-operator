// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestCAReloader_Reload(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate initial CA certificate
	caPEM1, _ := generateTestCertificate(t)
	caPath := filepath.Join(tmpDir, "ca.crt")
	require.NoError(t, os.WriteFile(caPath, caPEM1, 0600))

	logger := ctrl.Log.WithName("test")
	reloader, err := NewCAReloader(caPath, logger)
	require.NoError(t, err)

	initialCA := reloader.GetClientCAs()
	require.NotNil(t, initialCA)

	// Generate new CA certificate
	caPEM2, _ := generateTestCertificate(t)
	require.NoError(t, os.WriteFile(caPath, caPEM2, 0600))

	// Reload CA
	err = reloader.Reload()
	require.NoError(t, err)

	// Verify CA pool was updated
	newCA := reloader.GetClientCAs()
	require.NotNil(t, newCA)
}

func TestCAReloader_InvalidCA(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid CA first
	validCAPEM, _ := generateTestCertificate(t)
	caPath := filepath.Join(tmpDir, "ca.crt")
	require.NoError(t, os.WriteFile(caPath, validCAPEM, 0600))

	logger := ctrl.Log.WithName("test")
	reloader, err := NewCAReloader(caPath, logger)
	require.NoError(t, err)

	oldCA := reloader.GetClientCAs()
	require.NotNil(t, oldCA)

	// Write invalid CA
	require.NoError(t, os.WriteFile(caPath, []byte("invalid"), 0600))

	// Reload should fail
	err = reloader.Reload()
	require.Error(t, err)

	// Verify old CA is still in use
	currentCA := reloader.GetClientCAs()
	assert.Equal(t, oldCA, currentCA)
}
