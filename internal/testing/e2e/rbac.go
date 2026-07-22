// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package e2e

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// BindTargetAllocatorClusterRole applies the project's shipped ClusterRole
// (config/target-allocator/clusterrole.yaml — core target discovery plus the
// Prometheus CRDs) and binds it to the named ServiceAccount in ns. The ClusterRole is
// cluster-scoped and shared (server-side-applied); the per-test ClusterRoleBinding is
// removed on cleanup. It is reused for both the target allocator and an
// operator-managed oracle Prometheus, which needs the same core discovery access.
//
// The operator deliberately does not create the allocator's RBAC itself — the
// permissions a target allocator needs depend on what the user asks it to discover —
// so an e2e test that runs the allocator must supply them.
func BindTargetAllocatorClusterRole(ctx context.Context, t *testing.T, cfg *envconf.Config, ns, saName string) {
	t.Helper()
	c := CRClient(t, cfg)

	clusterRole, err := os.Open(filepath.Join(RepoRoot(t), "config", "target-allocator", "clusterrole.yaml"))
	require.NoError(t, err, "open clusterrole.yaml")
	defer clusterRole.Close()
	applyManifests(ctx, t, c, clusterRole, "")

	binding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: ns + "-" + saName},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     "target-allocator",
		},
		Subjects: []rbacv1.Subject{{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      saName,
			Namespace: ns,
		}},
	}
	require.NoError(t, c.Create(ctx, binding), "create clusterrolebinding %s", binding.Name)
	t.Cleanup(func() {
		err := c.Delete(context.WithoutCancel(ctx), binding)
		if !apierrors.IsNotFound(err) {
			assert.NoError(t, err, "delete clusterrolebinding %s", binding.Name)
		}
	})
}
