// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	_ "embed"
	"fmt"
	"testing"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

func TestDesiredRoutes(t *testing.T) {
	t.Run("should return nil invalid ingress type", func(t *testing.T) {
		params := manifests.Params{
			Config: config.Config{},
			Log:    logger,
			OtelCol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Ingress: v1alpha1.Ingress{
						Type: v1alpha1.IngressType("unknown"),
					},
				},
			},
		}

		actual, err := Routes(params)
		assert.NoError(t, err)
		assert.Nil(t, actual)
	})

	t.Run("should return nil unable to parse config", func(t *testing.T) {
		params := manifests.Params{
			Config: config.Config{},
			Log:    logger,
			OtelCol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Config: "!!!",
					Ingress: v1alpha1.Ingress{
						Type: v1alpha1.IngressTypeRoute,
						Route: v1alpha1.OpenShiftRoute{
							Termination: v1alpha1.TLSRouteTerminationTypeInsecure,
						},
					},
				},
			},
		}

		actual, err := Routes(params)
		assert.Nil(t, actual)
		assert.ErrorContains(t, err, "couldn't parse the opentelemetry-collector configuration")
	})

	t.Run("should return nil unable to parse receiver ports", func(t *testing.T) {
		params := manifests.Params{
			Config: config.Config{},
			Log:    logger,
			OtelCol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Config: "---",
					Ingress: v1alpha1.Ingress{
						Type: v1alpha1.IngressTypeRoute,
						Route: v1alpha1.OpenShiftRoute{
							Termination: v1alpha1.TLSRouteTerminationTypeInsecure,
						},
					},
				},
			},
		}

		actual, err := Routes(params)
		assert.Nil(t, actual)
		assert.ErrorContains(t, err, "no receivers available as part of the configuration")
	})

	t.Run("should return nil unable to do something else", func(t *testing.T) {
		var (
			ns       = "test"
			hostname = "example.com"
		)

		params, err := newParams("something:tag", testFileIngress)
		if err != nil {
			t.Fatal(err)
		}

		params.OtelCol.Namespace = ns
		params.OtelCol.Spec.Ingress = v1alpha1.Ingress{
			Type:        v1alpha1.IngressTypeRoute,
			Hostname:    hostname,
			Annotations: map[string]string{"some.key": "some.value"},
			Route: v1alpha1.OpenShiftRoute{
				Termination: v1alpha1.TLSRouteTerminationTypeInsecure,
			},
		}

		routes, err := Routes(params)
		assert.NoError(t, err)
		got := routes[0]

		assert.NotEqual(t, &routev1.Route{
			ObjectMeta: metav1.ObjectMeta{
				Name:        naming.Route(params.OtelCol.Name, ""),
				Namespace:   ns,
				Annotations: params.OtelCol.Spec.Ingress.Annotations,
				Labels: map[string]string{
					"app.kubernetes.io/name":       naming.Route(params.OtelCol.Name, ""),
					"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.OtelCol.Namespace, params.OtelCol.Name),
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
					"app.kubernetes.io/component":  "opentelemetry-collector",
				},
			},
			Spec: routev1.RouteSpec{
				Host: hostname,
				Path: "/abc",
				To: routev1.RouteTargetReference{
					Kind: "service",
					Name: "test-collector",
				},
				Port: &routev1.RoutePort{
					TargetPort: intstr.FromString("another-port"),
				},
				WildcardPolicy: routev1.WildcardPolicyNone,
				TLS: &routev1.TLSConfig{
					Termination:                   routev1.TLSTerminationPassthrough,
					InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyAllow,
				},
			},
		}, got)
	})
	t.Run("hostname is set", func(t *testing.T) {
		params, err := newParams("something:tag", testFileIngress)
		if err != nil {
			t.Fatal(err)
		}

		params.OtelCol.Namespace = "test"
		params.OtelCol.Spec.Ingress = v1alpha1.Ingress{
			Hostname: "example.com",
			Type:     v1alpha1.IngressTypeRoute,
			Route: v1alpha1.OpenShiftRoute{
				Termination: v1alpha1.TLSRouteTerminationTypeInsecure,
			},
		}

		routes, err := Routes(params)
		assert.NoError(t, err)
		require.Equal(t, 3, len(routes))
		assert.Equal(t, "web.example.com", routes[0].Spec.Host)
		assert.Equal(t, "otlp-grpc.example.com", routes[1].Spec.Host)
		assert.Equal(t, "otlp-test-grpc.example.com", routes[2].Spec.Host)
	})
	t.Run("hostname is not set", func(t *testing.T) {
		params, err := newParams("something:tag", testFileIngress)
		if err != nil {
			t.Fatal(err)
		}

		params.OtelCol.Namespace = "test"
		params.OtelCol.Spec.Ingress = v1alpha1.Ingress{
			Type: v1alpha1.IngressTypeRoute,
			Route: v1alpha1.OpenShiftRoute{
				Termination: v1alpha1.TLSRouteTerminationTypeInsecure,
			},
		}

		routes, err := Routes(params)
		assert.NoError(t, err)
		require.Equal(t, 3, len(routes))
		assert.Equal(t, "", routes[0].Spec.Host)
		assert.Equal(t, "", routes[1].Spec.Host)
		assert.Equal(t, "", routes[2].Spec.Host)
	})
}

func TestRoutes(t *testing.T) {
	t.Run("wrong mode", func(t *testing.T) {
		params := deploymentParams()
		routes, err := Routes(params)
		assert.NoError(t, err)
		assert.Nil(t, routes)
	})

	t.Run("supported mode and service exists", func(t *testing.T) {
		params := deploymentParams()
		routes, err := Routes(params)
		assert.NoError(t, err)
		assert.Nil(t, routes)
	})

}
