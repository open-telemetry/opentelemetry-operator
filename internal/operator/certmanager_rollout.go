// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/autodetectutils"
)

const certManagerRolloutAnnotation = "opentelemetry.io/cert-manager-availability-restart"

type certManagerRollout struct {
	clientset kubernetes.Interface
}

func (r *certManagerRollout) requestRollout(ctx context.Context) error {
	namespace, err := autodetectutils.GetOperatorNamespace()
	if err != nil {
		return fmt.Errorf("failed to determine operator namespace: %w", err)
	}
	podName, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to determine operator pod name: %w", err)
	}
	return r.rolloutDeployment(ctx, namespace, podName)
}

func (r *certManagerRollout) rolloutDeployment(ctx context.Context, namespace, podName string) error {
	pod, err := r.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get operator pod: %w", err)
	}
	rsRef := metav1.GetControllerOf(pod)
	if rsRef == nil || rsRef.Kind != "ReplicaSet" {
		return fmt.Errorf("operator pod %q is not owned by a ReplicaSet", podName)
	}
	rs, err := r.clientset.AppsV1().ReplicaSets(namespace).Get(ctx, rsRef.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get operator ReplicaSet: %w", err)
	}
	if rs.UID != rsRef.UID {
		return fmt.Errorf("operator ReplicaSet %q has a different UID", rs.Name)
	}
	deploymentRef := metav1.GetControllerOf(rs)
	if deploymentRef == nil || deploymentRef.Kind != "Deployment" {
		return fmt.Errorf("operator ReplicaSet %q is not owned by a Deployment", rs.Name)
	}
	deployments := r.clientset.AppsV1().Deployments(namespace)
	deployment, err := deployments.Get(ctx, deploymentRef.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get operator Deployment: %w", err)
	}
	if deployment.UID != deploymentRef.UID {
		return fmt.Errorf("operator Deployment %q has a different UID", deployment.Name)
	}
	if rs.Spec.Template.Annotations[certManagerRolloutAnnotation] != deployment.Spec.Template.Annotations[certManagerRolloutAnnotation] {
		return nil
	}

	patch, err := json.Marshal(map[string]any{
		"metadata": map[string]any{"resourceVersion": deployment.ResourceVersion},
		"spec": map[string]any{"template": map[string]any{"metadata": map[string]any{
			"annotations": map[string]string{certManagerRolloutAnnotation: string(uuid.NewUUID())},
		}}},
	})
	if err != nil {
		return fmt.Errorf("failed to create deployment patch: %w", err)
	}
	if _, err := deployments.Patch(ctx, deployment.Name, types.MergePatchType, patch, metav1.PatchOptions{}); err != nil {
		return fmt.Errorf("failed to patch operator Deployment: %w", err)
	}
	return nil
}
