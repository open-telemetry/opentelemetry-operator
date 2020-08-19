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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/open-telemetry/opentelemetry-operator/pkg/collector"
)

// DaemonSets reconciles the daemon set(s) required for the instance in the current context
func DaemonSets(ctx context.Context, params Params) error {
	desired := []appsv1.DaemonSet{}
	if params.Instance.Spec.Mode == "daemonset" {
		desired = append(desired, desiredDaemonSet(ctx, params))
	}

	// first, handle the create/update parts
	if err := expectedDaemonSets(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the expected daemon sets: %v", err)
	}

	// then, delete the extra objects
	if err := deleteDaemonSets(ctx, params, desired); err != nil {
		return fmt.Errorf("failed to reconcile the daemon sets to be deleted: %v", err)
	}

	return nil
}

func desiredDaemonSet(ctx context.Context, params Params) appsv1.DaemonSet {
	name := fmt.Sprintf("%s-collector", params.Instance.Name)

	image := params.Instance.Spec.Image
	if len(image) == 0 {
		image = params.Config.CollectorImage()
	}

	labels := collector.Labels(params.Instance)
	labels["app.kubernetes.io/name"] = name

	annotations := params.Instance.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}

	annotations["prometheus.io/scrape"] = "true"
	annotations["prometheus.io/port"] = "8888"
	annotations["prometheus.io/path"] = "/metrics"

	argsMap := params.Instance.Spec.Args
	if argsMap == nil {
		argsMap = map[string]string{}
	}

	if _, exists := argsMap["config"]; exists {
		params.Log.Info("the 'config' flag isn't allowed and is being ignored")
	}

	// this effectively overrides any 'config' entry that might exist in the CR
	argsMap["config"] = fmt.Sprintf("/conf/%s", params.Config.CollectorConfigMapEntry())

	var args []string
	for k, v := range argsMap {
		args = append(args, fmt.Sprintf("--%s=%s", k, v))
	}

	configMapVolumeName := fmt.Sprintf("otc-internal-%s", name)
	volumeMounts := []corev1.VolumeMount{{
		Name:      configMapVolumeName,
		MountPath: "/conf",
	}}
	volumes := []corev1.Volume{{
		Name: configMapVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: name},
				Items: []corev1.KeyToPath{{
					Key:  params.Config.CollectorConfigMapEntry(),
					Path: params.Config.CollectorConfigMapEntry(),
				}},
			},
		},
	}}

	if len(params.Instance.Spec.VolumeMounts) > 0 {
		volumeMounts = append(volumeMounts, params.Instance.Spec.VolumeMounts...)
	}

	if len(params.Instance.Spec.Volumes) > 0 {
		volumes = append(volumes, params.Instance.Spec.Volumes...)
	}

	return appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   params.Instance.Namespace,
			Labels:      labels,
			Annotations: params.Instance.Annotations,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: ServiceAccountNameFor(params.Instance),
					Containers: []corev1.Container{{
						Name:         "opentelemetry-collector",
						Image:        image,
						VolumeMounts: volumeMounts,
						Args:         args,
					}},
					Volumes: volumes,
				},
			},
		},
	}

}

func expectedDaemonSets(ctx context.Context, params Params, expected []appsv1.DaemonSet) error {
	for _, obj := range expected {
		desired := obj

		controllerutil.SetControllerReference(&params.Instance, &desired, params.Scheme)

		existing := &appsv1.DaemonSet{}
		nns := types.NamespacedName{Namespace: desired.Namespace, Name: desired.Name}
		err := params.Client.Get(ctx, nns, existing)
		if err != nil && k8serrors.IsNotFound(err) {
			if err := params.Client.Create(ctx, &desired); err != nil {
				return fmt.Errorf("failed to create: %w", err)
			}
			params.Log.V(2).Info("created", "daemonset.name", desired.Name, "daemonset.namespace", desired.Namespace)
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

		updated.Spec = desired.Spec
		updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences

		for k, v := range desired.ObjectMeta.Annotations {
			updated.ObjectMeta.Annotations[k] = v
		}
		for k, v := range desired.ObjectMeta.Labels {
			updated.ObjectMeta.Labels[k] = v
		}

		if err := params.Client.Update(ctx, updated); err != nil {
			return fmt.Errorf("failed to apply changes: %w", err)
		}

		params.Log.V(2).Info("applied", "daemonset.name", desired.Name, "daemonset.namespace", desired.Namespace)
	}

	return nil
}

func deleteDaemonSets(ctx context.Context, params Params, expected []appsv1.DaemonSet) error {
	opts := []client.ListOption{
		client.InNamespace(params.Instance.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", params.Instance.Namespace, params.Instance.Name),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	}
	list := &appsv1.DaemonSetList{}
	if err := params.Client.List(ctx, list, opts...); err != nil {
		return fmt.Errorf("failed to list: %w", err)
	}

	for _, existing := range list.Items {
		del := true
		for _, keep := range expected {
			if keep.Name == existing.Name && keep.Namespace == existing.Namespace {
				del = false
			}
		}

		if del {
			if err := params.Client.Delete(ctx, &existing); err != nil {
				return fmt.Errorf("failed to delete: %w", err)
			}
			params.Log.V(2).Info("deleted", "daemonset.name", existing.Name, "daemonset.namespace", existing.Namespace)
		}
	}

	return nil
}
