// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestDefaultConfig(t *testing.T) {
	cfg := New()
	f, err := os.ReadFile("testdata/config.yaml")
	require.NoError(t, err)
	actual := Config{}
	require.NoError(t, yaml.Unmarshal(f, &actual))
	assert.Equal(t, cfg, actual)
}

func TestSomeChanges(t *testing.T) {
	f, err := os.ReadFile("testdata/config2.yaml")
	require.NoError(t, err)
	actual := Config{}
	require.NoError(t, yaml.Unmarshal(f, &actual))
	assert.Equal(t, "foo:1", actual.AutoInstrumentationDotNetImage)
	assert.Equal(t, "foobar:1", actual.AutoInstrumentationGoImage)
	assert.Equal(t, "bar:1", actual.AutoInstrumentationApacheHttpdImage)
}
