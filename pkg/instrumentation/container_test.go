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

package instrumentation

import (
	"bytes"
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewConfigMapRef(name string, prefix string, optional *bool) corev1.EnvFromSource {
	return corev1.EnvFromSource{
		Prefix: prefix,
		ConfigMapRef: &corev1.ConfigMapEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: name,
			},
			Optional: optional,
		},
	}
}

func NewSecretRef(name string, prefix string, optional *bool) corev1.EnvFromSource {
	return corev1.EnvFromSource{
		Prefix: prefix,
		SecretRef: &corev1.SecretEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: name,
			},
			Optional: optional,
		},
	}
}

func NewConfigMapKeyRef(name string, key string, optional *bool) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{
		ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: name,
			},
			Key:      key,
			Optional: optional,
		},
	}
}

func NewSecretKeyRef(name string, key string, optional *bool) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: name,
			},
			Key:      key,
			Optional: optional,
		},
	}
}

func NewFieldRef(fieldPath string) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{
		FieldRef: &corev1.ObjectFieldSelector{
			FieldPath: fieldPath,
		},
	}
}

func NewResourceFieldRef(containerName string, resource string) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{
		ResourceFieldRef: &corev1.ResourceFieldSelector{
			ContainerName: containerName,
			Resource:      resource,
		},
	}
}

func NewStringLogger() (logr.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	logFunc := func(prefix, args string) { buf.WriteString(prefix + args + "\n") }
	options := funcr.Options{Verbosity: 4}
	logger := funcr.New(logFunc, options)
	return logger, &buf
}

func TestFindContainerByName(t *testing.T) {
	tests := []struct {
		name      string
		container string
		pod       corev1.Pod
		expected  int
	}{
		{
			name:      "Container found by name",
			container: "my-container",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "my-container"},
					},
				},
			},
			expected: 0,
		},
		{
			name:      "Containter found by name between multiple containers",
			container: "my-container2",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "my-container1"},
						{Name: "my-container2"},
					},
				},
			},
			expected: 1,
		},
		{
			name:      "Default container returned",
			container: "",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "my-container1"},
						{Name: "my-container2"},
					},
				},
			},
			expected: 0,
		},
		{
			name:      "No matching container found, default returned",
			container: "no-match",
			pod: corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "my-container1"},
						{Name: "my-container2"},
					},
				},
			},
			expected: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pod := test.pod
			index := getContainerIndex(&pod, test.container)
			assert.Equal(t, test.expected, index)
		})
	}
}

func TestInheritedEnv(t *testing.T) {
	true := true
	false := false

	tests := []struct {
		name     string
		ns       corev1.Namespace
		cm       []corev1.ConfigMap
		secret   []corev1.Secret
		pod      corev1.Pod
		err      string
		log      string
		expected map[string]string
	}{
		{
			name: "No ConfigMap usage",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "inheritedenv-noconfigmap",
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-pod",
					Namespace: "inheritedenv-noconfigmapv",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "app"}},
				},
			},
		},
		{
			name: "Simple ConfigMap usage",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "inheritedenv-simple",
				},
			},
			cm: []corev1.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-config",
						Namespace: "inheritedenv-simple",
					},
					Data: map[string]string{
						"OTEL_SERVICE_NAME":       "my-service",
						"OTEL_TRACES_SAMPLER_ARG": "0.85",
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-pod",
					Namespace: "inheritedenv-simple",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "app",
							EnvFrom: []corev1.EnvFromSource{NewConfigMapRef("my-config", "", nil)},
						},
					},
				},
			},
			expected: map[string]string{
				"OTEL_SERVICE_NAME":       "my-service",
				"OTEL_TRACES_SAMPLER_ARG": "0.85",
			},
		},
		{
			name: "Multiple ConfigMap usage with overriding",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "inheritedenv-multiple",
				},
			},
			cm: []corev1.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-config",
						Namespace: "inheritedenv-multiple",
					},
					Data: map[string]string{
						"OTEL_SERVICE_NAME":       "my-service",
						"OTEL_TRACES_SAMPLER":     "parentbased_traceidratio",
						"OTEL_TRACES_SAMPLER_ARG": "0.85",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-another-config",
						Namespace: "inheritedenv-multiple",
					},
					Data: map[string]string{
						"OTEL_TRACES_SAMPLER_ARG": "0.95",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-prefixed-config",
						Namespace: "inheritedenv-multiple",
					},
					Data: map[string]string{
						"EXPORTER_OTLP_TIMEOUT": "20",
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-pod",
					Namespace: "inheritedenv-multiple",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
							EnvFrom: []corev1.EnvFromSource{
								NewConfigMapRef("my-config", "", nil),
								NewConfigMapRef("my-another-config", "", nil),
								NewConfigMapRef("my-prefixed-config", "OTEL_", nil),
							},
						},
					},
				},
			},
			expected: map[string]string{
				"OTEL_SERVICE_NAME":          "my-service",
				"OTEL_TRACES_SAMPLER":        "parentbased_traceidratio",
				"OTEL_TRACES_SAMPLER_ARG":    "0.95",
				"OTEL_EXPORTER_OTLP_TIMEOUT": "20",
			},
		},
		{
			name: "Optional ConfigMap not found",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "inheritedenv-notfoundoptional",
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-pod",
					Namespace: "inheritedenv-notfoundoptional",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
							EnvFrom: []corev1.EnvFromSource{
								NewConfigMapRef("my-config", "", &true),
							},
						},
					},
				},
			},
		},
		{
			name: "Implicitly mandatory ConfigMap not found",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "inheritedenv-notfoundimplicit",
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-pod",
					Namespace: "inheritedenv-notfoundimplicit",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
							EnvFrom: []corev1.EnvFromSource{
								NewConfigMapRef("my-config", "", nil),
							},
						},
					},
				},
			},
			err: "failed to get ConfigMap inheritedenv-notfoundimplicit/my-config",
		},
		{
			name: "Explicitly mandatory ConfigMap not found",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "inheritedenv-notfoundexplicit",
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-pod",
					Namespace: "inheritedenv-notfoundexplicit",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
							EnvFrom: []corev1.EnvFromSource{
								NewConfigMapRef("my-config", "", &false),
							},
						},
					},
				},
			},
			err: "failed to get ConfigMap inheritedenv-notfoundexplicit/my-config",
		},
		{
			name: "SecretRef not supported",
			ns: corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "inheritedenv-secretref",
				},
			},
			secret: []corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-secret",
						Namespace: "inheritedenv-secretref",
					},
					Data: map[string][]byte{
						"OTEL_ENVFROM_SECRET_VALUE1": []byte("my-secret-value1"),
					},
				},
			},
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-pod",
					Namespace: "inheritedenv-secretref",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
							EnvFrom: []corev1.EnvFromSource{
								NewSecretRef("my-secret", "", nil),
							},
						},
					},
				},
			},
			log: "ignoring SecretRef in EnvFrom",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			err := k8sClient.Create(context.Background(), &test.ns)
			require.NoError(t, err)
			defer func() {
				_ = k8sClient.Delete(context.Background(), &test.ns)
			}()

			for _, cm := range test.cm {
				cm := cm
				err = k8sClient.Create(context.Background(), &cm)
				require.NoError(t, err)
				//goland:noinspection GoDeferInLoop
				defer func() {
					_ = k8sClient.Delete(context.Background(), &cm)
				}()
			}
			for _, secret := range test.secret {
				secret := secret
				err = k8sClient.Create(context.Background(), &secret)
				require.NoError(t, err)
				//goland:noinspection GoDeferInLoop
				defer func() {
					_ = k8sClient.Delete(context.Background(), &secret)
				}()
			}

			pod := test.pod
			logger, buf := NewStringLogger()
			container, err := NewContainer(k8sClient, context.Background(), logger, test.ns.Name, &pod, 0)
			if test.err == "" {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, container.inheritedEnv)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.err)
			}
			if test.log != "" {
				assert.Contains(t, buf.String(), test.log)
			}
		})
	}
}

