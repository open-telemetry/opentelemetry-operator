// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package target

import (
	"strconv"

	"github.com/prometheus/prometheus/model/labels"
)

// nodeLabels are labels that are used to identify the node on which the given
// target is residing. To learn more about these labels, please refer to:
// https://prometheus.io/docs/prometheus/latest/configuration/configuration/#kubernetes_sd_config
var (
	nodeLabels = []string{
		"__meta_kubernetes_pod_node_name",
		"__meta_kubernetes_node_name",
		"__meta_kubernetes_endpoint_node_name",
	}
	endpointSliceTargetKindLabel = "__meta_kubernetes_endpointslice_address_target_kind"
	endpointSliceTargetNameLabel = "__meta_kubernetes_endpointslice_address_target_name"
	relevantLabelNames           = append(nodeLabels, endpointSliceTargetKindLabel, endpointSliceTargetNameLabel)
)

type Item struct {
	JobName       string
	TargetURL     string
	Labels        labels.Labels
	CollectorName string
	hash          string
}

func (t *Item) Hash() string {
	return t.hash
}

func (t *Item) GetNodeName() string {
	relevantLabels := t.Labels.MatchLabels(true, relevantLabelNames...)
	for _, label := range nodeLabels {
		if val := relevantLabels.Get(label); val != "" {
			return val
		}
	}

	if val := relevantLabels.Get(endpointSliceTargetKindLabel); val != "Node" {
		return ""
	}

	return relevantLabels.Get(endpointSliceTargetNameLabel)
}

// NewItem Creates a new target item.
// INVARIANTS:
// * Item fields must not be modified after creation.
// * Item should only be made via its constructor, never directly.
func NewItem(jobName string, targetURL string, labels labels.Labels, collectorName string) *Item {
	return &Item{
		JobName:       jobName,
		hash:          jobName + targetURL + strconv.FormatUint(labels.Hash(), 10),
		TargetURL:     targetURL,
		Labels:        labels,
		CollectorName: collectorName,
	}
}
