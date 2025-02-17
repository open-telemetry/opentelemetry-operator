// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
