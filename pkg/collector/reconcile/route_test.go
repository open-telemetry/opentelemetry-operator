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

package reconcile

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"testing"

	"github.com/open-telemetry/opentelemetry-operator/internal/reconcileutil"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
)

func TestDesiredRoutes(t *testing.T) {
	t.Run("should return nil invalid ingress type", func(t *testing.T) {
		params := reconcileutil.Params{
			Config: config.Config{},
			Client: k8sClient,
			Log:    logger,
			Instance: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Ingress: v1alpha1.Ingress{
						Type: v1alpha1.IngressType("unknown"),
					},
				},
			},
		}

		actual := desiredRoutes(context.Background(), params)
		assert.Nil(t, actual)
	})

	t.Run("should return nil unable to parse config", func(t *testing.T) {
		params := reconcileutil.Params{
			Config: config.Config{},
			Client: k8sClient,
			Log:    logger,
			Instance: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Config: "!!!",
					Ingress: v1alpha1.Ingress{
						Type: v1alpha1.IngressTypeRoute,
					},
				},
			},
		}

		actual := desiredRoutes(context.Background(), params)
		assert.Nil(t, actual)
	})

	t.Run("should return nil unable to parse receiver ports", func(t *testing.T) {
		params := reconcileutil.Params{
			Config: config.Config{},
			Client: k8sClient,
			Log:    logger,
			Instance: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Config: "---",
					Ingress: v1alpha1.Ingress{
						Type: v1alpha1.IngressTypeRoute,
					},
				},
			},
		}

		actual := desiredRoutes(context.Background(), params)
		assert.Nil(t, actual)
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

		params.Instance.Namespace = ns
		params.Instance.Spec.Ingress = v1alpha1.Ingress{
			Type:        v1alpha1.IngressTypeRoute,
			Hostname:    hostname,
			Annotations: map[string]string{"some.key": "some.value"},
			Route: v1alpha1.OpenShiftRoute{
				Termination: v1alpha1.TLSRouteTerminationTypeInsecure,
			},
		}

		got := desiredRoutes(context.Background(), params)[0]

		assert.NotEqual(t, &routev1.Route{
			ObjectMeta: metav1.ObjectMeta{
				Name:        naming.Route(params.Instance, ""),
				Namespace:   ns,
				Annotations: params.Instance.Spec.Ingress.Annotations,
				Labels: map[string]string{
					"app.kubernetes.io/name":       naming.Route(params.Instance, ""),
					"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.Instance.Namespace, params.Instance.Name),
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
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
}

func TestExpectedRoutes(t *testing.T) {
	t.Run("should create and update route entry", func(t *testing.T) {
		ctx := context.Background()

		params, err := newParams("something:tag", testFileIngress)
		if err != nil {
			t.Fatal(err)
		}
		params.Instance.Spec.Ingress.Type = v1alpha1.IngressTypeRoute
		params.Instance.Spec.Ingress.Route.Termination = v1alpha1.TLSRouteTerminationTypeInsecure

		err = expectedRoutes(ctx, params, desiredRoutes(ctx, params))
		assert.NoError(t, err)

		nns := types.NamespacedName{Namespace: params.Instance.Namespace, Name: "otlp-grpc-test-route"}
		exists, err := populateObjectIfExists(t, &routev1.Route{}, nns)
		assert.NoError(t, err)
		assert.True(t, exists)

		// update fields
		const expectHostname = "something-else.com"
		params.Instance.Spec.Ingress.Annotations = map[string]string{"blub": "blob"}
		params.Instance.Spec.Ingress.Hostname = expectHostname

		err = expectedRoutes(ctx, params, desiredRoutes(ctx, params))
		assert.NoError(t, err)

		got := &routev1.Route{}
		err = params.Client.Get(ctx, nns, got)
		assert.NoError(t, err)

		gotHostname := got.Spec.Host
		if !strings.Contains(gotHostname, got.Spec.Host) {
			t.Errorf("host name is not up-to-date. expect: %s, got: %s", expectHostname, gotHostname)
		}

		if v, ok := got.Annotations["blub"]; !ok || v != "blob" {
			t.Error("annotations are not up-to-date. Missing entry or value is invalid.")
		}
	})
}

func TestDeleteRoutes(t *testing.T) {
	t.Run("should delete excess routes", func(t *testing.T) {
		// create
		ctx := context.Background()

		myParams, err := newParams("something:tag", testFileIngress)
		if err != nil {
			t.Fatal(err)
		}
		myParams.Instance.Spec.Ingress.Type = v1alpha1.IngressTypeRoute

		err = expectedRoutes(ctx, myParams, desiredRoutes(ctx, myParams))
		assert.NoError(t, err)

		nns := types.NamespacedName{Namespace: "default", Name: "otlp-grpc-test-route"}
		exists, err := populateObjectIfExists(t, &routev1.Route{}, nns)
		assert.NoError(t, err)
		assert.True(t, exists)

		// delete
		if err = deleteRoutes(ctx, params(), []routev1.Route{}); err != nil {
			t.Error(err)
		}

		// check
		exists, err = populateObjectIfExists(t, &routev1.Route{}, nns)
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestRoutes(t *testing.T) {
	t.Run("wrong mode", func(t *testing.T) {
		ctx := context.Background()
		err := Routes(ctx, params())
		assert.Nil(t, err)
	})

	t.Run("supported mode and service exists", func(t *testing.T) {
		ctx := context.Background()
		myParams := params()
		err := expectedServices(context.Background(), myParams, []corev1.Service{service("test-collector", params().Instance.Spec.Ports)})
		assert.NoError(t, err)

		assert.Nil(t, Routes(ctx, myParams))
	})

}
