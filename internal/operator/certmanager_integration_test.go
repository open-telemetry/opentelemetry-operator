// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	authorizationv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	kubeTesting "k8s.io/client-go/testing"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/autodetectutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/rbac"
)

type testCertManagerRollout struct {
	rollout   *certManagerRollout
	namespace string
	podName   string
}

func (r testCertManagerRollout) requestRollout(ctx context.Context) error {
	return r.rollout.rolloutDeployment(ctx, r.namespace, r.podName)
}

func TestCertManagerAvailabilityRolloutIntegration(t *testing.T) {
	pod, rs, deployment := certManagerRolloutObjects()
	t.Setenv(autodetectutils.NAMESPACE_ENV_VAR, pod.Namespace)
	t.Setenv(autodetectutils.SA_ENV_VAR, "operator")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(metav1.APIGroupList{Groups: []metav1.APIGroup{{Name: "cert-manager.io"}}})
	}))
	defer server.Close()

	clientset := fake.NewClientset(pod, rs, deployment)
	allowed := false
	clientset.PrependReactor("create", "subjectaccessreviews", func(action kubeTesting.Action) (bool, runtime.Object, error) {
		review := action.(kubeTesting.CreateAction).GetObject().(*authorizationv1.SubjectAccessReview).DeepCopy()
		review.Status.Allowed = allowed
		return true, review, nil
	})
	detector, err := autodetect.New(&rest.Config{Host: server.URL}, rbac.NewReviewer(clientset))
	require.NoError(t, err)
	ctx := context.Background()
	initial, err := detector.CertManagerAvailability(ctx)
	require.ErrorIs(t, err, certmanager.ErrMissingPermissions)

	watcher := certManagerAvailabilityWatcher{
		checker:             detector,
		initialAvailability: initial,
		rollout:             testCertManagerRollout{rollout: &certManagerRollout{clientset: clientset}, namespace: pod.Namespace, podName: pod.Name},
	}
	allowed = true
	watcher.check(ctx)
	updated, err := clientset.AppsV1().Deployments(pod.Namespace).Get(ctx, deployment.Name, metav1.GetOptions{})
	require.NoError(t, err)
	firstRollout := updated.Spec.Template.Annotations[certManagerRolloutAnnotation]
	require.NotEmpty(t, firstRollout)

	watcher.check(ctx)
	updated, err = clientset.AppsV1().Deployments(pod.Namespace).Get(ctx, deployment.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, firstRollout, updated.Spec.Template.Annotations[certManagerRolloutAnnotation])

	rs.Spec.Template = updated.Spec.Template
	_, err = clientset.AppsV1().ReplicaSets(pod.Namespace).Update(ctx, rs, metav1.UpdateOptions{})
	require.NoError(t, err)
	watcher.initialAvailability = certmanager.Available
	allowed = false
	watcher.check(ctx)
	updated, err = clientset.AppsV1().Deployments(pod.Namespace).Get(ctx, deployment.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.NotEqual(t, firstRollout, updated.Spec.Template.Annotations[certManagerRolloutAnnotation])

	clientset.PrependReactor("create", "subjectaccessreviews", func(kubeTesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewTimeoutError("unavailable", 1)
	})
	clientset.ClearActions()
	watcher.check(ctx)
	for _, action := range clientset.Actions() {
		require.NotEqual(t, "patch", action.GetVerb())
	}
}