type ModificationTester interface {
	Test(pod *corev1.Pod, c Container)
}

type ModificationTestFunc func(pod *corev1.Pod, c Container)

func (f ModificationTestFunc) Test(pod *corev1.Pod, c Container) {
	f(pod, c)
}

var _ ModificationTester = ModificationTestFunc(nil)

func TestModifications(t *testing.T) {
	testNs := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "modifications",
		},
	}
	testCm := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-config",
				Namespace: "modifications",
			},
			Data: map[string]string{
				"OTEL_ENVFROM_VALUE1": "my-envfrom-value1",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-refconfig",
				Namespace: "modifications",
			},
			Data: map[string]string{
				"ref-value1": "my-valuefrom-value1",
			},
		},
	}
	testSecret := []corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-secret",
				Namespace: "modifications",
			},
			Data: map[string][]byte{
				"OTEL_ENVFROM_SECRET_VALUE1": []byte("my-secret-value1"),
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-refsecret",
				Namespace: "modifications",
			},
			Data: map[string][]byte{
				"secret-ref-value1": []byte("my-valuefrom-value1"),
			},
		},
	}
	testPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-pod",
			Namespace: "modifications",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "app",
					EnvFrom: []corev1.EnvFromSource{
						{
							ConfigMapRef: &corev1.ConfigMapEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "my-config",
								},
							},
						},
					},
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_ENV_VALUE1",
							Value: "my-env-value1",
						},
						{
							Name:      "OTEL_ENV_VALUEFROM_CONFIGMAP1",
							ValueFrom: NewConfigMapKeyRef("my-refconfig", "ref-value1", nil),
						},
						{
							Name:      "OTEL_ENV_VALUEFROM_SECRET1",
							ValueFrom: NewSecretKeyRef("my-secret-refconfig", "secret-ref-value1", nil),
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name     string
		pod      *corev1.Pod
		tester   ModificationTester
		expected []corev1.EnvVar
	}{
		{
			name: "Test prepend",
			pod:  &testPod,
			tester: ModificationTestFunc(func(pod *corev1.Pod, c Container) {
				c.prepend(pod, "OTEL_ENV_VALUE2", "my-env-value2")
			}),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE2", Value: "my-env-value2"},
				{Name: "OTEL_ENV_VALUE1", Value: "my-env-value1"},
				{Name: "OTEL_ENV_VALUEFROM_CONFIGMAP1", ValueFrom: NewConfigMapKeyRef("my-refconfig", "ref-value1", nil)},
				{Name: "OTEL_ENV_VALUEFROM_SECRET1", ValueFrom: NewSecretKeyRef("my-secret-refconfig", "secret-ref-value1", nil)},
			},
		},
		{
			name: "Test prependEnvVar",
			pod:  &testPod,
			tester: ModificationTestFunc(func(pod *corev1.Pod, c Container) {
				c.prependEnvVar(pod, corev1.EnvVar{Name: "OTEL_ENV_VALUE2", Value: "my-env-value2"})
			}),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE2", Value: "my-env-value2"},
				{Name: "OTEL_ENV_VALUE1", Value: "my-env-value1"},
				{Name: "OTEL_ENV_VALUEFROM_CONFIGMAP1", ValueFrom: NewConfigMapKeyRef("my-refconfig", "ref-value1", nil)},
				{Name: "OTEL_ENV_VALUEFROM_SECRET1", ValueFrom: NewSecretKeyRef("my-secret-refconfig", "secret-ref-value1", nil)},
			},
		},
		{
			name: "Test append",
			pod:  &testPod,
			tester: ModificationTestFunc(func(pod *corev1.Pod, c Container) {
				c.append(pod, "OTEL_ENV_VALUE2", "my-env-value2")
			}),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE1", Value: "my-env-value1"},
				{Name: "OTEL_ENV_VALUEFROM_CONFIGMAP1", ValueFrom: NewConfigMapKeyRef("my-refconfig", "ref-value1", nil)},
				{Name: "OTEL_ENV_VALUEFROM_SECRET1", ValueFrom: NewSecretKeyRef("my-secret-refconfig", "secret-ref-value1", nil)},
				{Name: "OTEL_ENV_VALUE2", Value: "my-env-value2"},
			},
		},
		{
			name: "Test appendEnvVar",
			pod:  &testPod,
			tester: ModificationTestFunc(func(pod *corev1.Pod, c Container) {
				c.appendEnvVar(pod, corev1.EnvVar{Name: "OTEL_ENV_VALUE2", Value: "my-env-value2"})
			}),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE1", Value: "my-env-value1"},
				{Name: "OTEL_ENV_VALUEFROM_CONFIGMAP1", ValueFrom: NewConfigMapKeyRef("my-refconfig", "ref-value1", nil)},
				{Name: "OTEL_ENV_VALUEFROM_SECRET1", ValueFrom: NewSecretKeyRef("my-secret-refconfig", "secret-ref-value1", nil)},
				{Name: "OTEL_ENV_VALUE2", Value: "my-env-value2"},
			},
		},
		{
			name: "Test prependIfNotExists when env var does not exist",
			pod:  &testPod,
			tester: ModificationTestFunc(func(pod *corev1.Pod, c Container) {
				c.prependIfNotExists(pod, "OTEL_ENV_VALUE2", "my-env-value2")
			}),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE2", Value: "my-env-value2"},
				{Name: "OTEL_ENV_VALUE1", Value: "my-env-value1"},
				{Name: "OTEL_ENV_VALUEFROM_CONFIGMAP1", ValueFrom: NewConfigMapKeyRef("my-refconfig", "ref-value1", nil)},
				{Name: "OTEL_ENV_VALUEFROM_SECRET1", ValueFrom: NewSecretKeyRef("my-secret-refconfig", "secret-ref-value1", nil)},
			},
		},
		{
			name: "Test prependIfNotExists when env var exists",
			pod:  &testPod,
			tester: ModificationTestFunc(func(pod *corev1.Pod, c Container) {
				c.prependIfNotExists(pod, "OTEL_ENV_VALUE1", "my-overridden-value1")
			}),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE1", Value: "my-env-value1"},
				{Name: "OTEL_ENV_VALUEFROM_CONFIGMAP1", ValueFrom: NewConfigMapKeyRef("my-refconfig", "ref-value1", nil)},
				{Name: "OTEL_ENV_VALUEFROM_SECRET1", ValueFrom: NewSecretKeyRef("my-secret-refconfig", "secret-ref-value1", nil)},
			},
		},
		{
			name: "Test prependEnvVarIfNotExists when env var does not exist",
			pod:  &testPod,
			tester: ModificationTestFunc(func(pod *corev1.Pod, c Container) {
				c.prependEnvVarIfNotExists(pod, corev1.EnvVar{Name: "OTEL_ENV_VALUE2", Value: "my-env-value2"})
			}),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE2", Value: "my-env-value2"},
				{Name: "OTEL_ENV_VALUE1", Value: "my-env-value1"},
				{Name: "OTEL_ENV_VALUEFROM_CONFIGMAP1", ValueFrom: NewConfigMapKeyRef("my-refconfig", "ref-value1", nil)},
				{Name: "OTEL_ENV_VALUEFROM_SECRET1", ValueFrom: NewSecretKeyRef("my-secret-refconfig", "secret-ref-value1", nil)},
			},
		},
		{
			name: "Test prependEnvVarIfNotExists when env var exists",
			pod:  &testPod,
			tester: ModificationTestFunc(func(pod *corev1.Pod, c Container) {
				c.prependEnvVarIfNotExists(pod, corev1.EnvVar{Name: "OTEL_ENV_VALUE1", Value: "my-overridden-value1"})
			}),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE1", Value: "my-env-value1"},
				{Name: "OTEL_ENV_VALUEFROM_CONFIGMAP1", ValueFrom: NewConfigMapKeyRef("my-refconfig", "ref-value1", nil)},
				{Name: "OTEL_ENV_VALUEFROM_SECRET1", ValueFrom: NewSecretKeyRef("my-secret-refconfig", "secret-ref-value1", nil)},
			},
		},
		{
			name: "Test appendIfNotExists when env var does not exist",
			pod:  &testPod,
			tester: ModificationTestFunc(func(pod *corev1.Pod, c Container) {
				c.appendIfNotExists(pod, "OTEL_ENV_VALUE2", "my-env-value2")
			}),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE1", Value: "my-env-value1"},
				{Name: "OTEL_ENV_VALUEFROM_CONFIGMAP1", ValueFrom: NewConfigMapKeyRef("my-refconfig", "ref-value1", nil)},
				{Name: "OTEL_ENV_VALUEFROM_SECRET1", ValueFrom: NewSecretKeyRef("my-secret-refconfig", "secret-ref-value1", nil)},
				{Name: "OTEL_ENV_VALUE2", Value: "my-env-value2"},
			},
		},
		{
			name: "Test appendIfNotExists when env var exists",
			pod:  &testPod,
			tester: ModificationTestFunc(func(pod *corev1.Pod, c Container) {
				c.appendIfNotExists(pod, "OTEL_ENV_VALUE1", "my-overridden-value1")
			}),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE1", Value: "my-env-value1"},
				{Name: "OTEL_ENV_VALUEFROM_CONFIGMAP1", ValueFrom: NewConfigMapKeyRef("my-refconfig", "ref-value1", nil)},
				{Name: "OTEL_ENV_VALUEFROM_SECRET1", ValueFrom: NewSecretKeyRef("my-secret-refconfig", "secret-ref-value1", nil)},
			},
		},
		{
			name: "Test appendEnvVarIfNotExists when env var does not exist",
			pod:  &testPod,
			tester: ModificationTestFunc(func(pod *corev1.Pod, c Container) {
				c.appendEnvVarIfNotExists(pod, corev1.EnvVar{Name: "OTEL_ENV_VALUE2", Value: "my-env-value2"})
			}),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE1", Value: "my-env-value1"},
				{Name: "OTEL_ENV_VALUEFROM_CONFIGMAP1", ValueFrom: NewConfigMapKeyRef("my-refconfig", "ref-value1", nil)},
				{Name: "OTEL_ENV_VALUEFROM_SECRET1", ValueFrom: NewSecretKeyRef("my-secret-refconfig", "secret-ref-value1", nil)},
				{Name: "OTEL_ENV_VALUE2", Value: "my-env-value2"},
			},
		},
		{
			name: "Test appendEnvVarIfNotExists when env var exists",
			pod:  &testPod,
			tester: ModificationTestFunc(func(pod *corev1.Pod, c Container) {
				c.appendEnvVarIfNotExists(pod, corev1.EnvVar{Name: "OTEL_ENV_VALUE1", Value: "my-overridden-value1"})
			}),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE1", Value: "my-env-value1"},
				{Name: "OTEL_ENV_VALUEFROM_CONFIGMAP1", ValueFrom: NewConfigMapKeyRef("my-refconfig", "ref-value1", nil)},
				{Name: "OTEL_ENV_VALUEFROM_SECRET1", ValueFrom: NewSecretKeyRef("my-secret-refconfig", "secret-ref-value1", nil)},
			},
		},
		{
			name: "Test setOrAppendEnvVar when env var does not exist",
			pod:  &testPod,
			tester: ModificationTestFunc(func(pod *corev1.Pod, c Container) {
				c.setOrAppendEnvVar(pod, corev1.EnvVar{Name: "OTEL_ENV_VALUE2", Value: "my-env-value2"})
			}),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE1", Value: "my-env-value1"},
				{Name: "OTEL_ENV_VALUEFROM_CONFIGMAP1", ValueFrom: NewConfigMapKeyRef("my-refconfig", "ref-value1", nil)},
				{Name: "OTEL_ENV_VALUEFROM_SECRET1", ValueFrom: NewSecretKeyRef("my-secret-refconfig", "secret-ref-value1", nil)},
				{Name: "OTEL_ENV_VALUE2", Value: "my-env-value2"},
			},
		},
		{
			name: "Test setOrAppendEnvVar when env var exists in Env",
			pod:  &testPod,
			tester: ModificationTestFunc(func(pod *corev1.Pod, c Container) {
				c.setOrAppendEnvVar(pod, corev1.EnvVar{Name: "OTEL_ENV_VALUE1", Value: "my-overridden-value1"})
			}),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE1", Value: "my-overridden-value1"},
				{Name: "OTEL_ENV_VALUEFROM_CONFIGMAP1", ValueFrom: NewConfigMapKeyRef("my-refconfig", "ref-value1", nil)},
				{Name: "OTEL_ENV_VALUEFROM_SECRET1", ValueFrom: NewSecretKeyRef("my-secret-refconfig", "secret-ref-value1", nil)},
			},
		},
		{
			name: "Test setOrAppendEnvVar when env var exists as ConfigMapKeyRef",
			pod:  &testPod,
			tester: ModificationTestFunc(func(pod *corev1.Pod, c Container) {
				c.setOrAppendEnvVar(pod, corev1.EnvVar{Name: "OTEL_ENV_VALUEFROM_CONFIGMAP1", Value: "my-overridden-value1"})
			}),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE1", Value: "my-env-value1"},
				{Name: "OTEL_ENV_VALUEFROM_CONFIGMAP1", Value: "my-overridden-value1"},
				{Name: "OTEL_ENV_VALUEFROM_SECRET1", ValueFrom: NewSecretKeyRef("my-secret-refconfig", "secret-ref-value1", nil)},
			},
		},
		{
			name: "Test setOrAppendEnvVar on existing unsupported env var",
			pod:  &testPod,
			tester: ModificationTestFunc(func(pod *corev1.Pod, c Container) {
				c.setOrAppendEnvVar(pod, corev1.EnvVar{Name: "OTEL_ENV_VALUEFROM_SECRET1", Value: "my-overridden-value1"})
			}),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE1", Value: "my-env-value1"},
				{Name: "OTEL_ENV_VALUEFROM_CONFIGMAP1", ValueFrom: NewConfigMapKeyRef("my-refconfig", "ref-value1", nil)},
				{Name: "OTEL_ENV_VALUEFROM_SECRET1", Value: "my-overridden-value1"},
			},
		},
		{
			name: "Test moveToListEnd when env var exists",
			pod:  &testPod,
			tester: ModificationTestFunc(func(pod *corev1.Pod, c Container) {
				c.moveToListEnd(pod, "OTEL_ENV_VALUE1")
			}),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUEFROM_CONFIGMAP1", ValueFrom: NewConfigMapKeyRef("my-refconfig", "ref-value1", nil)},
				{Name: "OTEL_ENV_VALUEFROM_SECRET1", ValueFrom: NewSecretKeyRef("my-secret-refconfig", "secret-ref-value1", nil)},
				{Name: "OTEL_ENV_VALUE1", Value: "my-env-value1"},
			},
		},
		{
			name: "Test moveToListEnd when env var does not exist",
			pod:  &testPod,
			tester: ModificationTestFunc(func(pod *corev1.Pod, c Container) {
				c.moveToListEnd(pod, "OTEL_ENV_VALUE2")
			}),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE1", Value: "my-env-value1"},
				{Name: "OTEL_ENV_VALUEFROM_CONFIGMAP1", ValueFrom: NewConfigMapKeyRef("my-refconfig", "ref-value1", nil)},
				{Name: "OTEL_ENV_VALUEFROM_SECRET1", ValueFrom: NewSecretKeyRef("my-secret-refconfig", "secret-ref-value1", nil)},
			},
		},
	}

	err := k8sClient.Create(context.Background(), &testNs)
	require.NoError(t, err)
	defer func() {
		_ = k8sClient.Delete(context.Background(), &testNs)
	}()

	for _, cm := range testCm {
		cm := cm
		err = k8sClient.Create(context.Background(), &cm)
		require.NoError(t, err)
		//goland:noinspection GoDeferInLoop
		defer func() {
			_ = k8sClient.Delete(context.Background(), &cm)
		}()
	}
	for _, secret := range testSecret {
		secret := secret
		err = k8sClient.Create(context.Background(), &secret)
		require.NoError(t, err)
		//goland:noinspection GoDeferInLoop
		defer func() {
			_ = k8sClient.Delete(context.Background(), &secret)
		}()
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			pod := test.pod.DeepCopy()

			container, err := NewContainer(k8sClient, context.Background(), logr.Discard(), testNs.Name, pod, 0)
			require.NoError(t, err)
			test.tester.Test(pod, container)
			assert.Equal(t, test.expected, pod.Spec.Containers[0].Env)
		})
	}
}

