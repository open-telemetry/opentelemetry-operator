// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
