// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestMain(m *testing.M) {
	// Set up logger for tests
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	os.Exit(m.Run())
}

// generateTestCertificate generates a self-signed certificate for testing.
func generateTestCertificate(t *testing.T) (certPEM, keyPEM []byte) {
	t.Helper()

	// Generate private key
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create self-signed certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	require.NoError(t, err)

	// Encode certificate to PEM
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	// Encode private key to PEM
	privBytes, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})

	return certPEM, keyPEM
}

func TestNewTLSConfig_LoadsCertificates(t *testing.T) {
	tmpDir := t.TempDir()

	certPEM, keyPEM := generateTestCertificate(t)
	caPEM, _ := generateTestCertificate(t)

	certPath := filepath.Join(tmpDir, "tls.crt")
	keyPath := filepath.Join(tmpDir, "tls.key")
	caPath := filepath.Join(tmpDir, "ca.crt")

	require.NoError(t, os.WriteFile(certPath, certPEM, 0600))
	require.NoError(t, os.WriteFile(keyPath, keyPEM, 0600))
	require.NoError(t, os.WriteFile(caPath, caPEM, 0600))

	config := HTTPSServerConfig{
		TLSCertFilePath: certPath,
		TLSKeyFilePath:  keyPath,
		CAFilePath:      caPath,
	}

	logger := ctrl.Log.WithName("test")
	tlsConfig, certWatcher, err := config.NewTLSConfig(logger)
	require.NoError(t, err)
	require.NotNil(t, tlsConfig)
	require.NotNil(t, certWatcher)

	// Verify TLS config settings
	assert.Equal(t, tls.RequestClientCert, tlsConfig.ClientAuth)
	assert.Equal(t, uint16(tls.VersionTLS12), tlsConfig.MinVersion)
	assert.NotNil(t, tlsConfig.GetCertificate)
	assert.NotNil(t, tlsConfig.VerifyConnection)

	// Verify GetCertificate returns the loaded certificate
	cert, err := tlsConfig.GetCertificate(nil)
	require.NoError(t, err)
	require.NotNil(t, cert)
}

func TestNewTLSConfig_InvalidCertificate(t *testing.T) {
	tmpDir := t.TempDir()

	certPath := filepath.Join(tmpDir, "tls.crt")
	keyPath := filepath.Join(tmpDir, "tls.key")
	caPath := filepath.Join(tmpDir, "ca.crt")

	require.NoError(t, os.WriteFile(certPath, []byte("invalid"), 0600))
	require.NoError(t, os.WriteFile(keyPath, []byte("invalid"), 0600))
	require.NoError(t, os.WriteFile(caPath, []byte("invalid"), 0600))

	config := HTTPSServerConfig{
		TLSCertFilePath: certPath,
		TLSKeyFilePath:  keyPath,
		CAFilePath:      caPath,
	}

	logger := ctrl.Log.WithName("test")
	_, _, err := config.NewTLSConfig(logger)
	require.Error(t, err)
}
