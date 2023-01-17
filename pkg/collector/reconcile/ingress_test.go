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
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
)

const testFileIngress = "../testdata/ingress_testdata.yaml"

func TestDesiredIngresses(t *testing.T) {
	t.Run("should return nil invalid ingress type", func(t *testing.T) {
		params := Params{
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

		actual := desiredIngresses(context.Background(), params)
		assert.Nil(t, actual)
	})

	t.Run("should return nil unable to parse config", func(t *testing.T) {
		params := Params{
			Config: config.Config{},
			Client: k8sClient,
			Log:    logger,
			Instance: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Config: "!!!",
					Ingress: v1alpha1.Ingress{
						Type: v1alpha1.IngressTypeNginx,
					},
				},
			},
		}

		actual := desiredIngresses(context.Background(), params)
		assert.Nil(t, actual)
	})

	t.Run("should return nil unable to parse receiver ports", func(t *testing.T) {
		params := Params{
			Config: config.Config{},
			Client: k8sClient,
			Log:    logger,
			Instance: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Config: "---",
					Ingress: v1alpha1.Ingress{
						Type: v1alpha1.IngressTypeNginx,
					},
				},
			},
		}

		actual := desiredIngresses(context.Background(), params)
		assert.Nil(t, actual)
	})

	t.Run("should return nil unable to do something else", func(t *testing.T) {
		var (
			ns               = "test"
			hostname         = "example.com"
			ingressClassName = "nginx"
		)

		params, err := newParams("something:tag", testFileIngress)
		if err != nil {
			t.Fatal(err)
		}

		params.Instance.Namespace = ns
		params.Instance.Spec.Ingress = v1alpha1.Ingress{
			Type:             v1alpha1.IngressTypeNginx,
			Hostname:         hostname,
			Annotations:      map[string]string{"some.key": "some.value"},
			IngressClassName: &ingressClassName,
		}

		got := desiredIngresses(context.Background(), params)
		pathType := networkingv1.PathTypePrefix

		assert.NotEqual(t, &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:        naming.Ingress(params.Instance),
				Namespace:   ns,
				Annotations: params.Instance.Spec.Ingress.Annotations,
				Labels: map[string]string{
					"app.kubernetes.io/name":       naming.Ingress(params.Instance),
					"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.Instance.Namespace, params.Instance.Name),
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
				},
			},
			Spec: networkingv1.IngressSpec{
				IngressClassName: &ingressClassName,
				Rules: []networkingv1.IngressRule{
					{
						Host: hostname,
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/another-port",
										PathType: &pathType,
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: "test-collector",
												Port: networkingv1.ServiceBackendPort{
													Name: "another-port",
												},
											},
										},
									},
									{
										Path:     "/otlp-grpc",
										PathType: &pathType,
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: "test-collector",
												Port: networkingv1.ServiceBackendPort{
													Name: "otlp-grpc",
												},
											},
										},
									},
									{
										Path:     "/otlp-test-grpc",
										PathType: &pathType,
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: "test-collector",
												Port: networkingv1.ServiceBackendPort{
													Name: "otlp-test-grpc",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}, got)
	})

}

func TestExpectedIngresses(t *testing.T) {
	t.Run("should create and update ingress entry", func(t *testing.T) {
		ctx := context.Background()

		params, err := newParams("something:tag", testFileIngress)
		if err != nil {
			t.Fatal(err)
		}
		params.Instance.Spec.Ingress.Type = "ingress"

		err = expectedIngresses(ctx, params, []networkingv1.Ingress{*desiredIngresses(ctx, params)})
		assert.NoError(t, err)

		nns := types.NamespacedName{Namespace: "default", Name: "test-ingress"}
		exists, err := populateObjectIfExists(t, &networkingv1.Ingress{}, nns)
		assert.NoError(t, err)
		assert.True(t, exists)

		// update fields
		const expectHostname = "something-else.com"
		params.Instance.Spec.Ingress.Annotations = map[string]string{"blub": "blob"}
		params.Instance.Spec.Ingress.Hostname = expectHostname

		err = expectedIngresses(ctx, params, []networkingv1.Ingress{*desiredIngresses(ctx, params)})
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

		err = expectedIngresses(ctx, myParams, []networkingv1.Ingress{*desiredIngresses(ctx, myParams)})
		assert.NoError(t, err)

		nns := types.NamespacedName{Namespace: "default", Name: "test-ingress"}
		exists, err := populateObjectIfExists(t, &networkingv1.Ingress{}, nns)
		assert.NoError(t, err)
		assert.True(t, exists)

		// delete
		if delIngressErr := deleteIngresses(ctx, params(), []networkingv1.Ingress{}); delIngressErr != nil {
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
