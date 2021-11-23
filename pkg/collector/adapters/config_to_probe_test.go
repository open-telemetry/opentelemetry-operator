package adapters_test

import (
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSimpleCase(t *testing.T) {
	configStr := `extensions:
  health_check:
service:
  extensions: [health_check]`

	// prepare
	config, err := adapters.ConfigFromString(configStr)
	require.NoError(t, err)
	require.NotEmpty(t, config)

	// test
	actualProbe, err := adapters.ConfigToContainerProbe(logger, config)
	assert.NoError(t, err)
	assert.Equal(t, "/", actualProbe.HTTPGet.Path)
	assert.Equal(t, int32(13133), actualProbe.HTTPGet.Port.IntVal)
	assert.Equal(t, "0.0.0.0", actualProbe.HTTPGet.Host)
}

func TestShouldUseCustomEndpointAndPath(t *testing.T) {
	configStr := `extensions:
  health_check:
    endpoint: localhost:1234
    path: /checkit
service:
  extensions: [health_check]`

	// prepare
	config, err := adapters.ConfigFromString(configStr)
	require.NoError(t, err)
	require.NotEmpty(t, config)

	// test
	actualProbe, err := adapters.ConfigToContainerProbe(logger, config)
	assert.NoError(t, err)
	assert.Equal(t, "/checkit", actualProbe.HTTPGet.Path)
	assert.Equal(t, int32(1234), actualProbe.HTTPGet.Port.IntVal)
	assert.Equal(t, "localhost", actualProbe.HTTPGet.Host)
}

func TestShouldUseCustomEndpointAndDefaultPath(t *testing.T) {
	configStr := `extensions:
  health_check:
    endpoint: localhost:1234
service:
  extensions: [health_check]`

	// prepare
	config, err := adapters.ConfigFromString(configStr)
	require.NoError(t, err)
	require.NotEmpty(t, config)

	// test
	actualProbe, err := adapters.ConfigToContainerProbe(logger, config)
	assert.NoError(t, err)
	assert.Equal(t, "/", actualProbe.HTTPGet.Path)
	assert.Equal(t, int32(1234), actualProbe.HTTPGet.Port.IntVal)
	assert.Equal(t, "localhost", actualProbe.HTTPGet.Host)
}

func TestShouldUseDefaultEndpointAndCustomPath(t *testing.T) {
	configStr := `extensions:
  health_check:
    path: /checkit
service:
  extensions: [health_check]`

	// prepare
	config, err := adapters.ConfigFromString(configStr)
	require.NoError(t, err)
	require.NotEmpty(t, config)

	// test
	actualProbe, err := adapters.ConfigToContainerProbe(logger, config)
	assert.NoError(t, err)
	assert.Equal(t, "/checkit", actualProbe.HTTPGet.Path)
	assert.Equal(t, int32(13133), actualProbe.HTTPGet.Port.IntVal)
	assert.Equal(t, "0.0.0.0", actualProbe.HTTPGet.Host)
}

func TestShouldUseDefaultEndpointForUnexpectedEndpoint(t *testing.T) {
	configStr := `extensions:
  health_check:
    endpoint: 0:0:0"
service:
  extensions: [health_check]`

	// prepare
	config, err := adapters.ConfigFromString(configStr)
	require.NoError(t, err)
	require.NotEmpty(t, config)

	// test
	actualProbe, err := adapters.ConfigToContainerProbe(logger, config)
	assert.NoError(t, err)
	assert.Equal(t, "/", actualProbe.HTTPGet.Path)
	assert.Equal(t, int32(13133), actualProbe.HTTPGet.Port.IntVal)
	assert.Equal(t, "0.0.0.0", actualProbe.HTTPGet.Host)
}

func TestShouldErrorIfNoService(t *testing.T) {
	configStr := `extensions:
  health_check:`

	// prepare
	config, err := adapters.ConfigFromString(configStr)
	require.NoError(t, err)
	require.NotEmpty(t, config)

	// test
	_, err = adapters.ConfigToContainerProbe(logger, config)
	assert.Equal(t, adapters.ErrNoService, err)
}

func TestShouldErrorIfBadlyFormattedService(t *testing.T) {
	configStr := `extensions:
  health_check:
service: [hi]`

	// prepare
	config, err := adapters.ConfigFromString(configStr)
	require.NoError(t, err)
	require.NotEmpty(t, config)

	// test
	_, err = adapters.ConfigToContainerProbe(logger, config)
	assert.Equal(t, adapters.ErrServiceNotAMap, err)
}

func TestShouldErrorIfNoServiceExtensions(t *testing.T) {
	configStr := `service:
  pipelines:
    traces:
      receivers: [otlp]`

	// prepare
	config, err := adapters.ConfigFromString(configStr)
	require.NoError(t, err)
	require.NotEmpty(t, config)

	// test
	_, err = adapters.ConfigToContainerProbe(logger, config)
	assert.Equal(t, adapters.ErrNoServiceExtensions, err)
}

func TestShouldErrorIfBadlyFormattedServiceExtensions(t *testing.T) {
	configStr := `service:
  extensions:
    this: should-not-be-a-map`

	// prepare
	config, err := adapters.ConfigFromString(configStr)
	require.NoError(t, err)
	require.NotEmpty(t, config)

	// test
	_, err = adapters.ConfigToContainerProbe(logger, config)
	assert.Equal(t, adapters.ErrServiceExtensionsNotSlice, err)
}

func TestShouldErrorIfNoHealthCheckInServiceExtensions(t *testing.T) {
	configStr := `service:
  extensions: [pprof]`

	// prepare
	config, err := adapters.ConfigFromString(configStr)
	require.NoError(t, err)
	require.NotEmpty(t, config)

	// test
	_, err = adapters.ConfigToContainerProbe(logger, config)
	assert.Equal(t, adapters.ErrNoServiceExtensionHealthCheck, err)
}

func TestShouldErrorIfNoExtensions(t *testing.T) {
	configStr := `service:
  extensions: [health_check]`

	// prepare
	config, err := adapters.ConfigFromString(configStr)
	require.NoError(t, err)
	require.NotEmpty(t, config)

	// test
	_, err = adapters.ConfigToContainerProbe(logger, config)
	assert.Equal(t, adapters.ErrNoExtensions, err)
}

func TestShouldErrorIfBadlyFormattedExtensions(t *testing.T) {
	configStr := `extensions: [hi]
service:
  extensions: [health_check]`

	// prepare
	config, err := adapters.ConfigFromString(configStr)
	require.NoError(t, err)
	require.NotEmpty(t, config)

	// test
	_, err = adapters.ConfigToContainerProbe(logger, config)
	assert.Equal(t, adapters.ErrExtensionsNotAMap, err)
}

func TestShouldErrorIfNoHealthCheckExtension(t *testing.T) {
	configStr := `extensions:
  pprof:
service:
  extensions: [health_check]`

	// prepare
	config, err := adapters.ConfigFromString(configStr)
	require.NoError(t, err)
	require.NotEmpty(t, config)

	// test
	_, err = adapters.ConfigToContainerProbe(logger, config)
	assert.Equal(t, adapters.ErrNoExtensionHealthCheck, err)
}

func TestShouldErrorIfNoHealthCheckExtension_mustMatchFirstHealthCheck(t *testing.T) {
	configStr := `extensions:
  health_check:
service:
  extensions: [health_check/1, health_check]`

	// prepare
	config, err := adapters.ConfigFromString(configStr)
	require.NoError(t, err)
	require.NotEmpty(t, config)

	// test
	_, err = adapters.ConfigToContainerProbe(logger, config)
	assert.Equal(t, adapters.ErrNoExtensionHealthCheck, err)
}
