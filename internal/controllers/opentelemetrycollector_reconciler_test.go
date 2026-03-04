// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/rbac"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
)

var reconcilerTestScheme *runtime.Scheme

func init() {
	reconcilerTestScheme = runtime.NewScheme()
	utilruntime.Must(scheme.AddToScheme(reconcilerTestScheme))
	utilruntime.Must(monitoringv1.AddToScheme(reconcilerTestScheme))
	utilruntime.Must(networkingv1.AddToScheme(reconcilerTestScheme))
	utilruntime.Must(routev1.AddToScheme(reconcilerTestScheme))
	utilruntime.Must(v1alpha1.AddToScheme(reconcilerTestScheme))
	utilruntime.Must(v1beta1.AddToScheme(reconcilerTestScheme))
}

func TestGetCollectorConfigMapsToKeep(t *testing.T) {
	now := time.Now()
	testCases := []struct {
		name           string
		versionsToKeep int
		input          []*corev1.ConfigMap
		output         []*corev1.ConfigMap
	}{
		{
			name:   "no configmaps",
			input:  []*corev1.ConfigMap{},
			output: []*corev1.ConfigMap{},
		},
		{
			name: "one configmap",
			input: []*corev1.ConfigMap{
				{},
			},
			output: []*corev1.ConfigMap{
				{},
			},
		},
		{
			name: "two configmaps, keep one",
			input: []*corev1.ConfigMap{
				{ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: now}}},
				{ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: now.Add(time.Second)}}},
			},
			output: []*corev1.ConfigMap{
				{ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: now.Add(time.Second)}}},
			},
		},
		{
			name:           "three configmaps, keep two",
			versionsToKeep: 2,
			input: []*corev1.ConfigMap{
				{ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: now}}},
				{ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: now.Add(time.Second)}}},
				{ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: now.Add(time.Minute)}}},
			},
			output: []*corev1.ConfigMap{
				{ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: now.Add(time.Minute)}}},
				{ObjectMeta: metav1.ObjectMeta{CreationTimestamp: metav1.Time{Time: now.Add(time.Second)}}},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualOutput := getCollectorConfigMapsToKeep(tc.versionsToKeep, tc.input)
			assert.Equal(t, tc.output, actualOutput)
		})
	}
}

func TestMaybeAddFinalizer(t *testing.T) {
	testCases := []struct {
		name          string
		rbacAvailable rbac.Availability
		hasFinalizer  bool
		expectAdded   bool
	}{
		{
			name:          "adds finalizer when RBAC available and no finalizer exists",
			rbacAvailable: rbac.Available,
			hasFinalizer:  false,
			expectAdded:   true,
		},
		{
			name:          "does not add finalizer when RBAC not available",
			rbacAvailable: rbac.NotAvailable,
			hasFinalizer:  false,
			expectAdded:   false,
		},
		{
			name:          "does not add finalizer when already exists",
			rbacAvailable: rbac.Available,
			hasFinalizer:  true,
			expectAdded:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			instance := &v1beta1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
			}

			if tc.hasFinalizer {
				controllerutil.AddFinalizer(instance, collectorFinalizer)
			}

			params := manifests.Params{
				Config: config.Config{
					CreateRBACPermissions: tc.rbacAvailable,
				},
			}

			result := maybeAddFinalizer(params, instance)
			assert.Equal(t, tc.expectAdded, result)
		})
	}
}

func TestRemoveFinalizer(t *testing.T) {
	now := metav1.Now()

	testCases := []struct {
		name                   string
		rbacAvailable          rbac.Availability
		hasFinalizer           bool
		hasDeletionTimestamp   bool
		expectFinalizerRemoved bool
		expectDeletionTS       bool
		expectError            bool
	}{
		{
			name:                   "removes finalizer when deletion timestamp set and finalizer exists",
			rbacAvailable:          rbac.Available,
			hasFinalizer:           true,
			hasDeletionTimestamp:   true,
			expectFinalizerRemoved: true,
			expectDeletionTS:       true,
		},
		{
			name:                   "removes finalizer when RBAC not available and finalizer exists",
			rbacAvailable:          rbac.NotAvailable,
			hasFinalizer:           true,
			hasDeletionTimestamp:   false,
			expectFinalizerRemoved: true,
			expectDeletionTS:       false,
		},
		{
			name:                   "no-op when RBAC available and no deletion timestamp",
			rbacAvailable:          rbac.Available,
			hasFinalizer:           true,
			hasDeletionTimestamp:   false,
			expectFinalizerRemoved: false,
			expectDeletionTS:       false,
		},
		{
			name:                   "no-op when no finalizer and no deletion timestamp and RBAC available",
			rbacAvailable:          rbac.Available,
			hasFinalizer:           false,
			hasDeletionTimestamp:   false,
			expectFinalizerRemoved: false,
			expectDeletionTS:       false,
		},
		{
			name:                   "no-op when deletion timestamp set but no finalizer",
			rbacAvailable:          rbac.Available,
			hasFinalizer:           false,
			hasDeletionTimestamp:   true,
			expectFinalizerRemoved: false,
			expectDeletionTS:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			instance := &v1beta1.OpenTelemetryCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-collector",
					Namespace: "default",
				},
			}

			if tc.hasFinalizer {
				controllerutil.AddFinalizer(instance, collectorFinalizer)
			}
			if tc.hasDeletionTimestamp {
				instance.DeletionTimestamp = &now
				// The fake client requires at least one finalizer when deletionTimestamp is set.
				// Add a temporary one for object creation, then remove it if the test doesn't want it.
				if !tc.hasFinalizer {
					controllerutil.AddFinalizer(instance, "fake/temp")
				}
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(reconcilerTestScheme).
				WithObjects(instance).
				Build()

			// Remove the temporary finalizer so the test exercises the real condition.
			if tc.hasDeletionTimestamp && !tc.hasFinalizer {
				controllerutil.RemoveFinalizer(instance, "fake/temp")
			}

			reconciler := &OpenTelemetryCollectorReconciler{
				Client: fakeClient,
				log:    logr.Discard(),
				scheme: reconcilerTestScheme,
				config: config.Config{
					CreateRBACPermissions: tc.rbacAvailable,
				},
			}

			params := manifests.Params{
				Config: config.Config{
					CreateRBACPermissions: tc.rbacAvailable,
				},
			}

			deletionTS, err := removeFinalizer(reconciler, context.Background(), params, instance)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.expectDeletionTS {
				assert.NotNil(t, deletionTS)
			} else {
				assert.Nil(t, deletionTS)
			}

			if tc.expectFinalizerRemoved {
				assert.False(t, controllerutil.ContainsFinalizer(instance, collectorFinalizer),
					"expected finalizer to be removed")
			}
		})
	}
}
