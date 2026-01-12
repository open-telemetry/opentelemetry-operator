// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestEffectiveAnnotationValue(t *testing.T) {
	for _, tt := range []struct {
		desc     string
		expected string
		pod      corev1.Pod
		ns       corev1.Namespace
	}{
		{
			"pod-true-overrides-ns",
			"true",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJava: "true",
					},
				},
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJava: "false",
					},
				},
			},
		},

		{
			"ns-has-concrete-instance",
			"true",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJava: "true",
					},
				},
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJava: "some-instance",
					},
				},
			},
		},

		{
			"pod-has-concrete-instance",
			"",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJavaContainersName: "some-instance-from-pod",
					},
				},
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJavaContainersName: "some-instance",
					},
				},
			},
		},

		{
			"pod-has-concrete-instance-and-inject",
			"true",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJava:               "true",
						annotationInjectJavaContainersName: "some-instance-from-pod",
					},
				},
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJavaContainersName: "some-instance",
					},
				},
			},
		},

		{
			"pod-has-concrete-instance-and-ns-sdk",
			"true",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJava: "true",
					},
				},
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectSdk: "true",
					},
				},
			},
		},

		{
			"pod-python-overrides-ns-sdk",
			"",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectPython: "true",
					},
				},
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectSdk: "true",
					},
				},
			},
		},

		{
			"pod-python-and-java-no-sdk-from-ns",
			"true",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectPython: "true",
						annotationInjectJava:   "true",
					},
				},
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectSdk: "true",
					},
				},
			},
		},

		{
			"ns-java-applied-when-pod-has-no-instrumentation",
			"true",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"some.other.annotation": "value",
					},
				},
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJava: "true",
					},
				},
			},
		},

		{
			"pod-false-blocks-ns-sdk",
			"false",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJava: "false",
					},
				},
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectSdk: "true",
					},
				},
			},
		},

		{
			"pod-has-explicit-false",
			"false",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJava: "false",
					},
				},
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJava: "some-instance",
					},
				},
			},
		},

		{
			"pod-has-no-annotations",
			"some-instance",
			corev1.Pod{},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJava: "some-instance",
					},
				},
			},
		},

		{
			"ns-has-no-annotations",
			"true",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectJava: "true",
					},
				},
			},
			corev1.Namespace{},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			// test
			annValue := annotationValue(tt.ns.ObjectMeta, tt.pod.ObjectMeta, annotationInjectJava)

			// verify
			assert.Equal(t, tt.expected, annValue)
		})
	}
}

func TestCrossInstrumentationPrecedence(t *testing.T) {
	for _, tt := range []struct {
		desc       string
		expected   string
		pod        corev1.Pod
		ns         corev1.Namespace
		annotation string
	}{
		{
			"pod-python-blocks-ns-sdk-when-testing-sdk",
			"",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectPython: "true",
					},
				},
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectSdk: "true",
					},
				},
			},
			annotationInjectSdk,
		},
		{
			"ns-sdk-used-when-pod-has-no-instrumentation",
			"true",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"some.other.annotation": "value",
					},
				},
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectSdk: "true",
					},
				},
			},
			annotationInjectSdk,
		},
		{
			"pod-go-blocks-ns-python-when-testing-python",
			"",
			corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectGo: "true",
					},
				},
			},
			corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						annotationInjectPython: "true",
					},
				},
			},
			annotationInjectPython,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			// test
			annValue := annotationValue(tt.ns.ObjectMeta, tt.pod.ObjectMeta, tt.annotation)

			// verify
			assert.Equal(t, tt.expected, annValue)
		})
	}
}
