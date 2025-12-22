// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/rbac"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
)

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
