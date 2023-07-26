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
	"testing"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/collector"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
)

const testFileIngress = "testdata/ingress_testdata.yaml"

func TestExpectedIngresses(t *testing.T) {
	t.Run("should create and update ingress entry", func(t *testing.T) {
		ctx := context.Background()

		params, err := newParams("something:tag", testFileIngress)
		if err != nil {
			t.Fatal(err)
		}
		params.Instance.Spec.Ingress.Type = "ingress"

		err = expectedIngresses(ctx, params, []*networkingv1.Ingress{collector.DesiredIngress(params.Config, params.Log, params.Instance)})
		assert.NoError(t, err)

		nns := types.NamespacedName{Namespace: "default", Name: "test-ingress"}
		exists, err := populateObjectIfExists(t, &networkingv1.Ingress{}, nns)
		assert.NoError(t, err)
		assert.True(t, exists)

		// update fields
		const expectHostname = "something-else.com"
		params.Instance.Spec.Ingress.Annotations = map[string]string{"blub": "blob"}
		params.Instance.Spec.Ingress.Hostname = expectHostname

		err = expectedIngresses(ctx, params, []*networkingv1.Ingress{collector.DesiredIngress(params.Config, params.Log, params.Instance)})
		assert.NoError(t, err)

		got := &networkingv1.Ingress{}
		err = params.Client.Get(ctx, nns, got)
		assert.NoError(t, err)

		gotHostname := got.Spec.Rules[0].Host
		if gotHostname != expectHostname {
			t.Errorf("host name is not up-to-date. expect: %s, got: %s", expectHostname, gotHostname)
		}

		if v, ok := got.Annotations["blub"]; !ok || v != "blob" {
			t.Error("annotations are not up-to-date. Missing entry or value is invalid.")
		}
	})
}

func TestDeleteIngresses(t *testing.T) {
	t.Run("should delete excess ingress", func(t *testing.T) {
		// create
		ctx := context.Background()

		myParams, err := newParams("something:tag", testFileIngress)
		if err != nil {
			t.Fatal(err)
		}
		myParams.Instance.Spec.Ingress.Type = "ingress"

		err = expectedIngresses(ctx, myParams, []*networkingv1.Ingress{collector.DesiredIngress(myParams.Config, myParams.Log, myParams.Instance)})
		assert.NoError(t, err)

		nns := types.NamespacedName{Namespace: "default", Name: "test-ingress"}
		exists, err := populateObjectIfExists(t, &networkingv1.Ingress{}, nns)
		assert.NoError(t, err)
		assert.True(t, exists)

		// delete
		if delIngressErr := deleteIngresses(ctx, params(), []*networkingv1.Ingress{}); delIngressErr != nil {
			t.Error(delIngressErr)
		}

		// check
		exists, err = populateObjectIfExists(t, &networkingv1.Ingress{}, nns)
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestIngresses(t *testing.T) {
	t.Run("wrong mode", func(t *testing.T) {
		ctx := context.Background()
		err := Ingresses(ctx, params())
		assert.Nil(t, err)
	})

	t.Run("supported mode and service exists", func(t *testing.T) {
		ctx := context.Background()
		myParams := params()
		err := expectedServices(context.Background(), myParams, []corev1.Service{service("test-collector", params().Instance.Spec.Ports)})
		assert.NoError(t, err)

		assert.Nil(t, Ingresses(ctx, myParams))
	})

}
