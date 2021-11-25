package adapters

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestConfigToProbeShouldCreateProbeFor(t *testing.T) {
	tests := []struct {
		desc         string
		config       string
		expectedPort int32
		expectedPath string
	}{
		{
			desc:         "SimpleHappyPath",
			expectedPort: int32(13133),
			expectedPath: "/",
			config: `extensions:
  health_check:
service:
  extensions: [health_check]`,
		}, {
			desc:         "CustomEndpointAndPath",
			expectedPort: int32(1234),
			expectedPath: "/checkit",
			config: `extensions:
  health_check:
    endpoint: localhost:1234
    path: /checkit
service:
  extensions: [health_check]`,
		}, {
			desc:         "CustomEndpointAndDefaultPath",
			expectedPort: int32(1234),
			expectedPath: "/",
			config: `extensions:
  health_check:
    endpoint: localhost:1234
service:
  extensions: [health_check]`,
		}, {
			desc:         "CustomEndpointWithJustPortAndDefaultPath",
			expectedPort: int32(1234),
			expectedPath: "/",
			config: `extensions:
  health_check:
    endpoint: :1234
service:
  extensions: [health_check]`,
		}, {
			desc:         "DefaultEndpointAndCustomPath",
			expectedPort: int32(13133),
			expectedPath: "/checkit",
			config: `extensions:
  health_check:
    path: /checkit
service:
  extensions: [health_check]`,
		}, {
			desc:         "DefaultEndpointForUnexpectedEndpoint",
			expectedPort: int32(13133),
			expectedPath: "/",
			config: `extensions:
  health_check:
    endpoint: 0:0:0"
service:
  extensions: [health_check]`,
		}, {
			desc:         "DefaultEndpointForUnparseablendpoint",
			expectedPort: int32(13133),
			expectedPath: "/",
			config: `extensions:
  health_check:
    endpoint:
      this: should-not-be-a-map"
service:
  extensions: [health_check]`,
		}, {
			desc: "WillUseSecondServiceExtension",
			config: `extensions:
  health_check:
service:
  extensions: [health_check/1, health_check]`,
			expectedPort: int32(13133),
			expectedPath: "/",
		},
	}

	for _, test := range tests {
		// prepare
		config, err := ConfigFromString(test.config)
		require.NoError(t, err, test.desc)
		require.NotEmpty(t, config, test.desc)

		// test
		actualProbe, err := ConfigToContainerProbe(config)
		assert.NoError(t, err)
		assert.Equal(t, test.expectedPath, actualProbe.HTTPGet.Path, test.desc)
		assert.Equal(t, test.expectedPort, actualProbe.HTTPGet.Port.IntVal, test.desc)
		assert.Equal(t, "", actualProbe.HTTPGet.Host, test.desc)
	}
}

func TestConfigToProbeShouldErrorIf(t *testing.T) {
	tests := []struct {
		desc        string
		config      string
		expectedErr error
	}{
		{
			desc: "NoHealthCheckExtension",
			config: `extensions:
  pprof:
service:
  extensions: [health_check]`,
			expectedErr: errNoExtensionHealthCheck,
		}, {
			desc: "BadlyFormattedExtensions",
			config: `extensions: [hi]
service:
  extensions: [health_check]`,
			expectedErr: errExtensionsNotAMap,
		}, {
			desc: "NoExtensions",
			config: `service:
  extensions: [health_check]`,
			expectedErr: errNoExtensions,
		}, {
			desc: "NoHealthCheckInServiceExtensions",
			config: `service:
  extensions: [pprof]`,
			expectedErr: errNoServiceExtensionHealthCheck,
		}, {
			desc: "BadlyFormattedServiceExtensions",
			config: `service:
  extensions:
    this: should-not-be-a-map`,
			expectedErr: errServiceExtensionsNotSlice,
		}, {
			desc: "NoServiceExtensions",
			config: `service:
  pipelines:
    traces:
      receivers: [otlp]`,
			expectedErr: errNoServiceExtensions,
		}, {
			desc: "BadlyFormattedService",
			config: `extensions:
  health_check:
service: [hi]`,
			expectedErr: errServiceNotAMap,
		}, {
			desc: "NoService",
			config: `extensions:
  health_check:`,
			expectedErr: errNoService,
		},
	}

	for _, test := range tests {
		// prepare
		config, err := ConfigFromString(test.config)
		require.NoError(t, err, test.desc)
		require.NotEmpty(t, config, test.desc)

		// test
		_, err = ConfigToContainerProbe(config)
		assert.Equal(t, test.expectedErr, err, test.desc)
	}
}
