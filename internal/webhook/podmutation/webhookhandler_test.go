// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package podmutation_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	. "github.com/open-telemetry/opentelemetry-operator/internal/webhook/podmutation"
	"github.com/open-telemetry/opentelemetry-operator/pkg/sidecar"
)

var logger = logf.Log.WithName("unit-tests")

func TestShouldInjectSidecar(t *testing.T) {
	for _, tt := range []struct {
		name     string
		ns       corev1.Namespace
		pod      corev1.Pod
		otelcols []v1alpha1.OpenTelemetryCollector
	}{
		{
			// this is the simplest positive test: a pod is being created with an annotation
			// telling the operator to inject an instance, and the annotation's value contains
			// the name of an existing otelcol instance with Mode=Sidecar
			name: "simplest positive case",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-namespace-simplest-positive-case",
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{sidecar.Annotation: "my-instance"},
				},
			},
			otelcols: []v1alpha1.OpenTelemetryCollector{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-instance",
					Namespace: "my-namespace-simplest-positive-case",
				},
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode: v1alpha1.ModeSidecar,
				},
			}},
		},
		{
			// in this case, the annotation is at the namespace instead of at the pod
			name: "namespace is annotated",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "my-annotated-namespace",
					Annotations: map[string]string{sidecar.Annotation: "my-instance"},
				},
			},
			pod: corev1.Pod{},
			otelcols: []v1alpha1.OpenTelemetryCollector{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-instance",
					Namespace: "my-annotated-namespace",
				},
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode: v1alpha1.ModeSidecar,
				},
			}},
		},
		{
			// now, we automatically select an existing sidecar otelcol
			name: "auto-select based on the annotation's value",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "my-namespace-with-autoselect",
					Annotations: map[string]string{sidecar.Annotation: "true"},
				},
			},
			pod: corev1.Pod{},
			otelcols: []v1alpha1.OpenTelemetryCollector{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-instance",
						Namespace: "my-namespace-with-autoselect",
					},
					Spec: v1alpha1.OpenTelemetryCollectorSpec{
						Mode: v1alpha1.ModeSidecar,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "a-deployment-instance",
						Namespace: "my-namespace-with-autoselect",
					},
					Spec: v1alpha1.OpenTelemetryCollectorSpec{
						Mode: v1alpha1.ModeDeployment,
					},
				},
			},
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := k8sClient.Create(context.Background(), &tt.ns)
			require.NoError(t, err)
			defer func() {
				_ = k8sClient.Delete(context.Background(), &tt.ns)
			}()

			for i := range tt.otelcols {
				clientErr := k8sClient.Create(context.Background(), &tt.otelcols[i])
				require.NoError(t, clientErr)
			}

			encoded, err := json.Marshal(tt.pod)
			require.NoError(t, err)

			// the actual request we see in the webhook
			req := admission.Request{
				AdmissionRequest: admv1.AdmissionRequest{
					Namespace: tt.ns.Name,
					Object: runtime.RawExtension{
						Raw: encoded,
					},
				},
			}

			// the webhook handler
			cfg := config.New()
			decoder := admission.NewDecoder(scheme.Scheme)
			injector := NewWebhookHandler(cfg, logger, decoder, k8sClient, []PodMutator{sidecar.NewMutator(logger, cfg, k8sClient)})

			// test
			res := injector.Handle(context.Background(), req)

			// verify
			assert.True(t, res.Allowed)
			assert.Nil(t, res.AdmissionResponse.Result)
			assert.Len(t, res.Patches, 2)

			expectedMap := map[string]bool{
				"/metadata/labels": false, // add a new label
				"/spec/containers": false, // replace the containers, adding one new container
			}
			for _, patch := range res.Patches {
				// quick and dirty solution
				if patch.Path == "/spec/containers" {
					assert.Equal(t, "replace", patch.Operation)
				} else {
					assert.Equal(t, "add", patch.Operation)
				}

				expectedMap[patch.Path] = true
			}
			for k := range expectedMap {
				assert.True(t, expectedMap[k], "patch with path %s not found", k)
			}

			// cleanup
			for i := range tt.otelcols {
				require.NoError(t, k8sClient.Delete(context.Background(), &tt.otelcols[i]))
			}
		})
	}
}

