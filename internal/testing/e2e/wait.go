// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sobj "sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// WaitForStatefulSet blocks until the named StatefulSet reports >= replicas ready.
func WaitForStatefulSet(ctx context.Context, t *testing.T, cfg *envconf.Config, ns, name string, replicas int32, timeout time.Duration) {
	t.Helper()
	ss := &appsv1.StatefulSet{}
	ss.SetName(name)
	ss.SetNamespace(ns)
	err := wait.For(
		conditions.New(cfg.Client().Resources()).ResourceMatch(ss, func(obj k8sobj.Object) bool {
			return obj.(*appsv1.StatefulSet).Status.ReadyReplicas >= replicas
		}),
		wait.WithContext(ctx),
		wait.WithTimeout(timeout),
		wait.WithInterval(2*time.Second),
	)
	require.NoError(t, err, "statefulset %s/%s not ready", ns, name)
}

// WaitForDeployment blocks until the named Deployment reports Available. The
// Deployment not existing yet is tolerated (unlike the upstream
// DeploymentConditionMatch condition, which treats it as terminal): callers
// legitimately wait for Deployments another controller is about to create.
func WaitForDeployment(ctx context.Context, t *testing.T, cfg *envconf.Config, ns, name string, timeout time.Duration) {
	t.Helper()
	dep := &appsv1.Deployment{}
	dep.SetName(name)
	dep.SetNamespace(ns)
	err := wait.For(
		conditions.New(cfg.Client().Resources()).ResourceMatch(dep, func(obj k8sobj.Object) bool {
			for _, cond := range obj.(*appsv1.Deployment).Status.Conditions {
				if cond.Type == appsv1.DeploymentAvailable && cond.Status == corev1.ConditionTrue {
					return true
				}
			}
			return false
		}),
		wait.WithContext(ctx),
		wait.WithTimeout(timeout),
		wait.WithInterval(2*time.Second),
	)
	require.NoError(t, err, "deployment %s/%s not available", ns, name)
}
