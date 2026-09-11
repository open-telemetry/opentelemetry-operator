// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/rbac"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
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

			deletionTS, err := removeFinalizer(context.Background(), reconciler, params, instance)

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

func TestReconcileReportsParamsAndBuildErrors(t *testing.T) {
	const configYAML = `
receivers:
  otlp:
    protocols:
      grpc: {}
exporters:
  debug: {}
service:
  pipelines:
    metrics:
      receivers: [otlp]
      exporters: [debug]
`
	testCases := []struct {
		name       string
		collector  func() *v1beta1.OpenTelemetryCollector
		wantErrMsg string
	}{
		{
			name: "target allocator referenced by label does not exist",
			collector: func() *v1beta1.OpenTelemetryCollector {
				return &v1beta1.OpenTelemetryCollector{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "default",
						Labels:    map[string]string{constants.LabelTargetAllocator: "missing-ta"},
					},
				}
			},
			wantErrMsg: `"missing-ta" not found`,
		},
		{
			name: "target allocator enabled without prometheus receiver",
			collector: func() *v1beta1.OpenTelemetryCollector {
				var cfg v1beta1.Config
				require.NoError(t, yaml.Unmarshal([]byte(configYAML), &cfg))
				return &v1beta1.OpenTelemetryCollector{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "default",
					},
					Spec: v1beta1.OpenTelemetryCollectorSpec{
						Mode:            v1beta1.ModeStatefulSet,
						TargetAllocator: v1beta1.TargetAllocatorEmbedded{Enabled: true},
						Config:          cfg,
					},
				}
			},
			wantErrMsg: "no prometheus available as part of the configuration",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			instance := tc.collector()
			fakeClient := fake.NewClientBuilder().
				WithScheme(reconcilerTestScheme).
				WithObjects(instance).
				WithStatusSubresource(instance).
				Build()
			recorder := events.NewFakeRecorder(10)
			reconciler := NewReconciler(Params{
				Client:   fakeClient,
				Recorder: recorder,
				Scheme:   reconcilerTestScheme,
				Log:      logr.Discard(),
				Config: config.Config{
					CollectorConfigMapEntry:       "collector.yaml",
					TargetAllocatorConfigMapEntry: "targetallocator.yaml",
				},
			})
			nsn := types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}

			_, err := reconciler.Reconcile(t.Context(), reconcile.Request{NamespacedName: nsn})
			require.ErrorContains(t, err, tc.wantErrMsg)

			select {
			case event := <-recorder.Events:
				assert.Contains(t, event, corev1.EventTypeWarning)
				assert.Contains(t, event, tc.wantErrMsg)
			default:
				t.Errorf("expected a %s event containing %q, got none", corev1.EventTypeWarning, tc.wantErrMsg)
			}

			updated := &v1beta1.OpenTelemetryCollector{}
			require.NoError(t, fakeClient.Get(t.Context(), nsn, updated))
			ready := meta.FindStatusCondition(updated.Status.Conditions, "Ready")
			require.NotNil(t, ready, "expected a Ready condition on the collector status")
			assert.Equal(t, metav1.ConditionFalse, ready.Status)
			assert.Equal(t, "ReconcileError", ready.Reason)
			assert.True(t, strings.Contains(ready.Message, tc.wantErrMsg), "condition message %q should contain %q", ready.Message, tc.wantErrMsg)
		})
	}
}
