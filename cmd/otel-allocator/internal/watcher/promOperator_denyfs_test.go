// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"strings"
	"testing"
	"time"

	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/stretchr/testify/assert"
)

// TestValidateAndFilterScrapeConfigsWithCredentialsFile verifies that the guard
// rejects scrape configs with authorization.credentials_file.
func TestValidateAndFilterScrapeConfigsWithCredentialsFile(t *testing.T) {
	tw := &PrometheusCRWatcher{
		denyFSAccessThroughSMs: true,
	}

	got := &promconfig.Config{
		ScrapeConfigs: []*promconfig.ScrapeConfig{
			{
				JobName: "test",
				HTTPClientConfig: config.HTTPClientConfig{
					Authorization: &config.Authorization{
						Type:            "Bearer",
						CredentialsFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
					},
				},
			},
		},
	}

	err := tw.validateAndFilterScrapeConfigs(got)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "credentials_file")
}

// TestValidateAndFilterScrapeConfigsDisabled verifies that when the guard is
// disabled, it does not reject scrape configs.
func TestValidateAndFilterScrapeConfigsDisabled(t *testing.T) {
	tw := &PrometheusCRWatcher{
		denyFSAccessThroughSMs: false,
	}

	got := &promconfig.Config{
		ScrapeConfigs: []*promconfig.ScrapeConfig{
			{
				JobName: "test",
				HTTPClientConfig: config.HTTPClientConfig{
					Authorization: &config.Authorization{
						Type:            "Bearer",
						CredentialsFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
					},
				},
			},
		},
	}

	err := tw.validateAndFilterScrapeConfigs(got)
	assert.NoError(t, err)
}

// TestValidateAndFilterScrapeConfigsWithTLSFiles verifies the guard also
// rejects tlsConfig.caFile, certFile, and keyFile.
func TestValidateAndFilterScrapeConfigsWithTLSFiles(t *testing.T) {
	tw := &PrometheusCRWatcher{
		denyFSAccessThroughSMs: true,
	}

	got := &promconfig.Config{
		ScrapeConfigs: []*promconfig.ScrapeConfig{
			{
				JobName: "test",
				HTTPClientConfig: config.HTTPClientConfig{
					TLSConfig: config.TLSConfig{
						CAFile:   "/path/to/ca.crt",
						CertFile: "/path/to/cert.crt",
						KeyFile:  "/path/to/key.key",
					},
				},
			},
		},
	}

	err := tw.validateAndFilterScrapeConfigs(got)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ca_file")
}

// TestValidateAndFilterScrapeConfigsWithTLSFilesCert verifies cert_file is
// detected.
func TestValidateAndFilterScrapeConfigsWithTLSFilesCert(t *testing.T) {
	tw := &PrometheusCRWatcher{
		denyFSAccessThroughSMs: true,
	}

	got := &promconfig.Config{
		ScrapeConfigs: []*promconfig.ScrapeConfig{
			{
				JobName: "test",
				HTTPClientConfig: config.HTTPClientConfig{
					TLSConfig: config.TLSConfig{
						CertFile: "/path/to/cert.crt",
					},
				},
			},
		},
	}

	err := tw.validateAndFilterScrapeConfigs(got)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cert_file")
}

// TestValidateAndFilterScrapeConfigsWithTLSFilesKey verifies key_file is
// detected.
func TestValidateAndFilterScrapeConfigsWithTLSFilesKey(t *testing.T) {
	tw := &PrometheusCRWatcher{
		denyFSAccessThroughSMs: true,
	}

	got := &promconfig.Config{
		ScrapeConfigs: []*promconfig.ScrapeConfig{
			{
				JobName: "test",
				HTTPClientConfig: config.HTTPClientConfig{
					TLSConfig: config.TLSConfig{
						KeyFile: "/path/to/key.key",
					},
				},
			},
		},
	}

	err := tw.validateAndFilterScrapeConfigs(got)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key_file")
}

// TestValidateAndFilterScrapeConfigsWithEmptyScrapeConfigs verifies that
// empty scrape configs do not cause errors.
func TestValidateAndFilterScrapeConfigsWithEmptyScrapeConfigs(t *testing.T) {
	tw := &PrometheusCRWatcher{
		denyFSAccessThroughSMs: true,
	}

	got := &promconfig.Config{
		ScrapeConfigs: []*promconfig.ScrapeConfig{},
	}

	err := tw.validateAndFilterScrapeConfigs(got)
	assert.NoError(t, err)
}