type LoadConfigMapTester interface {
	Test(client client.Reader, ctx context.Context, configMaps map[string]*corev1.ConfigMap) (*corev1.ConfigMap, error)
}

type LoadConfigMapTestFunc func(client client.Reader, ctx context.Context, configMaps map[string]*corev1.ConfigMap) (*corev1.ConfigMap, error)

func (f LoadConfigMapTestFunc) Test(client client.Reader, ctx context.Context, configMaps map[string]*corev1.ConfigMap) (*corev1.ConfigMap, error) {
	return f(client, ctx, configMaps)
}

var _ LoadConfigMapTester = LoadConfigMapTestFunc(nil)

func TestLoadConfigMap(t *testing.T) {
	testNs := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "loadconfigmap",
		},
	}
	testCm := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-config",
				Namespace: "loadconfigmap",
			},
			Data: map[string]string{
				"OTEL_ENVFROM_VALUE1": "my-envfrom-value1",
			},
		},
	}

	tests := []struct {
		name     string
		tester   LoadConfigMapTester
		err      string
		expected map[string]map[string]string
	}{
		{
			name: "Test loading of ConfigMap",
			tester: LoadConfigMapTestFunc(func(client client.Reader, ctx context.Context, configMaps map[string]*corev1.ConfigMap) (*corev1.ConfigMap, error) {
				return getOrLoadResource(client, ctx, "loadconfigmap", configMaps, "my-config")
			}),
			expected: map[string]map[string]string{
				"my-config": {"OTEL_ENVFROM_VALUE1": "my-envfrom-value1"},
			},
		},
		{
			name: "Test cached loading of ConfigMap",
			tester: LoadConfigMapTestFunc(func(client client.Reader, ctx context.Context, configMaps map[string]*corev1.ConfigMap) (*corev1.ConfigMap, error) {
				_, _ = getOrLoadResource(client, ctx, "loadconfigmap", configMaps, "my-config")
				return getOrLoadResource(nil, nil, "loadconfigmap", configMaps, "my-config")
			}),
			expected: map[string]map[string]string{
				"my-config": {"OTEL_ENVFROM_VALUE1": "my-envfrom-value1"},
			},
		},
		{
			name: "Test missing ConfigMap",
			tester: LoadConfigMapTestFunc(func(client client.Reader, ctx context.Context, configMaps map[string]*corev1.ConfigMap) (*corev1.ConfigMap, error) {
				return getOrLoadResource(client, ctx, "loadconfigmap", configMaps, "nonexisting-config")
			}),
			err: "failed to get ConfigMap loadconfigmap/nonexisting-config: ",
			expected: map[string]map[string]string{
				"nonexisting-config": nil,
			},
		},
		{
			name: "Test cached missing ConfigMap",
			tester: LoadConfigMapTestFunc(func(client client.Reader, ctx context.Context, configMaps map[string]*corev1.ConfigMap) (*corev1.ConfigMap, error) {
				_, _ = getOrLoadResource(client, ctx, "loadconfigmap", configMaps, "nonexisting-config")
				return getOrLoadResource(nil, nil, "loadconfigmap", configMaps, "nonexisting-config")
			}),
			err: "failed to get ConfigMap loadconfigmap/nonexisting-config",
			expected: map[string]map[string]string{
				"nonexisting-config": nil,
			},
		},
		{
			name: "Test unconfigured",
			tester: LoadConfigMapTestFunc(func(client client.Reader, ctx context.Context, configMaps map[string]*corev1.ConfigMap) (*corev1.ConfigMap, error) {
				return getOrLoadResource(nil, nil, "loadconfigmap", configMaps, "my-config")
			}),
			err: "cannot load ConfigMap loadconfigmap/my-config",
			expected: map[string]map[string]string{
				"my-config": nil,
			},
		},
	}

	err := k8sClient.Create(context.Background(), &testNs)
	require.NoError(t, err)
	defer func() {
		_ = k8sClient.Delete(context.Background(), &testNs)
	}()

	for _, cm := range testCm {
		cm := cm
		err = k8sClient.Create(context.Background(), &cm)
		require.NoError(t, err)
		//goland:noinspection GoDeferInLoop
		defer func() {
			_ = k8sClient.Delete(context.Background(), &cm)
		}()
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			maps := map[string]*corev1.ConfigMap{}
			cm, err := test.tester.Test(k8sClient, context.Background(), maps)

			if test.err == "" {
				assert.NoError(t, err)
				assert.Equal(t, testCm[0].Data, cm.Data)
			} else {
				if assert.Error(t, err) {
					assert.Contains(t, err.Error(), test.err)
				}
			}

			if test.expected == nil {
				assert.Nil(t, cm)
			} else {
				for key, value := range test.expected {
					cm, ok := maps[key]
					assert.True(t, ok)
					if value == nil {
						assert.Nil(t, cm)
					} else {
						if assert.NotNil(t, cm) {
							assert.Equal(t, value, cm.Data)
						}
					}
				}
				assert.Len(t, maps, len(test.expected))
			}
		})
	}
}

