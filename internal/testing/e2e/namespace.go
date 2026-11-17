// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8swait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// nsContextKey is the context key under which SetupNamespace stores the namespace.
type nsContextKey struct{}

// SetupNamespace creates a namespace named after the running test (see
// NamespaceFromT), registers its deletion as test cleanup, and returns a context
// carrying the namespace name — retrieve it with Namespace. Every test gets its own
// namespace, which is what lets the manifests use fixed object names.
func SetupNamespace(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
	t.Helper()
	ns := NamespaceFromT(t)
	CreateNamespace(ctx, t, cfg, ns)
	t.Cleanup(func() {
		// The feature's context is already cancelled by the time cleanup runs.
		DeleteNamespace(context.WithoutCancel(ctx), t, cfg, ns)
	})
	return context.WithValue(ctx, nsContextKey{}, ns)
}

// Namespace returns the namespace SetupNamespace stored in ctx, failing t when the
// context does not carry one (i.e. the feature's Setup did not call SetupNamespace).
func Namespace(t *testing.T, ctx context.Context) string {
	t.Helper()
	ns, ok := ctx.Value(nsContextKey{}).(string)
	require.True(t, ok, "no namespace in context: call e2e.SetupNamespace in the feature's Setup")
	return ns
}

// CreateNamespace creates ns.
func CreateNamespace(ctx context.Context, t *testing.T, cfg *envconf.Config, ns string) {
	t.Helper()
	obj := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}
	require.NoError(t, CRClient(t, cfg).Create(ctx, obj), "create namespace %s", ns)
}

// The OpenShift namespace annotations that the restricted SCC takes the pod user and group IDs from.
const (
	openShiftUIDRangeAnnotation           = "openshift.io/sa.scc.uid-range"
	openShiftSupplementalGroupsAnnotation = "openshift.io/sa.scc.supplemental-groups"
	openShiftMCSAnnotation                = "openshift.io/sa.scc.mcs"
)

// AllowSampleAppIDsOnOpenShift lets the pods of ns run with the IDs the sample apps
// set (runAsUser 1000, runAsGroup and fsGroup 3000) on OpenShift, whose restricted
// SCC rejects IDs outside the namespace's ranges. Other clusters are left as they are.
func AllowSampleAppIDsOnOpenShift(ctx context.Context, t *testing.T, cfg *envconf.Config, ns string) {
	t.Helper()
	require.NoError(t, allowSampleAppIDsOnOpenShift(ctx, ClientSet(t, cfg), ns, time.Minute),
		"allow the sample app IDs in namespace %s", ns)
}

func allowSampleAppIDsOnOpenShift(ctx context.Context, cs kubernetes.Interface, ns string, timeout time.Duration) error {
	if _, err := cs.Discovery().ServerResourcesForGroupVersion("security.openshift.io/v1"); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("discover the OpenShift security API: %w", err)
	}
	// OpenShift only assigns the ranges and the MCS labels to a namespace that has no
	// UID range yet, so the ranges are replaced after it has annotated the namespace,
	// as tests/step-templates/openshift-namespace-annotations.yaml does for chainsaw.
	var getErr error
	err := k8swait.PollUntilContextTimeout(ctx, time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		obj, err := cs.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
		if err != nil {
			getErr = err
			return false, nil
		}
		_, ok := obj.Annotations[openShiftMCSAnnotation]
		return ok, nil
	})
	if err != nil {
		return fmt.Errorf("wait for the %s annotation: %w", openShiftMCSAnnotation, errors.Join(err, getErr))
	}
	patch, err := json.Marshal(map[string]any{"metadata": map[string]any{"annotations": map[string]string{
		openShiftUIDRangeAnnotation:           "1000/1000",
		openShiftSupplementalGroupsAnnotation: "3000/3000",
	}}})
	if err != nil {
		return err
	}
	_, err = cs.CoreV1().Namespaces().Patch(ctx, ns, types.MergePatchType, patch, metav1.PatchOptions{FieldManager: fieldManager})
	return err
}

// DeleteNamespace deletes ns (ignoring not-found), used for test cleanup.
func DeleteNamespace(ctx context.Context, t *testing.T, cfg *envconf.Config, ns string) {
	t.Helper()
	obj := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}
	err := CRClient(t, cfg).Delete(ctx, obj)
	if !apierrors.IsNotFound(err) {
		require.NoError(t, err, "delete namespace %s", ns)
	}
}
