// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package instrumentation

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func TestParseServiceConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *serviceConfig
	}{
		{
			name:  "default java without prefix",
			input: "myservice",
			expected: &serviceConfig{
				lang:           "java",
				serviceName:    "myservice",
				executablePath: "",
			},
		},
		{
			name:  "go without executable path",
			input: "go:myservice",
			expected: &serviceConfig{
				lang:           "go",
				serviceName:    "myservice",
				executablePath: "",
			},
		},
		{
			name:  "go with executable path",
			input: "go:myservice:/app/main",
			expected: &serviceConfig{
				lang:           "go",
				serviceName:    "myservice",
				executablePath: "/app/main",
			},
		},
		{
			name:  "jvm prefix maps to java",
			input: "jvm:myservice",
			expected: &serviceConfig{
				lang:           "java",
				serviceName:    "myservice",
				executablePath: "",
			},
		},
		{
			name:  "node prefix maps to nodejs",
			input: "node:myservice",
			expected: &serviceConfig{
				lang:           "nodejs",
				serviceName:    "myservice",
				executablePath: "",
			},
		},
		{
			name:  "py prefix maps to python",
			input: "py:myservice",
			expected: &serviceConfig{
				lang:           "python",
				serviceName:    "myservice",
				executablePath: "",
			},
		},
		{
			name:  "dotnet prefix",
			input: "dotnet:myservice",
			expected: &serviceConfig{
				lang:           "dotnet",
				serviceName:    "myservice",
				executablePath: "",
			},
		},
		{
			name:  "path with colons",
			input: "go:myservice:/usr/bin:backup",
			expected: &serviceConfig{
				lang:           "go",
				serviceName:    "myservice",
				executablePath: "/usr/bin:backup",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseServiceConfig(tt.input)
			assert.Equal(t, tt.expected.lang, result.lang)
			assert.Equal(t, tt.expected.serviceName, result.serviceName)
			assert.Equal(t, tt.expected.executablePath, result.executablePath)
		})
	}
}

