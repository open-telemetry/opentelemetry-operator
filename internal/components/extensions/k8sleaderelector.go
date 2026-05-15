// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
)

// generatek8sleaderelectorRbacRules returns the RBAC policy rules required by the
// k8s_leader_elector extension.
//
// The extension uses Kubernetes Lease objects from the coordination.k8s.io
// API group for leader election among multiple collector replicas, enabling
// HA deployments of receivers that should only run on a single instance
// (e.g. k8sclusterreceiver, k8seventsreceiver).
//
// All seven verbs are required:
//   - get/list/watch: observe existing leases and detect leader changes
//   - create: create a new Lease if one doesn't yet exist
//   - update/patch: renew the lease (heartbeat) while holding leadership
//   - delete: release/clean up leases
//
// Ref: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/extension/k8sleaderelector
func generatek8sleaderelectorRbacRules(_ logr.Logger, _ any) ([]rbacv1.PolicyRule, error) {
	return []rbacv1.PolicyRule{
		{
			APIGroups: []string{"coordination.k8s.io"},
			Resources: []string{"leases"},
			Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
		},
	}, nil
}
