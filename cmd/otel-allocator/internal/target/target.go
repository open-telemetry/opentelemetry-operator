// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package target

import (
	"slices"
	"strings"

	"github.com/prometheus/common/model"
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

// metricResourceLabels are added to the resource by default in OTel.
// The original values should be retrieved for further processing.
// For details, see https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/receiver/prometheusreceiver/internal/prom_to_otlp.go#L89.
var (
	metricResourceLabels = append(nodeLabels, []string{
		"__meta_kubernetes_pod_name",
		"__meta_kubernetes_pod_uid",
		"__meta_kubernetes_pod_container_name",
		"__meta_kubernetes_namespace",
	}...)
)

type ItemHash uint64

type Item struct {
	JobName        string
	TargetURL      string
	Labels         labels.Labels
	ReservedLabels labels.Labels
	CollectorName  string
	hash           ItemHash
}

type ItemOption func(*Item)

func WithReservedLabelMatching(labels labels.Labels) ItemOption {
	return func(i *Item) {
		i.ReservedLabels = labels.MatchLabels(true, metricResourceLabels...)
	}
}

// WithFilterMetaLabels is used to remove labels with the MetaLabelPrefix to prevent them from being used in hash calculation.
// In Prometheus, labels with the MetaLabelPrefix are discarded after relabeling and are not used in hash calculation.
// For details, see https://github.com/prometheus/prometheus/blob/e6cfa720fbe6280153fab13090a483dbd40bece3/scrape/target.go#L534.
func WithFilterMetaLabels() ItemOption {
	return func(i *Item) {
		writeIndex := 0
		for _, l := range i.Labels {
			if !strings.HasPrefix(l.Name, model.MetaLabelPrefix) {
				i.Labels[writeIndex] = l
				writeIndex++
			}
		}
		i.Labels = i.Labels[:writeIndex]
		i.Labels = slices.Clip(i.Labels)
	}
}

func (t *Item) Hash() ItemHash {
	if t.hash == 0 {
		t.hash = ItemHash(t.Labels.Hash())
	}
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

func (t *Item) AllLabels() labels.Labels {
	allLabels := make(labels.Labels, 0, len(t.Labels)+len(t.ReservedLabels))
	allLabels = append(allLabels, t.ReservedLabels...)
	return append(allLabels, t.Labels.MatchLabels(false, relevantLabelNames...)...)
}

// NewItem Creates a new target item.
// INVARIANTS:
// * Item fields must not be modified after creation.
func NewItem(jobName string, targetURL string, labels labels.Labels, collectorName string, opts ...ItemOption) *Item {
	item := &Item{
		JobName:       jobName,
		TargetURL:     targetURL,
		Labels:        labels,
		CollectorName: collectorName,
	}
	for _, opt := range opts {
		opt(item)
	}
	return item
}