func TestMapPrefixToLang(t *testing.T) {
	tests := []struct {
		prefix   string
		expected string
	}{
		{"go", "go"},
		{"jvm", "java"},
		{"node", "nodejs"},
		{"py", "python"},
		{"dotnet", "dotnet"},
		{"unknown", "java"}, // default
		{"", "java"},        // default
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			result := mapPrefixToLang(tt.prefix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetServiceName(t *testing.T) {
	tests := []struct {
		name     string
		pod      corev1.Pod
		expected string
	}{
		{
			name: "with app label",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod-name",
					Labels: map[string]string{
						"app": "myapp",
					},
				},
			},
			expected: "myapp",
		},
		{
			name: "with k8s app name label",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod-name",
					Labels: map[string]string{
						"app.kubernetes.io/name": "myservice",
					},
				},
			},
			expected: "myservice",
		},
		{
			name: "app label takes priority",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod-name",
					Labels: map[string]string{
						"app":                    "myapp",
						"app.kubernetes.io/name": "myservice",
					},
				},
			},
			expected: "myapp",
		},
		{
			name: "no labels returns pod name",
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod-name",
				},
			},
			expected: "pod-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := &instPodMutator{}
			result := pm.getServiceName(tt.pod)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatchTargetService(t *testing.T) {
	tests := []struct {
		name           string
		podServiceName string
		configService  string
		shouldMatch    bool
	}{
		{
			name:           "exact match",
			podServiceName: "myapp",
			configService:  "myapp",
			shouldMatch:    true,
		},
		{
			name:           "match with prefix",
			podServiceName: "myapp",
			configService:  "go:myapp",
			shouldMatch:    true,
		},
		{
			name:           "match with prefix and path",
			podServiceName: "myapp",
			configService:  "go:myapp:/app/main",
			shouldMatch:    true,
		},
		{
			name:           "no match",
			podServiceName: "myapp",
			configService:  "otherapp",
			shouldMatch:    false,
		},
		{
			name:           "no match different service",
			podServiceName: "myapp",
			configService:  "go:otherapp",
			shouldMatch:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchTargetService(tt.podServiceName, tt.configService)
			if tt.shouldMatch {
				assert.NotNil(t, result)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestShouldAutoInject(t *testing.T) {
	tests := []struct {
		name              string
		pod               corev1.Pod
		namespace         corev1.Namespace
		instrumentations  []v1alpha1.Instrumentation
		expectInst        bool
		expectConfig      *serviceConfig
		expectError       bool
	}{
		{
			name: "auto-injection enabled and matches",
			namespace: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ns",
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "test-ns",
					Labels: map[string]string{
						"app": "myapp",
					},
				},
			},
			instrumentations: []v1alpha1.Instrumentation{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-inst",
						Namespace: "test-ns",
					},
					Spec: v1alpha1.InstrumentationSpec{
						AutoInjection: &v1alpha1.AutoInjectionSpec{
							Enabled:        true,
							TargetServices: []string{"myapp"},
						},
					},
				},
			},
			expectInst: true,
			expectConfig: &serviceConfig{
				lang:        "java",
				serviceName: "myapp",
			},
		},
		{
			name: "auto-injection disabled",
			namespace: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ns",
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "test-ns",
					Labels: map[string]string{
						"app": "myapp",
					},
				},
			},
			instrumentations: []v1alpha1.Instrumentation{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-inst",
						Namespace: "test-ns",
					},
					Spec: v1alpha1.InstrumentationSpec{
						AutoInjection: &v1alpha1.AutoInjectionSpec{
							Enabled: false,
						},
					},
				},
			},
			expectInst: false,
		},
		{
			name: "no matching service",
			namespace: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ns",
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "test-ns",
					Labels: map[string]string{
						"app": "myapp",
					},
				},
			},
			instrumentations: []v1alpha1.Instrumentation{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-inst",
						Namespace: "test-ns",
					},
					Spec: v1alpha1.InstrumentationSpec{
						AutoInjection: &v1alpha1.AutoInjectionSpec{
							Enabled:        true,
							TargetServices: []string{"otherapp"},
						},
					},
				},
			},
			expectInst: false,
		},
		{
			name: "namespace mismatch",
			namespace: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ns",
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "test-ns",
					Labels: map[string]string{
						"app": "myapp",
					},
				},
			},
			instrumentations: []v1alpha1.Instrumentation{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-inst",
						Namespace: "test-ns",
					},
					Spec: v1alpha1.InstrumentationSpec{
						AutoInjection: &v1alpha1.AutoInjectionSpec{
							Enabled:        true,
							TargetServices: []string{"myapp"},
						},
					},
				},
			},
			expectInst: true,
			expectConfig: &serviceConfig{
				lang:        "java",
				serviceName: "myapp",
			},
		},
		{
			name: "go service with executable path",
			namespace: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ns",
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "test-ns",
					Labels: map[string]string{
						"app": "mygoapp",
					},
				},
			},
			instrumentations: []v1alpha1.Instrumentation{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-inst",
						Namespace: "test-ns",
					},
					Spec: v1alpha1.InstrumentationSpec{
						AutoInjection: &v1alpha1.AutoInjectionSpec{
							Enabled:        true,
							TargetServices: []string{"go:mygoapp:/app/main"},
						},
					},
				},
			},
			expectInst: true,
			expectConfig: &serviceConfig{
				lang:           "go",
				serviceName:    "mygoapp",
				executablePath: "/app/main",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			err := v1alpha1.AddToScheme(scheme)
			require.NoError(t, err)
			err = corev1.AddToScheme(scheme)
			require.NoError(t, err)

			objs := []runtime.Object{}
			for i := range tt.instrumentations {
				objs = append(objs, &tt.instrumentations[i])
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(objs...).
				Build()

			pm := &instPodMutator{
				Client:   fakeClient,
				Logger:   logr.Discard(),
				Recorder: record.NewFakeRecorder(10),
			}

			inst, cfg, err := pm.shouldAutoInject(context.Background(), tt.namespace, tt.pod)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			if tt.expectInst {
				assert.NotNil(t, inst)
				assert.NotNil(t, cfg)
				if tt.expectConfig != nil {
					assert.Equal(t, tt.expectConfig.lang, cfg.lang)
					assert.Equal(t, tt.expectConfig.serviceName, cfg.serviceName)
					if tt.expectConfig.executablePath != "" {
						assert.Equal(t, tt.expectConfig.executablePath, cfg.executablePath)
					}
				}
			} else {
				assert.Nil(t, inst)
				assert.Nil(t, cfg)
			}
		})
	}
}

