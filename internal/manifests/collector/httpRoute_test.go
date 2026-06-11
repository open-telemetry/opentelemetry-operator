// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func TestDesiredHTTPRoutes(t *testing.T) {
	t.Run("should return nil when HTTPRoute is not enabled", func(t *testing.T) {
		params := manifests.Params{
			Config: config.Config{},
			Log:    testLogger,
			OtelCol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					HttpRoute: v1beta1.HttpRouteConfig{
						Enabled: false,
					},
				},
			},
		}

		actual, err := HTTPRoutes(params)
		assert.NoError(t, err)
		assert.Nil(t, actual)
	})

	t.Run("should return nil when HTTPRoute config is not provided", func(t *testing.T) {
		params := manifests.Params{
			Config: config.Config{},
			Log:    testLogger,
			OtelCol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					HttpRoute: v1beta1.HttpRouteConfig{
						Enabled: false,
					},
				},
			},
		}

		actual, err := HTTPRoutes(params)
		assert.NoError(t, err)
		assert.Nil(t, actual)
	})

	t.Run("should return nil for sidecar mode", func(t *testing.T) {
		params := manifests.Params{
			Config: config.Config{},
			Log:    testLogger,
			OtelCol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: v1beta1.ModeSidecar,
					HttpRoute: v1beta1.HttpRouteConfig{
						Enabled: true,
					},
				},
			},
		}

		actual, err := HTTPRoutes(params)
		assert.NoError(t, err)
		assert.Nil(t, actual)
	})

	t.Run("should return nil when gateway name is not specified", func(t *testing.T) {
		params := manifests.Params{
			Config: config.Config{},
			Log:    testLogger,
			OtelCol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					HttpRoute: v1beta1.HttpRouteConfig{
						Enabled: true,
					},
				},
			},
		}

		actual, err := HTTPRoutes(params)
		assert.Nil(t, actual)
		assert.NoError(t, err)
	})

	t.Run("should create HTTPRoutes with valid configuration", func(t *testing.T) {
		var (
			ns          = "test"
			gatewayName = "example-gateway"
			gatewayNs   = "gateway-system"
			hostnames   = []string{"otel.example.com"}
		)

		params, err := newParams("something:tag", testFileIngress, nil)
		require.NoError(t, err)

		params.OtelCol.Namespace = ns
		params.OtelCol.Spec.HttpRoute = v1beta1.HttpRouteConfig{
			Enabled:          true,
			Gateway:          gatewayName,
			GatewayNamespace: gatewayNs,
			Hostnames:        hostnames,
		}

		httpRoutes, err := HTTPRoutes(params)
		require.NoError(t, err)
		require.NotNil(t, httpRoutes)
		require.Greater(t, len(httpRoutes), 0)

		got := httpRoutes[0]
		assert.Equal(t, ns, got.Namespace)
		assert.NotEmpty(t, got.Name)
		assert.Contains(t, got.Name, "httproute")

		require.Len(t, got.Spec.ParentRefs, 1)
		assert.Equal(t, gatewayv1.ObjectName(gatewayName), got.Spec.ParentRefs[0].Name)
		assert.Equal(t, gatewayv1.Namespace(gatewayNs), *got.Spec.ParentRefs[0].Namespace)

		assert.Equal(t, gatewayv1.Group("gateway.networking.k8s.io"), *got.Spec.ParentRefs[0].Group)
		assert.Equal(t, gatewayv1.Kind("Gateway"), *got.Spec.ParentRefs[0].Kind)

		require.Len(t, got.Spec.Hostnames, 1)
		assert.Equal(t, gatewayv1.Hostname(hostnames[0]), got.Spec.Hostnames[0])

		require.Greater(t, len(got.Spec.Rules), 0)
		rule := got.Spec.Rules[0]
		require.Greater(t, len(rule.Matches), 0)
		require.NotNil(t, rule.Matches[0].Path)
		assert.NotEmpty(t, *rule.Matches[0].Path.Value)

		require.Greater(t, len(rule.BackendRefs), 0)
		backendRef := rule.BackendRefs[0]
		assert.Equal(t, gatewayv1.ObjectName(naming.Service(params.OtelCol.Name)), backendRef.Name)
		assert.NotNil(t, backendRef.Port)
	})
}