// TestValidateAndFilterScrapeConfigsWithNormalPrometheusConfig verifies
// that normal scrape configs without file references are allowed through.
func TestValidateAndFilterScrapeConfigsWithNormalPrometheusConfig(t *testing.T) {
	tw := &PrometheusCRWatcher{
		denyFSAccessThroughSMs: true,
	}

	got := &promconfig.Config{
		ScrapeConfigs: []*promconfig.ScrapeConfig{
			{
				JobName:           "test",
				MetricsPath:       "/metrics",
				ScrapeInterval:    model.Duration(30 * time.Second),
				EnableCompression: true,
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "user",
						Password: "password",
					},
				},
			},
		},
	}

	err := tw.validateAndFilterScrapeConfigs(got)
	assert.NoError(t, err)
}

// TestValidateAndFilterScrapeConfigsWithBearerTokenFileRejection tests the
// exact attack scenario: bearerTokenFile pointing to the Collector's service
// account token should be rejected.
func TestValidateAndFilterScrapeConfigsWithBearerTokenFileRejection(t *testing.T) {
	tw := &PrometheusCRWatcher{
		denyFSAccessThroughSMs: true,
	}

	got := &promconfig.Config{
		ScrapeConfigs: []*promconfig.ScrapeConfig{
			{
				JobName: "test",
				HTTPClientConfig: config.HTTPClientConfig{
					Authorization: &config.Authorization{
						Type:            "Bearer",
						CredentialsFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
					},
				},
			},
		},
	}

	err := tw.validateAndFilterScrapeConfigs(got)
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "credentials_file"),
		"expected error to mention credentials_file")
	assert.True(t, strings.Contains(err.Error(),
		"/var/run/secrets/kubernetes.io/serviceaccount/token"),
		"expected error to mention the token path")
}

// TestValidateAndFilterScrapeConfigsMultipleScrapeConfigs verifies that
// the guard checks all scrape configs in a list.
func TestValidateAndFilterScrapeConfigsMultipleScrapeConfigs(t *testing.T) {
	tw := &PrometheusCRWatcher{
		denyFSAccessThroughSMs: true,
	}

	got := &promconfig.Config{
		ScrapeConfigs: []*promconfig.ScrapeConfig{
			{
				JobName:           "safe-config",
				EnableCompression: true,
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "user",
						Password: "pass",
					},
				},
			},
			{
				JobName:           "unsafe-config",
				EnableCompression: true,
				HTTPClientConfig: config.HTTPClientConfig{
					Authorization: &config.Authorization{
						Type:            "Bearer",
						CredentialsFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
					},
				},
			},
		},
	}

	err := tw.validateAndFilterScrapeConfigs(got)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "credentials_file")
}

// TestValidateAndFilterScrapeConfigsWithNilAuthorization verifies that nil
// Authorization fields do not cause panics.
func TestValidateAndFilterScrapeConfigsWithNilAuthorization(t *testing.T) {
	tw := &PrometheusCRWatcher{
		denyFSAccessThroughSMs: true,
	}

	got := &promconfig.Config{
		ScrapeConfigs: []*promconfig.ScrapeConfig{
			{
				JobName:           "test",
				EnableCompression: true,
				HTTPClientConfig: config.HTTPClientConfig{
					Authorization: nil,
				},
			},
		},
	}

	err := tw.validateAndFilterScrapeConfigs(got)
	assert.NoError(t, err)
}

// TestValidateAndFilterScrapeConfigsWithNilTLSConfig verifies that nil
// TLSConfig fields do not cause panics.
func TestValidateAndFilterScrapeConfigsWithNilTLSConfig(t *testing.T) {
	tw := &PrometheusCRWatcher{
		denyFSAccessThroughSMs: true,
	}

	got := &promconfig.Config{
		ScrapeConfigs: []*promconfig.ScrapeConfig{
			{
				JobName:           "test",
				EnableCompression: true,
				HTTPClientConfig: config.HTTPClientConfig{
					TLSConfig: config.TLSConfig{},
				},
			},
		},
	}

	err := tw.validateAndFilterScrapeConfigs(got)
	assert.NoError(t, err)
}