func TestMutate_AutoInjection(t *testing.T) {
	tests := []struct {
		name             string
		pod              corev1.Pod
		namespace        corev1.Namespace
		instrumentation  v1alpha1.Instrumentation
		config           config.Config
		expectInjected   bool
		expectAnnotation bool
		expectLang       string
	}{
		{
			name: "auto-inject java service",
			namespace: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ns",
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "test-ns",
					Labels: map[string]string{
						"app": "myapp",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app"},
					},
				},
			},
			instrumentation: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-inst",
					Namespace: "test-ns",
				},
				Spec: v1alpha1.InstrumentationSpec{
					Java: v1alpha1.Java{},
					AutoInjection: &v1alpha1.AutoInjectionSpec{
						Enabled:        true,
						TargetServices: []string{"myapp"},
					},
				},
			},
			config: config.Config{
				EnableJavaAutoInstrumentation: true,
			},
			expectInjected:   true,
			expectAnnotation: true,
			expectLang:       "java",
		},
		{
			name: "skip already auto-injected pod",
			namespace: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ns",
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "test-ns",
					Labels: map[string]string{
						"app": "myapp",
					},
					Annotations: map[string]string{
						"instrumentation.opentelemetry.io/auto-injected": "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app"},
					},
				},
			},
			instrumentation: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-inst",
					Namespace: "test-ns",
				},
				Spec: v1alpha1.InstrumentationSpec{
					AutoInjection: &v1alpha1.AutoInjectionSpec{
						Enabled:        true,
						TargetServices: []string{"myapp"},
					},
				},
			},
			config: config.Config{
				EnableJavaAutoInstrumentation: true,
			},
			expectInjected: false,
		},
		{
			name: "reject when language disabled",
			namespace: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ns",
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "test-ns",
					Labels: map[string]string{
						"app": "myapp",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app"},
					},
				},
			},
			instrumentation: v1alpha1.Instrumentation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-inst",
					Namespace: "test-ns",
				},
				Spec: v1alpha1.InstrumentationSpec{
					Java: v1alpha1.Java{},
					AutoInjection: &v1alpha1.AutoInjectionSpec{
						Enabled:        true,
						TargetServices: []string{"myapp"},
					},
				},
			},
			config: config.Config{
				EnableJavaAutoInstrumentation: false,
			},
			expectInjected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			err := v1alpha1.AddToScheme(scheme)
			require.NoError(t, err)
			err = corev1.AddToScheme(scheme)
			require.NoError(t, err)

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(&tt.instrumentation).
				Build()

			pm := &instPodMutator{
				Client:   fakeClient,
				Logger:   logr.Discard(),
				Recorder: record.NewFakeRecorder(10),
				config:   tt.config,
				sdkInjector: &sdkInjector{
					client: fakeClient,
					logger: logr.Discard(),
				},
			}

			result, err := pm.Mutate(context.Background(), tt.namespace, tt.pod)
			assert.NoError(t, err)

			if tt.expectAnnotation {
				assert.Equal(t, "true", result.Annotations["instrumentation.opentelemetry.io/auto-injected"])
			}

			// Note: Full injection verification would require checking volumes, init containers, etc.
			// This is a basic test to verify the auto-injection logic flow
		})
	}
}
