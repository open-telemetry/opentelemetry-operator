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

// generateCertificateChain generates a certificate chain: root CA -> intermediate CA -> leaf cert.
func generateCertificateChain(t *testing.T) (rootCAPEM []byte, intermediateCert, leafCert *x509.Certificate) {
	t.Helper()

	// Generate root CA
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	rootTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Root CA"},
			CommonName:   "Test Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            2,
	}

	rootDER, err := x509.CreateCertificate(rand.Reader, &rootTemplate, &rootTemplate, &rootKey.PublicKey, rootKey)
	require.NoError(t, err)
	rootCAPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootDER})
	rootCert, err := x509.ParseCertificate(rootDER)
	require.NoError(t, err)

	// Generate intermediate CA
	intermediateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	intermediateTemplate := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Test Intermediate CA"},
			CommonName:   "Test Intermediate CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
		MaxPathLenZero:        false,
	}

	intermediateDER, err := x509.CreateCertificate(rand.Reader, &intermediateTemplate, rootCert, &intermediateKey.PublicKey, rootKey)
	require.NoError(t, err)
	intermediateCert, err = x509.ParseCertificate(intermediateDER)
	require.NoError(t, err)

	// Generate leaf certificate
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	leafTemplate := x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject: pkix.Name{
			Organization: []string{"Test Client"},
			CommonName:   "client.test.local",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	leafDER, err := x509.CreateCertificate(rand.Reader, &leafTemplate, intermediateCert, &leafKey.PublicKey, intermediateKey)
	require.NoError(t, err)
	leafCert, err = x509.ParseCertificate(leafDER)
	require.NoError(t, err)

	return rootCAPEM, intermediateCert, leafCert
}

func TestNewTLSConfig_VerifyConnection_WithIntermediateCertificates(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate server certificate and CA chain
	serverCertPEM, serverKeyPEM := generateTestCertificate(t)
	rootCAPEM, intermediateCert, leafCert := generateCertificateChain(t)

	// Write server certificate and key
	certPath := filepath.Join(tmpDir, "tls.crt")
	keyPath := filepath.Join(tmpDir, "tls.key")
	caPath := filepath.Join(tmpDir, "ca.crt")

	require.NoError(t, os.WriteFile(certPath, serverCertPEM, 0600))
	require.NoError(t, os.WriteFile(keyPath, serverKeyPEM, 0600))
	require.NoError(t, os.WriteFile(caPath, rootCAPEM, 0600))

	config := HTTPSServerConfig{
		TLSCertFilePath: certPath,
		TLSKeyFilePath:  keyPath,
		CAFilePath:      caPath,
	}

	logger := ctrl.Log.WithName("test")
	tlsConfig, _, err := config.NewTLSConfig(logger)
	require.NoError(t, err)
	require.NotNil(t, tlsConfig)
	require.NotNil(t, tlsConfig.VerifyConnection)

	tests := []struct {
		name          string
		peerCerts     []*x509.Certificate
		expectError   bool
		errorContains string
	}{
		{
			name:          "no client certificate",
			peerCerts:     []*x509.Certificate{},
			expectError:   true,
			errorContains: "no client certificate provided",
		},
		{
			name:          "valid certificate chain with intermediate",
			peerCerts:     []*x509.Certificate{leafCert, intermediateCert},
			expectError:   false,
			errorContains: "",
		},
		{
			name:          "leaf certificate only (missing intermediate)",
			peerCerts:     []*x509.Certificate{leafCert},
			expectError:   true,
			errorContains: "certificate verification failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := tls.ConnectionState{
				PeerCertificates: tt.peerCerts,
			}

			err := tlsConfig.VerifyConnection(cs)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewTLSConfig_VerifyConnection_OnlyVerifiesLeafCertificate(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate server certificate
	serverCertPEM, serverKeyPEM := generateTestCertificate(t)
	rootCAPEM, intermediateCert, leafCert := generateCertificateChain(t)

	// Write server certificate and key
	certPath := filepath.Join(tmpDir, "tls.crt")
	keyPath := filepath.Join(tmpDir, "tls.key")
	caPath := filepath.Join(tmpDir, "ca.crt")

	require.NoError(t, os.WriteFile(certPath, serverCertPEM, 0600))
	require.NoError(t, os.WriteFile(keyPath, serverKeyPEM, 0600))
	require.NoError(t, os.WriteFile(caPath, rootCAPEM, 0600))

	config := HTTPSServerConfig{
		TLSCertFilePath: certPath,
		TLSKeyFilePath:  keyPath,
		CAFilePath:      caPath,
	}

	logger := ctrl.Log.WithName("test")
	tlsConfig, _, err := config.NewTLSConfig(logger)
	require.NoError(t, err)

	// Test that verification succeeds with correct chain order (leaf first, then intermediate)
	cs := tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{leafCert, intermediateCert},
	}

	err = tlsConfig.VerifyConnection(cs)
	require.NoError(t, err, "Should verify successfully with leaf cert first and intermediate second")

	// Test with multiple intermediates - verify they are all added to the intermediates pool
	csMultipleIntermediates := tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{leafCert, intermediateCert, intermediateCert},
	}

	err = tlsConfig.VerifyConnection(csMultipleIntermediates)
	require.NoError(t, err, "Should handle multiple intermediate certificates correctly")
}
