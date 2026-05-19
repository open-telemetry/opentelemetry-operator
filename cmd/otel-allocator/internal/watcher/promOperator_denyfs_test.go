// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"log/slog"
	"testing"
	"time"

	"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	promconfig "github.com/prometheus/prometheus/config"
	"github.com/stretchr/testify/assert"
)

func newDenyFSWatcher(deny bool) *PrometheusCRWatcher {
	return &PrometheusCRWatcher{
		denyFSAccessThroughSMs: deny,
		logger:                 slog.Default(),
	}
}

// TestFilterScrapeConfigsDropsCredentialsFile verifies that a scrape config
// with authorization.credentials_file is dropped.
func TestFilterScrapeConfigsDropsCredentialsFile(t *testing.T) {
	tw := newDenyFSWatcher(true)

	cfg := &promconfig.Config{
		ScrapeConfigs: []*promconfig.ScrapeConfig{
			{
				JobName: "unsafe",
				HTTPClientConfig: config.HTTPClientConfig{
					Authorization: &config.Authorization{
						Type:            "Bearer",
						CredentialsFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
					},
				},
			},
		},
	}

	tw.filterScrapeConfigs(cfg)
	assert.Empty(t, cfg.ScrapeConfigs)
}

// TestFilterScrapeConfigsDisabled verifies that when the guard is disabled, no
// scrape configs are dropped.
func TestFilterScrapeConfigsDisabled(t *testing.T) {
	tw := newDenyFSWatcher(false)

	cfg := &promconfig.Config{
		ScrapeConfigs: []*promconfig.ScrapeConfig{
			{
				JobName: "unsafe",
				HTTPClientConfig: config.HTTPClientConfig{
					Authorization: &config.Authorization{
						Type:            "Bearer",
						CredentialsFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
					},
				},
			},
		},
	}

	tw.filterScrapeConfigs(cfg)
	assert.Len(t, cfg.ScrapeConfigs, 1)
}

// TestFilterScrapeConfigsDropsTLSFiles verifies the guard drops scrape configs
// that set tlsConfig.caFile / certFile / keyFile.
func TestFilterScrapeConfigsDropsTLSFiles(t *testing.T) {
	for _, tc := range []struct {
		name string
		tls  config.TLSConfig
	}{
		{name: "ca_file", tls: config.TLSConfig{CAFile: "/path/to/ca.crt"}},
		{name: "cert_file", tls: config.TLSConfig{CertFile: "/path/to/cert.crt"}},
		{name: "key_file", tls: config.TLSConfig{KeyFile: "/path/to/key.key"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tw := newDenyFSWatcher(true)

			cfg := &promconfig.Config{
				ScrapeConfigs: []*promconfig.ScrapeConfig{
					{
						JobName: "tls-" + tc.name,
						HTTPClientConfig: config.HTTPClientConfig{
							TLSConfig: tc.tls,
						},
					},
				},
			}

			tw.filterScrapeConfigs(cfg)
			assert.Empty(t, cfg.ScrapeConfigs)
		})
	}
}

// TestFilterScrapeConfigsEmpty verifies that an empty scrape config list is a
// no-op.
func TestFilterScrapeConfigsEmpty(t *testing.T) {
	tw := newDenyFSWatcher(true)

	cfg := &promconfig.Config{
		ScrapeConfigs: []*promconfig.ScrapeConfig{},
	}

	tw.filterScrapeConfigs(cfg)
	assert.Empty(t, cfg.ScrapeConfigs)
}

// TestFilterScrapeConfigsKeepsSafeConfigs verifies that scrape configs without
// file references are kept.
func TestFilterScrapeConfigsKeepsSafeConfigs(t *testing.T) {
	tw := newDenyFSWatcher(true)

	cfg := &promconfig.Config{
		ScrapeConfigs: []*promconfig.ScrapeConfig{
			{
				JobName:           "safe",
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

	tw.filterScrapeConfigs(cfg)
	assert.Len(t, cfg.ScrapeConfigs, 1)
	assert.Equal(t, "safe", cfg.ScrapeConfigs[0].JobName)
}

// TestFilterScrapeConfigsKeepsSafeDropsUnsafe verifies that in a mixed list,
// only the unsafe scrape configs are dropped.
func TestFilterScrapeConfigsKeepsSafeDropsUnsafe(t *testing.T) {
	tw := newDenyFSWatcher(true)

	cfg := &promconfig.Config{
		ScrapeConfigs: []*promconfig.ScrapeConfig{
			{
				JobName: "safe",
				HTTPClientConfig: config.HTTPClientConfig{
					BasicAuth: &config.BasicAuth{
						Username: "user",
						Password: "pass",
					},
				},
			},
			{
				JobName: "unsafe-credentials",
				HTTPClientConfig: config.HTTPClientConfig{
					Authorization: &config.Authorization{
						Type:            "Bearer",
						CredentialsFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
					},
				},
			},
			{
				JobName: "unsafe-tls",
				HTTPClientConfig: config.HTTPClientConfig{
					TLSConfig: config.TLSConfig{KeyFile: "/path/to/key.key"},
				},
			},
		},
	}

	tw.filterScrapeConfigs(cfg)
	assert.Len(t, cfg.ScrapeConfigs, 1)
	assert.Equal(t, "safe", cfg.ScrapeConfigs[0].JobName)
}

// TestFilterScrapeConfigsNilAuthorization verifies nil Authorization fields do
// not cause panics or unintended drops.
func TestFilterScrapeConfigsNilAuthorization(t *testing.T) {
	tw := newDenyFSWatcher(true)

	cfg := &promconfig.Config{
		ScrapeConfigs: []*promconfig.ScrapeConfig{
			{
				JobName: "no-auth",
				HTTPClientConfig: config.HTTPClientConfig{
					Authorization: nil,
				},
			},
		},
	}

	tw.filterScrapeConfigs(cfg)
	assert.Len(t, cfg.ScrapeConfigs, 1)
}

// TestFilterScrapeConfigsEmptyTLSConfig verifies an empty TLSConfig keeps the
// scrape config.
func TestFilterScrapeConfigsEmptyTLSConfig(t *testing.T) {
	tw := newDenyFSWatcher(true)

	cfg := &promconfig.Config{
		ScrapeConfigs: []*promconfig.ScrapeConfig{
			{
				JobName: "empty-tls",
				HTTPClientConfig: config.HTTPClientConfig{
					TLSConfig: config.TLSConfig{},
				},
			},
		},
	}

	tw.filterScrapeConfigs(cfg)
	assert.Len(t, cfg.ScrapeConfigs, 1)
}