func TestResolving(t *testing.T) {
	true := true
	false := false

	testNs := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "validations",
		},
	}
	testCm := []corev1.ConfigMap{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-config",
				Namespace: "validations",
			},
			Data: map[string]string{
				"OTEL_ENVFROM_VALUE1": "my-envfrom-value1",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-refconfig",
				Namespace: "validations",
			},
			Data: map[string]string{
				"ref-value1": "my-valuefrom-value1",
			},
		},
	}
	testSecret := []corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-secret",
				Namespace: "validations",
			},
			Data: map[string][]byte{
				"OTEL_ENVFROM_SECRET_VALUE1": []byte("my-secret-value1"),
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-refsecret",
				Namespace: "validations",
			},
			Data: map[string][]byte{
				"secret-ref-value1": []byte("my-valuefrom-value1"),
			},
		},
	}
	testPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-pod",
			Namespace: "validations",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "app",
					EnvFrom: []corev1.EnvFromSource{
						NewConfigMapRef("my-config", "", nil),
						NewSecretRef("my-secret", "", nil),
					},
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_ENV_VALUE1",
							Value: "my-env-value1",
						},
						{
							Name:      "OTEL_ENV_VALUEFROM_CONFIGMAP1",
							ValueFrom: NewConfigMapKeyRef("my-refconfig", "ref-value1", nil),
						},
						{
							Name:      "OTEL_ENV_VALUEFROM_CONFIGMAP2",
							ValueFrom: NewConfigMapKeyRef("my-refconfig", "ref-value-nonexisting", nil),
						},
						{
							Name:      "OTEL_ENV_VALUEFROM_CONFIGMAP3",
							ValueFrom: NewConfigMapKeyRef("my-refconfig", "ref-value-nonexisting", &false),
						},
						{
							Name:      "OTEL_ENV_VALUEFROM_CONFIGMAP4",
							ValueFrom: NewConfigMapKeyRef("my-refconfig", "ref-value-nonexisting", &true),
						},
						{
							Name:      "OTEL_ENV_VALUEFROM_CONFIGMAP5",
							ValueFrom: NewConfigMapKeyRef("my-refconfig-nonexisting", "ref-value1", nil),
						},
						{
							Name:      "OTEL_ENV_VALUEFROM_CONFIGMAP6",
							ValueFrom: NewConfigMapKeyRef("my-refconfig-nonexisting", "ref-value1", &false),
						},
						{
							Name:      "OTEL_ENV_VALUEFROM_CONFIGMAP7",
							ValueFrom: NewConfigMapKeyRef("my-refconfig-nonexisting", "ref-value1", &true),
						},
						{
							Name:      "OTEL_ENV_VALUEFROM_SECRET1",
							ValueFrom: NewSecretKeyRef("my-refsecret", "secret-ref-value1", nil),
						},
						{
							Name:      "OTEL_ENV_VALUEFROM_FIELD1",
							ValueFrom: NewFieldRef("spec.nodeName"),
						},
						{
							Name:      "OTEL_ENV_VALUEFROM_RESOURCEFIELD1",
							ValueFrom: NewResourceFieldRef("app", "limits.cpu"),
						},
						{
							Name:      "OTEL_ENV_VALUEFROM_INVALID",
							ValueFrom: &corev1.EnvVarSource{},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name            string
		variable        string
		err             string
		expectedExists  bool
		expectedResolve string
	}{
		{
			name:            "Test existing variable",
			variable:        "OTEL_ENV_VALUE1",
			expectedExists:  true,
			expectedResolve: "my-env-value1",
		},
		{
			name:            "Test non-existing variable",
			variable:        "OTEL_ENV_VALUE_NONEXISTING",
			expectedExists:  false,
			expectedResolve: "",
		},
		{
			name:            "Test existing ConfigMap variable",
			variable:        "OTEL_ENVFROM_VALUE1",
			expectedExists:  true,
			expectedResolve: "my-envfrom-value1",
		},
		{
			name:            "Test existing ConfigMapKeyRef variable",
			variable:        "OTEL_ENV_VALUEFROM_CONFIGMAP1",
			expectedExists:  true,
			expectedResolve: "my-valuefrom-value1",
		},
		{
			name:           "Test implicitly mandatory non-existing ConfigMapKeyRef variable",
			variable:       "OTEL_ENV_VALUEFROM_CONFIGMAP2",
			expectedExists: true,
			err:            "failed to resolve environment variable OTEL_ENV_VALUEFROM_CONFIGMAP2, key ref-value-nonexisting not found in ConfigMap validations/my-refconfig",
		},
		{
			name:           "Test explicitly mandatory non-existing ConfigMapKeyRef variable",
			variable:       "OTEL_ENV_VALUEFROM_CONFIGMAP3",
			expectedExists: true,
			err:            "failed to resolve environment variable OTEL_ENV_VALUEFROM_CONFIGMAP3, key ref-value-nonexisting not found in ConfigMap validations/my-refconfig",
		},
		{
			name:            "Test optional non-existing ConfigMapKeyRef variable",
			variable:        "OTEL_ENV_VALUEFROM_CONFIGMAP4",
			expectedExists:  true,
			expectedResolve: "",
		},
		{
			name:           "Test implicitly mandatory variable of non-existing ConfigMap",
			variable:       "OTEL_ENV_VALUEFROM_CONFIGMAP5",
			expectedExists: true,
			err:            "failed to resolve environment variable OTEL_ENV_VALUEFROM_CONFIGMAP5: ",
		},
		{
			name:           "Test explicitly mandatory variable of non-existing ConfigMap",
			variable:       "OTEL_ENV_VALUEFROM_CONFIGMAP6",
			expectedExists: true,
			err:            "failed to resolve environment variable OTEL_ENV_VALUEFROM_CONFIGMAP6: ",
		},
		{
			name:            "Test optional variable of non-existing ConfigMap",
			variable:        "OTEL_ENV_VALUEFROM_CONFIGMAP7",
			expectedExists:  true,
			expectedResolve: "",
		},
		{
			name:           "Test unsupported existing Secret variable",
			variable:       "OTEL_ENV_VALUEFROM_SECRET1",
			expectedExists: true,
			err:            "the container defines env var value via ValueFrom.SecretKeyRef, envVar: OTEL_ENV_VALUEFROM_SECRET1",
		},
		{
			name:           "Test unsupported existing FieldRef variable",
			variable:       "OTEL_ENV_VALUEFROM_FIELD1",
			expectedExists: true,
			err:            "the container defines env var value via ValueFrom.FieldRef, envVar: OTEL_ENV_VALUEFROM_FIELD1",
		},
		{
			name:           "Test unsupported existing ResourceFieldRef variable",
			variable:       "OTEL_ENV_VALUEFROM_RESOURCEFIELD1",
			expectedExists: true,
			err:            "the container defines env var value via ValueFrom.ResourceFieldRef, envVar: OTEL_ENV_VALUEFROM_RESOURCEFIELD1",
		},
		{
			name:           "Test invalid variable",
			variable:       "OTEL_ENV_VALUEFROM_INVALID",
			expectedExists: true,
			err:            "the container defines env var value via ValueFrom, envVar: OTEL_ENV_VALUEFROM_INVALID",
		},
	}

	err := k8sClient.Create(context.Background(), &testNs)
	require.NoError(t, err)
	defer func() {
		_ = k8sClient.Delete(context.Background(), &testNs)
	}()

	for _, cm := range testCm {
		cm := cm
		err = k8sClient.Create(context.Background(), &cm)
		require.NoError(t, err)
		//goland:noinspection GoDeferInLoop
		defer func() {
			_ = k8sClient.Delete(context.Background(), &cm)
		}()
	}
	for _, secret := range testSecret {
		secret := secret
		err = k8sClient.Create(context.Background(), &secret)
		require.NoError(t, err)
		//goland:noinspection GoDeferInLoop
		defer func() {
			_ = k8sClient.Delete(context.Background(), &secret)
		}()
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			c, err := NewContainer(k8sClient, context.Background(), logr.Discard(), testNs.Name, &testPod, 0)
			require.NoError(t, err)

			exists := c.exists(&testPod, test.variable)
			errValidate := c.validate(&testPod, test.variable)
			resolved, errGet := c.getOrMakeEnvVar(&testPod, test.variable)

			assert.Equal(t, test.expectedExists, exists)

			if test.err == "" {
				assert.NoError(t, errValidate)
				assert.NoError(t, errGet)
				assert.Equal(t, test.variable, resolved.Name)
				assert.Equal(t, test.expectedResolve, resolved.Value)
			} else {
				if assert.Error(t, errValidate) {
					assert.Contains(t, errValidate.Error(), test.err)
				}
				if assert.Error(t, errGet) {
					assert.Contains(t, errGet.Error(), test.err)
				}
			}
		})
	}
}

