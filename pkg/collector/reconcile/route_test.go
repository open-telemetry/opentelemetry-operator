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
	"strings"
	"testing"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"
)

func TestExpectedRoutes(t *testing.T) {
	t.Run("should create and update route entry", func(t *testing.T) {
		ctx := context.Background()

		params, err := newParams("something:tag", testFileIngress)
		if err != nil {
			t.Fatal(err)
		}
		params.Instance.Spec.Ingress.Type = v1alpha1.IngressTypeRoute
		params.Instance.Spec.Ingress.Route.Termination = v1alpha1.TLSRouteTerminationTypeInsecure

		routes := collector.Routes(params.Config, params.Log, params.Instance, "")
		err = expectedRoutes(ctx, params, routes)
		assert.NoError(t, err)

		nns := types.NamespacedName{Namespace: params.Instance.Namespace, Name: "otlp-grpc-test-route"}
		exists, err := populateObjectIfExists(t, &routev1.Route{}, nns)
		assert.NoError(t, err)
		assert.True(t, exists)

		// update fields
		const expectHostname = "something-else.com"
		params.Instance.Spec.Ingress.Annotations = map[string]string{"blub": "blob"}
		params.Instance.Spec.Ingress.Hostname = expectHostname

		routes = collector.Routes(params.Config, params.Log, params.Instance, "")
		err = expectedRoutes(ctx, params, routes)
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

		routes := collector.Routes(myParams.Config, myParams.Log, myParams.Instance, "")
		err = expectedRoutes(ctx, myParams, routes)
		assert.NoError(t, err)

		nns := types.NamespacedName{Namespace: "default", Name: "otlp-grpc-test-route"}
		exists, err := populateObjectIfExists(t, &routev1.Route{}, nns)
		assert.NoError(t, err)
		assert.True(t, exists)

		// delete
		if err = deleteRoutes(ctx, params(), []*routev1.Route{}); err != nil {
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
		err := expectedServices(context.Background(), myParams, []*corev1.Service{service("test-collector", params().Instance.Spec.Ports)})
		assert.NoError(t, err)

		assert.Nil(t, Routes(ctx, myParams))
	})

}