func TestPodShouldNotBeChanged(t *testing.T) {
	for _, tt := range []struct {
		name     string
		ns       corev1.Namespace
		pod      corev1.Pod
		otelcols []v1alpha1.OpenTelemetryCollector
	}{
		{
			name: "namespace has no annotations",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-namespace-no-annotations",
				},
			},
			pod: corev1.Pod{},
			otelcols: []v1alpha1.OpenTelemetryCollector{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-instance",
					Namespace: "my-namespace-no-annotations",
				},
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode: v1alpha1.ModeSidecar,
				},
			}},
		},
		{
			name: "multiple possible otelcols",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "my-namespace-multiple-otelcols",
					Annotations: map[string]string{sidecar.Annotation: "true"},
				},
			},
			pod: corev1.Pod{},
			otelcols: []v1alpha1.OpenTelemetryCollector{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-instance-1",
						Namespace: "my-namespace-multiple-otelcols",
					},
					Spec: v1alpha1.OpenTelemetryCollectorSpec{
						Mode: v1alpha1.ModeSidecar,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-instance-2",
						Namespace: "my-namespace-multiple-otelcols",
					},
					Spec: v1alpha1.OpenTelemetryCollectorSpec{
						Mode: v1alpha1.ModeSidecar,
					},
				},
			},
		},
		{
			name: "no otelcols",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "my-namespace-no-otelcols",
					Annotations: map[string]string{sidecar.Annotation: "true"},
				},
			},
			pod:      corev1.Pod{},
			otelcols: []v1alpha1.OpenTelemetryCollector{},
		},
		{
			name: "otelcol is not a sidecar",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "my-namespace-no-sidecar-otelcol",
					Annotations: map[string]string{sidecar.Annotation: "my-instance"},
				},
			},
			pod: corev1.Pod{},
			otelcols: []v1alpha1.OpenTelemetryCollector{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-instance",
					Namespace: "my-namespace-no-sidecar-otelcol",
				},
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode: v1alpha1.ModeDaemonSet,
				},
			}},
		},
		{
			name: "automatically injected otelcol is not a sidecar",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "my-namespace-no-automatic-sidecar-otelcol",
					Annotations: map[string]string{sidecar.Annotation: "true"},
				},
			},
			pod: corev1.Pod{},
			otelcols: []v1alpha1.OpenTelemetryCollector{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-instance",
					Namespace: "my-namespace-no-automatic-sidecar-otelcol",
				},
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode: v1alpha1.ModeDaemonSet,
				},
			}},
		},
		{
			name: "pod has sidecar already",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-namespace-pod-has-sidecar",
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{sidecar.Annotation: "my-instance"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name: naming.Container(),
					}},
				},
			},
			otelcols: []v1alpha1.OpenTelemetryCollector{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-instance",
					Namespace: "my-namespace-pod-has-sidecar",
				},
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode: v1alpha1.ModeSidecar,
				},
			}},
		},
		{
			name: "sidecar not desired",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-namespace-sidecar-not-desired",
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{sidecar.Annotation: "false"},
				},
			},
			otelcols: []v1alpha1.OpenTelemetryCollector{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-instance",
					Namespace: "my-namespace-sidecar-not-desired",
				},
				Spec: v1alpha1.OpenTelemetryCollectorSpec{
					Mode: v1alpha1.ModeSidecar,
				},
			}},
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := k8sClient.Create(context.Background(), &tt.ns)
			require.NoError(t, err)
			defer func() {
				_ = k8sClient.Delete(context.Background(), &tt.ns)
			}()

			for i := range tt.otelcols {
				clientErr := k8sClient.Create(context.Background(), &tt.otelcols[i])
				require.NoError(t, clientErr)
			}

			encoded, err := json.Marshal(tt.pod)
			require.NoError(t, err)

			// the actual request we see in the webhook
			req := admission.Request{
				AdmissionRequest: admv1.AdmissionRequest{
					Namespace: tt.ns.Name,
					Object: runtime.RawExtension{
						Raw: encoded,
					},
				},
			}

			// the webhook handler
			cfg := config.New()
			decoder := admission.NewDecoder(scheme.Scheme)
			injector := NewWebhookHandler(cfg, logger, decoder, k8sClient, []PodMutator{sidecar.NewMutator(logger, cfg, k8sClient)})
			require.NoError(t, err)

			// test
			res := injector.Handle(context.Background(), req)

			// verify
			assert.True(t, res.Allowed)
			assert.Nil(t, res.AdmissionResponse.Result)
			assert.Len(t, res.Patches, 0)

			// cleanup
			for i := range tt.otelcols {
				require.NoError(t, k8sClient.Delete(context.Background(), &tt.otelcols[i]))
			}
		})
	}
}

func TestFailOnInvalidRequest(t *testing.T) {
	// we use a typical Go table-test instad of Ginkgo's DescribeTable because we need to
	// do an assertion during the declaration of the table params, which isn't supported (yet?)
	for _, tt := range []struct {
		req      admission.Request
		name     string
		expected int32
		allowed  bool
	}{
		{
			name:     "empty payload",
			req:      admission.Request{},
			expected: http.StatusBadRequest,
			allowed:  false,
		},
		{
			name: "namespace doesn't exist",
			req: func() admission.Request {
				pod := corev1.Pod{}
				encoded, err := json.Marshal(pod)
				require.NoError(t, err)

				return admission.Request{
					AdmissionRequest: admv1.AdmissionRequest{
						Namespace: "non-existing",
						Object: runtime.RawExtension{
							Raw: encoded,
						},
					},
				}
			}(),
			expected: http.StatusInternalServerError,
			allowed:  true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// prepare
			cfg := config.New()
			decoder := admission.NewDecoder(scheme.Scheme)
			injector := NewWebhookHandler(cfg, logger, decoder, k8sClient, []PodMutator{sidecar.NewMutator(logger, cfg, k8sClient)})

			// test
			res := injector.Handle(context.Background(), tt.req)

			// verify
			assert.Equal(t, tt.allowed, res.Allowed)
			assert.NotNil(t, res.AdmissionResponse.Result)
			assert.Equal(t, tt.expected, res.AdmissionResponse.Result.Code)
		})
	}
}
