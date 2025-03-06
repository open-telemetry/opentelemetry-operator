// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

const testFileIngress = "testdata/ingress_testdata.yaml"

func TestDesiredIngresses(t *testing.T) {
	t.Run("should return nil invalid ingress type", func(t *testing.T) {
		params := manifests.Params{
			Config: config.Config{},
			Log:    logger,
			OtelCol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Ingress: v1beta1.Ingress{
						Type: v1beta1.IngressType("unknown"),
					},
				},
			},
		}

		actual, err := Ingress(params)
		assert.Nil(t, actual)
		assert.NoError(t, err)
	})

	t.Run("should return nil, no ingress set", func(t *testing.T) {
		params := manifests.Params{
			Config: config.Config{},
			Log:    logger,
			OtelCol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Mode: "Deployment",
				},
			},
		}

		actual, err := Ingress(params)
		assert.Nil(t, actual)
		assert.NoError(t, err)
	})

	t.Run("should return nil unable to parse receiver ports", func(t *testing.T) {
		params := manifests.Params{
			Config: config.Config{},
			Log:    logger,
			OtelCol: v1beta1.OpenTelemetryCollector{
				Spec: v1beta1.OpenTelemetryCollectorSpec{
					Config: v1beta1.Config{},
					Ingress: v1beta1.Ingress{
						Type: v1beta1.IngressTypeIngress,
					},
				},
			},
		}

		actual, err := Ingress(params)
		assert.Nil(t, actual)
		assert.NoError(t, err)
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
		params.OtelCol.Spec.Ingress = v1beta1.Ingress{
			Type:             v1beta1.IngressTypeIngress,
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
					"app.kubernetes.io/component":  "opentelemetry-collector",
					"app.kubernetes.io/part-of":    "opentelemetry",
					"app.kubernetes.io/version":    "latest",
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
		params.OtelCol.Spec.Ingress = v1beta1.Ingress{
			Type:             v1beta1.IngressTypeIngress,
			RuleType:         v1beta1.IngressRuleTypeSubdomain,
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
