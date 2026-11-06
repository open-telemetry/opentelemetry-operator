// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	kubeTesting "k8s.io/client-go/testing"
)

func certManagerRolloutObjects() (*corev1.Pod, *appsv1.ReplicaSet, *appsv1.Deployment) {
	controller := true
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "operator", Namespace: "operator-system", UID: "deployment-uid", ResourceVersion: "1"}}
	rs := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
		Name: "operator-replicaset", Namespace: deployment.Namespace, UID: "replicaset-uid",
		OwnerReferences: []metav1.OwnerReference{{Kind: "Deployment", Name: deployment.Name, UID: deployment.UID, Controller: &controller}},
	}}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name: "operator-pod", Namespace: deployment.Namespace,
		OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", Name: rs.Name, UID: rs.UID, Controller: &controller}},
	}}
	return pod, rs, deployment
}

func TestCertManagerRolloutDeployment(t *testing.T) {
	pod, rs, deployment := certManagerRolloutObjects()
	clientset := fake.NewClientset(pod, rs, deployment)
	rollout := certManagerRollout{clientset: clientset}
	ctx := context.Background()

	require.NoError(t, rollout.rolloutDeployment(ctx, pod.Namespace, pod.Name))
	updated, err := clientset.AppsV1().Deployments(pod.Namespace).Get(ctx, deployment.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, updated.Spec.Template.Annotations[certManagerRolloutAnnotation])

	clientset.ClearActions()
	require.NoError(t, rollout.rolloutDeployment(ctx, pod.Namespace, pod.Name))
	for _, action := range clientset.Actions() {
		require.NotEqual(t, "patch", action.GetVerb())
	}
}

func TestCertManagerRolloutDeploymentErrors(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*corev1.Pod, *appsv1.ReplicaSet, *appsv1.Deployment)
		message   string
	}{
		{
			name: "pod is not owned by ReplicaSet",
			configure: func(pod *corev1.Pod, _ *appsv1.ReplicaSet, _ *appsv1.Deployment) {
				pod.OwnerReferences = nil
			},
			message: "not owned by a ReplicaSet",
		},
		{
			name: "ReplicaSet UID changed",
			configure: func(_ *corev1.Pod, rs *appsv1.ReplicaSet, _ *appsv1.Deployment) {
				rs.UID = "different"
			},
			message: "different UID",
		},
		{
			name: "ReplicaSet is not owned by Deployment",
			configure: func(_ *corev1.Pod, rs *appsv1.ReplicaSet, _ *appsv1.Deployment) {
				rs.OwnerReferences = nil
			},
			message: "not owned by a Deployment",
		},
		{
			name: "Deployment UID changed",
			configure: func(_ *corev1.Pod, _ *appsv1.ReplicaSet, deployment *appsv1.Deployment) {
				deployment.UID = "different"
			},
			message: "different UID",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pod, rs, deployment := certManagerRolloutObjects()
			test.configure(pod, rs, deployment)
			rollout := certManagerRollout{clientset: fake.NewClientset(pod, rs, deployment)}
			require.ErrorContains(t, rollout.rolloutDeployment(context.Background(), pod.Namespace, pod.Name), test.message)
		})
	}
}

func TestCertManagerRolloutPatchFailure(t *testing.T) {
	pod, rs, deployment := certManagerRolloutObjects()
	clientset := fake.NewClientset(pod, rs, deployment)
	clientset.PrependReactor("patch", "deployments", func(action kubeTesting.Action) (bool, runtime.Object, error) {
		var patch struct {
			Metadata struct {
				ResourceVersion string `json:"resourceVersion"`
			} `json:"metadata"`
		}
		require.NoError(t, json.Unmarshal(action.(kubeTesting.PatchAction).GetPatch(), &patch))
		require.Equal(t, deployment.ResourceVersion, patch.Metadata.ResourceVersion)
		return true, nil, errors.New("patch denied")
	})
	rollout := certManagerRollout{clientset: clientset}
	require.ErrorContains(t, rollout.rolloutDeployment(context.Background(), pod.Namespace, pod.Name), "patch denied")
}

func TestCertManagerRolloutLookupFailures(t *testing.T) {
	for _, test := range []struct {
		resource string
		message  string
	}{
		{resource: "pods", message: "failed to get operator pod"},
		{resource: "replicasets", message: "failed to get operator ReplicaSet"},
		{resource: "deployments", message: "failed to get operator Deployment"},
	} {
		t.Run(test.resource, func(t *testing.T) {
			pod, rs, deployment := certManagerRolloutObjects()
			clientset := fake.NewClientset(pod, rs, deployment)
			clientset.PrependReactor("get", test.resource, func(kubeTesting.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("API unavailable")
			})
			rollout := certManagerRollout{clientset: clientset}
			err := rollout.rolloutDeployment(context.Background(), pod.Namespace, pod.Name)
			require.ErrorContains(t, err, test.message)
			require.ErrorContains(t, err, "API unavailable")
			for _, action := range clientset.Actions() {
				require.NotEqual(t, "patch", action.GetVerb())
			}
		})
	}
}
