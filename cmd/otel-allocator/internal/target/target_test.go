// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package target

import (
	"testing"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
)

func TestItemHash_String(t *testing.T) {
	tests := []struct {
		name string
		h    ItemHash
		want string
	}{
		{
			name: "empty",
			h:    0,
			want: "0",
		},
		{
			name: "non-empty",
			h:    1,
			want: "1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, tt.h.String(), "String()")
		})
	}
}

func TestDuplicatedTargetMetricsWhenDualStackEnabled(t *testing.T) {
	tests := []struct {
		name                string
		tg                  targetgroup.Group
		expectedTargetCount int
	}{
		{
			name: "should ignore duplicate target allocation when dualStack is enabled",
			tg: targetgroup.Group{
				Labels: model.LabelSet{
					"job": "serviceMonitor/opentelemetry-operator-system/kube-state-metrics/0",
				},
				Targets: []model.LabelSet{
					{
						model.AddressLabel:                     "[2600:1234:5678:b307:48::8a7b]:8080",
						"__meta_kubernetes_service_name":       "opentelemetry-kube-stack-kube-state-metrics",
						"__meta_kubernetes_pod_name":           "kube-state-metrics-pod-1",
						"__meta_kubernetes_namespace":          "opentelemetry-operator-system",
						"__meta_kubernetes_endpointslice_name": "opentelemetry-kube-stack-kube-state-metrics-tl2q7",
					},
					{
						model.AddressLabel:                     "10.233.12.197:8080",
						"__meta_kubernetes_service_name":       "opentelemetry-kube-stack-kube-state-metrics",
						"__meta_kubernetes_pod_name":           "kube-state-metrics-pod-1",
						"__meta_kubernetes_namespace":          "opentelemetry-operator-system",
						"__meta_kubernetes_endpointslice_name": "opentelemetry-kube-stack-kube-state-metrics-tzk4w",
					},
				},
			},
			expectedTargetCount: 1,
		},
		{
			name: "should keep targets from different pods",
			tg: targetgroup.Group{
				Labels: model.LabelSet{
					"job": "serviceMonitor/default/my-service/0",
				},
				Targets: []model.LabelSet{
					{
						model.AddressLabel:               "10.233.12.198:8080",
						"__meta_kubernetes_service_name": "my-service",
						"__meta_kubernetes_pod_name":     "my-service-pod-1",
						"__meta_kubernetes_namespace":    "default",
					},
					{
						model.AddressLabel:               "10.233.12.199:8080",
						"__meta_kubernetes_service_name": "my-service",
						"__meta_kubernetes_pod_name":     "my-service-pod-2",
						"__meta_kubernetes_namespace":    "default",
					},
				},
			},
			expectedTargetCount: 2,
		},
		{
			name: "should keep targets without service name",
			tg: targetgroup.Group{
				Labels: model.LabelSet{
					"job": "erviceMonitor/default/my-service/0",
				},
				Targets: []model.LabelSet{
					{
						model.AddressLabel:            "10.233.12.200:8080",
						"__meta_kubernetes_namespace": "default",
					},
					{
						model.AddressLabel:            "10.233.12.201:8080",
						"__meta_kubernetes_namespace": "default",
					},
				},
			},
			expectedTargetCount: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			builder := labels.NewBuilder(labels.Labels{})
			var seenServicePods = make(map[string]bool)
			index := 0
			tg := test.tg
			builder.Reset(labels.EmptyLabels())

			intoTargets := make([]*Item, len(tg.Targets))

			for ln, lv := range tg.Labels {
				builder.Set(string(ln), string(lv))
			}
			groupLabels := builder.Labels()
			for _, t := range tg.Targets {
				builder.Reset(groupLabels)
				for ln, lv := range t {
					builder.Set(string(ln), string(lv))
				}
				item := NewItem(test.name, string(t[model.AddressLabel]), builder.Labels(), "")
				if !item.IsDualStackDuplicate(seenServicePods) {
					intoTargets[index] = item
					index++
					if key := item.GetDualStackKey(); key != "" {
						seenServicePods[key] = true
					}
				}
			}
			intoTargets = intoTargets[:index]
			assert.Equal(t, test.expectedTargetCount, len(intoTargets), "unexpected targets after deduplication")
		})
	}
}
