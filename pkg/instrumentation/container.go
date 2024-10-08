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
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Container struct {
	client       client.Reader
	ctx          context.Context
	logger       logr.Logger
	namespace    string
	index        int
	inheritedEnv map[string]string
	configMaps   map[string]*corev1.ConfigMap
}

func NewContainer(client client.Reader, ctx context.Context, logger logr.Logger, namespace string, pod *corev1.Pod, index int) (Container, error) {
	if pod.Namespace != "" {
		namespace = pod.Namespace
	}
	container := &pod.Spec.Containers[index]

	configMaps := make(map[string]*corev1.ConfigMap)
	inheritedEnv := make(map[string]string)
	for _, envsFrom := range container.EnvFrom {
		if envsFrom.ConfigMapRef != nil {
			prefix := envsFrom.Prefix
			name := envsFrom.ConfigMapRef.Name
			if cm, err := getOrLoadResource(client, ctx, namespace, configMaps, name); err == nil {
				for k, v := range cm.Data {
					// Safely overwrite the value, last one from EnvFrom wins in Kubernetes, with the direct value
					// from the container itself taking precedence
					inheritedEnv[prefix+k] = v
				}
			} else if envsFrom.ConfigMapRef.Optional == nil || !*envsFrom.ConfigMapRef.Optional {
				return Container{}, fmt.Errorf("failed to load environment variables: %w", err)
			}
		} else if envsFrom.SecretRef != nil {
			logger.V(2).Info("ignoring SecretRef in EnvFrom", "container", container.Name, "secret", envsFrom.SecretRef.Name)
		}
	}

	if len(inheritedEnv) == 0 {
		inheritedEnv = nil
	}

	return Container{
		client:       client,
		ctx:          ctx,
		logger:       logger,
		namespace:    namespace,
		index:        index,
		inheritedEnv: inheritedEnv,
		configMaps:   configMaps,
	}, nil
}

func getOrLoadResource[T any, PT interface {
	client.Object
	*T
}](client client.Reader, ctx context.Context, namespace string, cache map[string]*T, name string) (*T, error) {
	var obj T
	if cached, ok := cache[name]; ok {
		if cached != nil {
			return cached, nil
		} else {
			return nil, fmt.Errorf("failed to get %s %s/%s", reflect.TypeOf(obj).Name(), namespace, name)
		}
	}

	if client == nil || ctx == nil {
		// Cache error value
		cache[name] = nil
		return nil, fmt.Errorf("client or context is nil, cannot load %s %s/%s", reflect.TypeOf(obj).Name(), namespace, name)
	}

	err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, PT(&obj))
	if err != nil {
		// Cache error value
		cache[name] = nil
		return nil, fmt.Errorf("failed to get %s %s/%s: %w", reflect.TypeOf(obj).Name(), namespace, name, err)
	}

	cache[name] = &obj
	return &obj, nil
}

