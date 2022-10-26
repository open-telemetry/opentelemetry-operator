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
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/open-telemetry/opentelemetry-operator/pkg/collector"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/targetallocator"
	ta "github.com/open-telemetry/opentelemetry-operator/pkg/targetallocator/adapters"
)

// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// ConfigMaps reconciles the config map(s) required for the instance in the current context.
func ConfigMaps(ctx context.Context, params Params) error {
	desired := []corev1.ConfigMap{
		desiredConfigMap(ctx, params),
	}

	if params.Instance.Spec.TargetAllocator.Enabled {
		cm, err := desiredTAConfigMap(params)
		if err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}
		desired = append(desired, cm)
	}

	// first, handle the create/update parts
	if err := expectedConfigMaps(ctx, params, desired, true); err != nil {
		return fmt.Errorf("failed to reconcile the expected configmaps: %w", err)
	}

	// then, delete the extra objects
	if err := deleteConfigMaps(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the configmaps to be deleted: %w", err)
	}

	return nil
}

func desiredConfigMap(_ context.Context, params Params) corev1.ConfigMap {
	name := naming.ConfigMap(params.Instance)
	version := strings.Split(params.Instance.Spec.Image, ":")
	labels := collector.Labels(params.Instance, []string{})
	labels["app.kubernetes.io/name"] = name
	if len(version) > 1 {
		labels["app.kubernetes.io/version"] = version[len(version)-1]
	} else {
		labels["app.kubernetes.io/version"] = "latest"
	}
	config, err := ReplaceConfig(params)
	if err != nil {
		params.Log.V(2).Info("failed to update prometheus config to use sharded targets: ", err)
	}

	return corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   params.Instance.Namespace,
			Labels:      labels,
			Annotations: params.Instance.Annotations,
		},
		Data: map[string]string{
			"collector.yaml": config,
		},
	}
}

func desiredTAConfigMap(params Params) (corev1.ConfigMap, error) {
	name := naming.TAConfigMap(params.Instance)
	version := strings.Split(params.Instance.Spec.Image, ":")
	labels := targetallocator.Labels(params.Instance)
	labels["app.kubernetes.io/name"] = name
	if len(version) > 1 {
		labels["app.kubernetes.io/version"] = version[len(version)-1]
	} else {
		labels["app.kubernetes.io/version"] = "latest"
	}

	promConfig, err := ta.ConfigToPromConfig(params.Instance.Spec.Config)
	if err != nil {
		return corev1.ConfigMap{}, err
	}

	taConfig := make(map[interface{}]interface{})
	taConfig["label_selector"] = map[string]string{
		"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.Instance.Namespace, params.Instance.Name),
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
		"app.kubernetes.io/component":  "opentelemetry-collector",
	}
	taConfig["config"] = promConfig
	if len(params.Instance.Spec.TargetAllocator.AllocationStrategy) > 0 {
		taConfig["allocation_strategy"] = params.Instance.Spec.TargetAllocator.AllocationStrategy
	} else {
		taConfig["allocation_strategy"] = "least-weighted"
	}

	if len(params.Instance.Spec.TargetAllocator.FilterStrategy) > 0 {
		taConfig["filter_strategy"] = params.Instance.Spec.TargetAllocator.FilterStrategy
	}

	taConfigYAML, err := yaml.Marshal(taConfig)
	if err != nil {
		return corev1.ConfigMap{}, err
	}

	return corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   params.Instance.Namespace,
			Labels:      labels,
			Annotations: params.Instance.Annotations,
		},
		Data: map[string]string{
			"targetallocator.yaml": string(taConfigYAML),
		},
	}, nil
}

func expectedConfigMaps(ctx context.Context, params Params, expected []corev1.ConfigMap, retry bool) error {
	for _, obj := range expected {
		desired := obj

		if err := controllerutil.SetControllerReference(&params.Instance, &desired, params.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference: %w", err)
		}

		existing := &corev1.ConfigMap{}
		nns := types.NamespacedName{Namespace: desired.Namespace, Name: desired.Name}
		err := params.Client.Get(ctx, nns, existing)
		if err != nil && errors.IsNotFound(err) {
			if err := params.Client.Create(ctx, &desired); err != nil {
				if errors.IsAlreadyExists(err) && retry {
					// let's try again? we probably had multiple updates at one, and now it exists already
					if err := expectedConfigMaps(ctx, params, expected, false); err != nil {
						// somethin else happened now...
						return err
					}

					// we succeeded in the retry, exit this attempt
					return nil
				}
				return fmt.Errorf("failed to create: %w", err)
			}
			params.Log.V(2).Info("created", "configmap.name", desired.Name, "configmap.namespace", desired.Namespace)
			continue
		} else if err != nil {
			return fmt.Errorf("failed to get: %w", err)
		}

		// it exists already, merge the two if the end result isn't identical to the existing one
		updated := existing.DeepCopy()
		if updated.Annotations == nil {
			updated.Annotations = map[string]string{}
		}
		if updated.Labels == nil {
			updated.Labels = map[string]string{}
		}

		updated.Data = desired.Data
		updated.BinaryData = desired.BinaryData
		updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences

		for k, v := range desired.ObjectMeta.Annotations {
			updated.ObjectMeta.Annotations[k] = v
		}
		for k, v := range desired.ObjectMeta.Labels {
			updated.ObjectMeta.Labels[k] = v
		}

		patch := client.MergeFrom(existing)

		if err := params.Client.Patch(ctx, updated, patch); err != nil {
			return fmt.Errorf("failed to apply changes: %w", err)
		}
		if configMapChanged(&desired, existing) {
			params.Recorder.Event(updated, "Normal", "ConfigUpdate ", fmt.Sprintf("OpenTelemetry Config changed - %s/%s", desired.Namespace, desired.Name))
		}

		params.Log.V(2).Info("applied", "configmap.name", desired.Name, "configmap.namespace", desired.Namespace)
	}

	return nil
}

func deleteConfigMaps(ctx context.Context, params Params, expected []corev1.ConfigMap) error {
	opts := []client.ListOption{
		client.InNamespace(params.Instance.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.Instance.Namespace, params.Instance.Name),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	}
	list := &corev1.ConfigMapList{}
	if err := params.Client.List(ctx, list, opts...); err != nil {
		return fmt.Errorf("failed to list: %w", err)
	}

	for i := range list.Items {
		existing := list.Items[i]
		del := true
		for _, keep := range expected {
			if keep.Name == existing.Name && keep.Namespace == existing.Namespace {
				del = false
				break
			}
		}

		if del {
			if err := params.Client.Delete(ctx, &existing); err != nil {
				return fmt.Errorf("failed to delete: %w", err)
			}
			params.Log.V(2).Info("deleted", "configmap.name", existing.Name, "configmap.namespace", existing.Namespace)
		}
	}

	return nil
}

func configMapChanged(desired *corev1.ConfigMap, actual *corev1.ConfigMap) bool {
	return !reflect.DeepEqual(desired.Data, actual.Data)

}
