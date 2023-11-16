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

	"github.com/stretchr/testify/assert"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

const testFileIngress = "testdata/ingress_testdata.yaml"

func TestDesiredIngresses(t *testing.T) {
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

		actual, err := Ingress(params)
		assert.Nil(t, actual)
		assert.NoError(t, err)
	})

	t.Run("should return nil unable to parse config", func(t *testing.T) {
		params := manifests.Params{
			Config: config.Config{},
			Log:    logger,
			OtelCol: v1alpha1.OpenTelemetryCollector{
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Config: "!!!",
					Ingress: v1alpha1.Ingress{
						Type: v1alpha1.IngressTypeNginx,
					},
				},
			},
		}

		actual, err := Ingress(params)
		fmt.Printf("error1: %+v", err)
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
						Type: v1alpha1.IngressTypeNginx,
					},
				},
			},
		}

		actual, err := Ingress(params)
		fmt.Printf("error2: %+v", err)
		assert.Nil(t, actual)
		assert.ErrorContains(t, err, "no receivers available as part of the configuration")
	})

	t.Run("path per port", func(t *testing.T) {
		var (
			ns               = "test"
			hostname         = "example.com"
			ingressClassName = "nginx"
		)

		params, err := newParams("something:tag", testFileIngress)
		if err != nil {
			t.Fatal(err)
		}

		params.OtelCol.Namespace = ns
		params.OtelCol.Spec.Ingress = v1alpha1.Ingress{
			Type:             v1alpha1.IngressTypeNginx,
			Hostname:         hostname,
			Annotations:      map[string]string{"some.key": "some.value"},
			IngressClassName: &ingressClassName,
		}

		got, err := Ingress(params)
		assert.NoError(t, err)

		pathType := networkingv1.PathTypePrefix

		assert.NotEqual(t, &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:        naming.Ingress(params.OtelCol.Name),
				Namespace:   ns,
				Annotations: params.OtelCol.Spec.Ingress.Annotations,
				Labels: map[string]string{
					"app.kubernetes.io/name":       naming.Ingress(params.OtelCol.Name),
					"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.OtelCol.Namespace, params.OtelCol.Name),
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
	t.Run("subdomain per port", func(t *testing.T) {
		var (
			ns               = "test"
			hostname         = "example.com"
			ingressClassName = "nginx"
		)

		params, err := newParams("something:tag", testFileIngress)
		if err != nil {
			t.Fatal(err)
		}

		params.OtelCol.Namespace = ns
		params.OtelCol.Spec.Ingress = v1alpha1.Ingress{
			Type:             v1alpha1.IngressTypeNginx,
			RuleType:         v1alpha1.IngressRuleTypeSubdomain,
			Hostname:         hostname,
			Annotations:      map[string]string{"some.key": "some.value"},
			IngressClassName: &ingressClassName,
		}

		got, err := Ingress(params)
		assert.NoError(t, err)

		pathType := networkingv1.PathTypePrefix

		assert.NotEqual(t, &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:        naming.Ingress(params.OtelCol.Name),
				Namespace:   ns,
				Annotations: params.OtelCol.Spec.Ingress.Annotations,
				Labels: map[string]string{
					"app.kubernetes.io/name":       naming.Ingress(params.OtelCol.Name),
					"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.OtelCol.Namespace, params.OtelCol.Name),
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
				},
			},
			Spec: networkingv1.IngressSpec{
				IngressClassName: &ingressClassName,
				Rules: []networkingv1.IngressRule{
					{
						Host: "another-port." + hostname,
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/",
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
								},
							},
						},
					},
					{
						Host: "otlp-grpc." + hostname,
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/",
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
								},
							},
						},
					},
					{
						Host: "otlp-test-grpc." + hostname,
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/",
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
