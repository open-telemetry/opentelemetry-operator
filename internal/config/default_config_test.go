// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaultCollectorConfig(t *testing.T) {
	t.Run("empty path is a no-op", func(t *testing.T) {
		c := &Config{}
		require.NoError(t, c.loadDefaultCollectorConfig())
		assert.Nil(t, c.Internal.DefaultCollectorConfig)
	})

	t.Run("parses a config file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "default.yaml")
		content := `
receivers:
  otlp:
    protocols:
      grpc: {}
processors:
  batch: {}
exporters:
  debug: {}
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug]
`
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

		c := &Config{DefaultCollectorConfigFile: path}
		require.NoError(t, c.loadDefaultCollectorConfig())
		require.NotNil(t, c.Internal.DefaultCollectorConfig)

		cfg := c.Internal.DefaultCollectorConfig
		assert.Contains(t, cfg.Receivers.Object, "otlp")
		require.NotNil(t, cfg.Processors)
		assert.Contains(t, cfg.Processors.Object, "batch")
		assert.Contains(t, cfg.Exporters.Object, "debug")
		require.Contains(t, cfg.Service.Pipelines, "traces")
		assert.Equal(t, []string{"batch"}, cfg.Service.Pipelines["traces"].Processors)
	})

	t.Run("missing file returns an error", func(t *testing.T) {
		c := &Config{DefaultCollectorConfigFile: filepath.Join(t.TempDir(), "does-not-exist.yaml")}
		assert.Error(t, c.loadDefaultCollectorConfig())
	})

	t.Run("invalid yaml returns an error", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad.yaml")
		require.NoError(t, os.WriteFile(path, []byte("\tnot: [valid"), 0o600))

		c := &Config{DefaultCollectorConfigFile: path}
		assert.Error(t, c.loadDefaultCollectorConfig())
	})
}