func TestConcatenations(t *testing.T) {
	testNs := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "concatenations",
		},
	}
	testPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-pod",
			Namespace: "concatenations",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "app",
					Env: []corev1.EnvVar{
						{
							Name:  "OTEL_ENV_VALUE",
							Value: "my-env-value",
						},
						{
							Name:  "OTEL_ENV_COLON",
							Value: "my-env-value:",
						},
						{
							Name:      "OTEL_ENV_INVALID",
							ValueFrom: NewConfigMapKeyRef("non-existing", "some-key", nil),
						},
					},
				},
			},
		},
	}

	//goland:noinspection SpellCheckingInspection
	tests := []struct {
		name      string
		variable  string
		value     string
		concatter Concatter
		err       string
		expected  []corev1.EnvVar
	}{
		{
			name:      "Test concatenation on non-existing variable",
			variable:  "OTEL_ENV_NEW",
			value:     "added",
			concatter: ConcatFunc(concatWithColon),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE", Value: "my-env-value"},
				{Name: "OTEL_ENV_COLON", Value: "my-env-value:"},
				{Name: "OTEL_ENV_INVALID", ValueFrom: NewConfigMapKeyRef("non-existing", "some-key", nil)},
				{Name: "OTEL_ENV_NEW", Value: "added"},
			},
		},
		{
			name:      "Test concatenation with colon",
			variable:  "OTEL_ENV_VALUE",
			value:     "added",
			concatter: ConcatFunc(concatWithColon),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE", Value: "my-env-value:added"},
				{Name: "OTEL_ENV_COLON", Value: "my-env-value:"},
				{Name: "OTEL_ENV_INVALID", ValueFrom: NewConfigMapKeyRef("non-existing", "some-key", nil)},
			},
		},
		{
			name:      "Test concatenation of empty value",
			variable:  "OTEL_ENV_VALUE",
			value:     "",
			concatter: ConcatFunc(concatWithColon),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE", Value: "my-env-value"},
				{Name: "OTEL_ENV_COLON", Value: "my-env-value:"},
				{Name: "OTEL_ENV_INVALID", ValueFrom: NewConfigMapKeyRef("non-existing", "some-key", nil)},
			},
		},
		{
			name:      "Test concatenation not adding redundant colon",
			variable:  "OTEL_ENV_COLON",
			value:     "added",
			concatter: ConcatFunc(concatWithColon),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE", Value: "my-env-value"},
				{Name: "OTEL_ENV_COLON", Value: "my-env-value:added"},
				{Name: "OTEL_ENV_INVALID", ValueFrom: NewConfigMapKeyRef("non-existing", "some-key", nil)},
			},
		},
		{
			name:      "Test concatenation not adding redundant colon on both sides",
			variable:  "OTEL_ENV_COLON",
			value:     ":added",
			concatter: ConcatFunc(concatWithColon),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE", Value: "my-env-value"},
				{Name: "OTEL_ENV_COLON", Value: "my-env-value:added"},
				{Name: "OTEL_ENV_INVALID", ValueFrom: NewConfigMapKeyRef("non-existing", "some-key", nil)},
			},
		},
		{
			name:      "Test concatenation with comma",
			variable:  "OTEL_ENV_VALUE",
			value:     "added",
			concatter: ConcatFunc(concatWithComma),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE", Value: "my-env-value,added"},
				{Name: "OTEL_ENV_COLON", Value: "my-env-value:"},
				{Name: "OTEL_ENV_INVALID", ValueFrom: NewConfigMapKeyRef("non-existing", "some-key", nil)},
			},
		},
		{
			name:      "Test concatenation with space",
			variable:  "OTEL_ENV_VALUE",
			value:     "added",
			concatter: ConcatFunc(concatWithSpace),
			expected: []corev1.EnvVar{
				{Name: "OTEL_ENV_VALUE", Value: "my-env-value added"},
				{Name: "OTEL_ENV_COLON", Value: "my-env-value:"},
				{Name: "OTEL_ENV_INVALID", ValueFrom: NewConfigMapKeyRef("non-existing", "some-key", nil)},
			},
		},
		{
			name:      "Test concatenation with non-resolvable variable",
			variable:  "OTEL_ENV_INVALID",
			value:     "added",
			concatter: ConcatFunc(concatWithColon),
			err:       "failed to resolve environment variable OTEL_ENV_INVALID: ",
		},
		{
			name:      "Test concatenation with nil concatter",
			variable:  "OTEL_ENV_VALUE",
			value:     "added",
			concatter: nil,
			err:       "concatter is nil",
		},
	}

	err := k8sClient.Create(context.Background(), &testNs)
	require.NoError(t, err)
	defer func() {
		_ = k8sClient.Delete(context.Background(), &testNs)
	}()

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			pod := testPod.DeepCopy()
			c, err := NewContainer(k8sClient, context.Background(), logr.Discard(), testNs.Name, pod, 0)
			require.NoError(t, err)

			err = c.appendOrConcat(pod, test.variable, test.value, test.concatter)

			if test.err == "" {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, pod.Spec.Containers[0].Env)
			} else {
				if assert.Error(t, err) {
					assert.Contains(t, err.Error(), test.err)
				}
			}
		})
	}
}