func (c *Container) validate(pod *corev1.Pod, envsToBeValidated ...string) error {
	// Try if the value is resolvable
	for _, envToBeValidated := range envsToBeValidated {
		for _, envVar := range pod.Spec.Containers[c.index].Env {
			if envVar.Name == envToBeValidated {
				if _, err := c.resolveEnvVar(envVar); err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}

func getContainerIndex(pod *corev1.Pod, containerName string) int {
	// We search for specific container to inject variables and if no one is found
	// We fall back to the first container
	var index = 0
	for idx, container := range pod.Spec.Containers {
		if container.Name == containerName {
			index = idx
		}
	}

	return index
}

func (c *Container) exists(pod *corev1.Pod, name string) bool {
	if found := existsEnvVarInEnv(pod.Spec.Containers[c.index].Env, name); found {
		return found
	}
	if _, found := c.inheritedEnv[name]; found {
		return found
	}
	return false
}

func (c *Container) setOrAppendEnvVar(pod *corev1.Pod, envVar corev1.EnvVar) {
	if idx, found := findEnvVarInEnv(pod.Spec.Containers[c.index].Env, envVar.Name); found {
		pod.Spec.Containers[c.index].Env[idx] = envVar
	} else {
		c.appendEnvVar(pod, envVar)
	}
}

func (c *Container) getOrMakeEnvVar(pod *corev1.Pod, name string) (corev1.EnvVar, error) {
	var envVar corev1.EnvVar
	var idx int
	var found bool
	if idx, found = findEnvVarInEnv(pod.Spec.Containers[c.index].Env, name); found {
		envVar = pod.Spec.Containers[c.index].Env[idx]
	} else if envVar, found = getEnvVarFromMap(c.inheritedEnv, name); found {
		// do nothing
	} else {
		envVar = corev1.EnvVar{Name: name}
	}
	return c.resolveEnvVar(envVar)
}

func (c *Container) resolveEnvVar(envVar corev1.EnvVar) (corev1.EnvVar, error) {
	if envVar.Value == "" && envVar.ValueFrom != nil {
		if envVar.ValueFrom.ConfigMapKeyRef != nil {
			configMapName := envVar.ValueFrom.ConfigMapKeyRef.Name
			configMapKey := envVar.ValueFrom.ConfigMapKeyRef.Key
			if cm, err := getOrLoadResource(c.client, c.ctx, c.namespace, c.configMaps, configMapName); err == nil {
				if value, ok := cm.Data[configMapKey]; ok {
					return corev1.EnvVar{Name: envVar.Name, Value: value}, nil
				} else if envVar.ValueFrom.ConfigMapKeyRef.Optional == nil || !*envVar.ValueFrom.ConfigMapKeyRef.Optional {
					return corev1.EnvVar{}, fmt.Errorf("failed to resolve environment variable %s, key %s not found in ConfigMap %s/%s", envVar.Name, configMapKey, c.namespace, configMapName)
				} else {
					return corev1.EnvVar{Name: envVar.Name, Value: ""}, nil
				}
			} else if envVar.ValueFrom.ConfigMapKeyRef.Optional == nil || !*envVar.ValueFrom.ConfigMapKeyRef.Optional {
				return corev1.EnvVar{}, fmt.Errorf("failed to resolve environment variable %s: %w", envVar.Name, err)
			} else {
				return corev1.EnvVar{Name: envVar.Name, Value: ""}, nil
			}
		} else {
			v := reflect.ValueOf(*envVar.ValueFrom)
			for i := 0; i < v.NumField(); i++ {
				if v.Field(i).Kind() == reflect.Ptr && !v.Field(i).IsNil() {
					return corev1.EnvVar{}, fmt.Errorf("the container defines env var value via ValueFrom.%s, envVar: %s", v.Type().Field(i).Name, envVar.Name)
				}
			}
			return corev1.EnvVar{}, fmt.Errorf("the container defines env var value via ValueFrom, envVar: %s", envVar.Name)
		}
	}
	return envVar, nil
}

func existsEnvVarInEnv(env []corev1.EnvVar, name string) bool {
	for i := range env {
		if env[i].Name == name {
			return true
		}
	}
	return false
}

func findEnvVarInEnv(env []corev1.EnvVar, name string) (int, bool) {
	for i := range env {
		if env[i].Name == name {
			return i, true
		}
	}
	return -1, false
}

func getEnvVarFromMap(env map[string]string, name string) (corev1.EnvVar, bool) {
	if value, ok := env[name]; ok {
		return corev1.EnvVar{Name: name, Value: value}, true
	}
	return corev1.EnvVar{}, false
}

func (c *Container) prependEnvVar(pod *corev1.Pod, envVar corev1.EnvVar) {
	pod.Spec.Containers[c.index].Env = append([]corev1.EnvVar{envVar}, pod.Spec.Containers[c.index].Env...)
}

func (c *Container) prepend(pod *corev1.Pod, name string, value string) {
	c.prependEnvVar(pod, corev1.EnvVar{Name: name, Value: value})
}

func (c *Container) appendEnvVar(pod *corev1.Pod, envVar corev1.EnvVar) {
	pod.Spec.Containers[c.index].Env = append(pod.Spec.Containers[c.index].Env, envVar)
}

func (c *Container) append(pod *corev1.Pod, name string, value string) {
	c.appendEnvVar(pod, corev1.EnvVar{Name: name, Value: value})
}

func (c *Container) prependIfNotExists(pod *corev1.Pod, name string, value string) {
	if !c.exists(pod, name) {
		c.prepend(pod, name, value)
	}
}

func (c *Container) prependEnvVarIfNotExists(pod *corev1.Pod, envVar corev1.EnvVar) {
	if !c.exists(pod, envVar.Name) {
		c.prependEnvVar(pod, envVar)
	}
}

func (c *Container) appendIfNotExists(pod *corev1.Pod, name string, value string) {
	if !c.exists(pod, name) {
		c.append(pod, name, value)
	}
}

func (c *Container) appendEnvVarIfNotExists(pod *corev1.Pod, envVar corev1.EnvVar) {
	if !c.exists(pod, envVar.Name) {
		c.appendEnvVar(pod, envVar)
	}
}

//goland:noinspection SpellCheckingInspection
type Concatter interface {
	Concat(vals ...string) string
}

type ConcatFunc func(vals ...string) string

func (f ConcatFunc) Concat(vals ...string) string {
	return f(vals...)
}

func (c *Container) appendOrConcat(pod *corev1.Pod, name string, value string, concatter Concatter) error {
	if concatter == nil {
		return fmt.Errorf("concatter is nil")
	}

	if envVar, err := c.getOrMakeEnvVar(pod, name); err == nil {
		envVar.Value = concatter.Concat(envVar.Value, value)
		c.setOrAppendEnvVar(pod, envVar)
		return nil
	} else {
		return err
	}
}

func (c *Container) moveToListEnd(pod *corev1.Pod, name string) {
	if idx, ok := findEnvVarInEnv(pod.Spec.Containers[c.index].Env, name); ok {
		envToMove := pod.Spec.Containers[c.index].Env[idx]
		envs := append(pod.Spec.Containers[c.index].Env[:idx], pod.Spec.Containers[c.index].Env[idx+1:]...)
		pod.Spec.Containers[c.index].Env = append(envs, envToMove)
	}
}

func concatWithCharacterChecked(val1, val2, char string) string {
	if val1 != "" {
		if val2 != "" {
			if val1[len(val1)-1:] == char {
				if val2[:1] == char {
					return val1 + val2[1:]
				} else {
					return val1 + val2
				}
			} else {
				return val1 + char + val2
			}
		} else {
			return val1
		}
	} else {
		return val2
	}
}

func concatWithCharacter(char string, vals ...string) string {
	result := ""
	for _, val := range vals {
		result = concatWithCharacterChecked(result, val, char)
	}
	return result
}

func concatWithSpace(vals ...string) string {
	return concatWithCharacter(" ", vals...)
}

func concatWithComma(vals ...string) string {
	return concatWithCharacter(",", vals...)
}

func concatWithColon(vals ...string) string {
	return concatWithCharacter(":", vals...)
}
