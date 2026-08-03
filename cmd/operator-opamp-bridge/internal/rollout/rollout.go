// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package rollout

import (
	"context"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RestartAnnotation is the pod template annotation used to trigger a rolling restart,
// identical to the annotation set by `kubectl rollout restart`.
const RestartAnnotation = "kubectl.kubernetes.io/restartedAt"

// TriggerRollout annotates the pod template of the named workload with the current
// timestamp, causing Kubernetes to perform a rolling restart. workloadType must be one
// of "deployment", "daemonset", or "statefulset" (case-insensitive).
//
// Uses Get -> DeepCopy -> mutate -> MergeFrom + Patch so that only the annotation field
// is sent in the PATCH body. This avoids the conflict window of a full Update: the patch
// is merged server-side and is only rejected if the object no longer exists, not on any
// concurrent spec change.
func TriggerRollout(ctx context.Context, k8sClient client.Client, namespace, workloadType, workloadName string) error {
	restartVal := time.Now().Format(time.RFC3339)
	switch strings.ToLower(workloadType) {
	case "deployment":
		return patchDeployment(ctx, k8sClient, namespace, workloadName, restartVal)
	case "daemonset":
		return patchDaemonSet(ctx, k8sClient, namespace, workloadName, restartVal)
	case "statefulset":
		return patchStatefulSet(ctx, k8sClient, namespace, workloadName, restartVal)
	default:
		return fmt.Errorf("unsupported workload type %q for rollout restart", workloadType)
	}
}

func patchDeployment(ctx context.Context, k8sClient client.Client, namespace, name, restartVal string) error {
	deploy := &appsv1.Deployment{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, deploy); err != nil {
		return fmt.Errorf("failed to get Deployment %s/%s for restart: %w", namespace, name, err)
	}
	updated := deploy.DeepCopy()
	if updated.Spec.Template.Annotations == nil {
		updated.Spec.Template.Annotations = map[string]string{}
	}
	updated.Spec.Template.Annotations[RestartAnnotation] = restartVal
	if err := k8sClient.Patch(ctx, updated, client.MergeFrom(deploy)); err != nil {
		return fmt.Errorf("failed to restart Deployment %s/%s: %w", namespace, name, err)
	}
	return nil
}

func patchDaemonSet(ctx context.Context, k8sClient client.Client, namespace, name, restartVal string) error {
	ds := &appsv1.DaemonSet{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, ds); err != nil {
		return fmt.Errorf("failed to get DaemonSet %s/%s for restart: %w", namespace, name, err)
	}
	updated := ds.DeepCopy()
	if updated.Spec.Template.Annotations == nil {
		updated.Spec.Template.Annotations = map[string]string{}
	}
	updated.Spec.Template.Annotations[RestartAnnotation] = restartVal
	if err := k8sClient.Patch(ctx, updated, client.MergeFrom(ds)); err != nil {
		return fmt.Errorf("failed to restart DaemonSet %s/%s: %w", namespace, name, err)
	}
	return nil
}

func patchStatefulSet(ctx context.Context, k8sClient client.Client, namespace, name, restartVal string) error {
	sts := &appsv1.StatefulSet{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, sts); err != nil {
		return fmt.Errorf("failed to get StatefulSet %s/%s for restart: %w", namespace, name, err)
	}
	updated := sts.DeepCopy()
	if updated.Spec.Template.Annotations == nil {
		updated.Spec.Template.Annotations = map[string]string{}
	}
	updated.Spec.Template.Annotations[RestartAnnotation] = restartVal
	if err := k8sClient.Patch(ctx, updated, client.MergeFrom(sts)); err != nil {
		return fmt.Errorf("failed to restart StatefulSet %s/%s: %w", namespace, name, err)
	}
	return nil
}
