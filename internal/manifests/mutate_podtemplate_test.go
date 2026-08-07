// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package manifests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

// TestMutatePodTemplateAuthoritativeMetadata covers the pod template metadata
// semantics: the desired state generated from the CR is authoritative, so
// entries removed from the CR (e.g. spec.podAnnotations) are removed from the
// workload (Refs #2795). Only keys matching the configured preserve patterns
// survive when absent from the desired state.
func TestMutatePodTemplateAuthoritativeMetadata(t *testing.T) {
	const (
		sha256Key = "opentelemetry-operator-config/sha256"
		oldHash   = "old-hash"
		newHash   = "new-hash"
	)

	tests := []struct {
		name              string
		cfg               config.Config
		existingTemplate  corev1.PodTemplateSpec
		desiredTemplate   corev1.PodTemplateSpec
		expectAnnotations map[string]string
		expectLabels      map[string]string
	}{
		{
			name: "entries removed from the CR are removed from the pod template",
			existingTemplate: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"actokey": "actokey",
						sha256Key: oldHash,
					},
				},
			},
			desiredTemplate: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sha256Key: newHash,
					},
				},
			},
			expectAnnotations: map[string]string{
				sha256Key: newHash,
			},
		},
		{
			name: "out-of-band template annotations are dropped by default",
			existingTemplate: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"kubectl.kubernetes.io/restartedAt": "2024-01-01T00:00:00Z",
						sha256Key:                           oldHash,
					},
				},
			},
			desiredTemplate: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sha256Key: newHash,
					},
				},
			},
			expectAnnotations: map[string]string{
				sha256Key: newHash,
			},
		},
		{
			name: "annotations matching a preserve pattern survive",
			cfg: config.Config{
				PreservedAnnotations: []string{"kubectl.kubernetes.io/*"},
			},
			existingTemplate: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"kubectl.kubernetes.io/restartedAt": "2024-01-01T00:00:00Z",
						"argocd.argoproj.io/tracking-id":    "should-not-survive",
						sha256Key:                           oldHash,
					},
				},
			},
			desiredTemplate: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sha256Key: newHash,
					},
				},
			},
			expectAnnotations: map[string]string{
				"kubectl.kubernetes.io/restartedAt": "2024-01-01T00:00:00Z",
				sha256Key:                           newHash,
			},
		},
		{
			name: "desired value wins over preserved value for the same key",
			cfg: config.Config{
				PreservedAnnotations: []string{"prometheus.io/*"},
			},
			existingTemplate: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"prometheus.io/port": "9999",
						sha256Key:            oldHash,
					},
				},
			},
			desiredTemplate: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"prometheus.io/port": "8888",
						sha256Key:            newHash,
					},
				},
			},
			expectAnnotations: map[string]string{
				"prometheus.io/port": "8888",
				sha256Key:            newHash,
			},
		},
		{
			name: "disablePrometheusAnnotations transition removes operator-stamped defaults",
			existingTemplate: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"prometheus.io/scrape": "true",
						"prometheus.io/port":   "8888",
						"prometheus.io/path":   "/metrics",
						sha256Key:              oldHash,
					},
				},
			},
			desiredTemplate: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						sha256Key: newHash,
					},
				},
			},
			expectAnnotations: map[string]string{
				sha256Key: newHash,
			},
		},
		{
			name: "labels removed from the CR are removed from the pod template",
			existingTemplate: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": "test",
						"stale-label":            "stale",
					},
				},
			},
			desiredTemplate: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": "test",
					},
				},
			},
			expectLabels: map[string]string{
				"app.kubernetes.io/name": "test",
			},
		},
		{
			name: "labels matching a preserve pattern survive",
			cfg: config.Config{
				PreservedLabels: []string{"cost-center.io/*"},
			},
			existingTemplate: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": "test",
						"cost-center.io/team":    "payments",
						"stale-label":            "stale",
					},
				},
			},
			desiredTemplate: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": "test",
					},
				},
			},
			expectLabels: map[string]string{
				"app.kubernetes.io/name": "test",
				"cost-center.io/team":    "payments",
			},
		},
		{
			name:             "empty desired metadata results in nil maps, never empty maps",
			existingTemplate: corev1.PodTemplateSpec{},
			desiredTemplate:  corev1.PodTemplateSpec{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existing := tt.existingTemplate.DeepCopy()
			desired := tt.desiredTemplate.DeepCopy()
			err := mutatePodTemplate(existing, desired, tt.cfg)
			require.NoError(t, err)
			assert.Exactly(t, tt.expectAnnotations, existing.Annotations)
			assert.Exactly(t, tt.expectLabels, existing.Labels)
		})
	}
}

// TestMutatePodTemplateSpecReplaced guards that the pod spec itself keeps being
// fully replaced, independent of the metadata semantics.
func TestMutatePodTemplateSpecReplaced(t *testing.T) {
	existing := &corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "old"}},
		},
	}
	desired := &corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "new"}},
		},
	}

	err := mutatePodTemplate(existing, desired, config.Config{})
	require.NoError(t, err)
	assert.Exactly(t, desired.Spec, existing.Spec)
}
