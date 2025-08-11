// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package target

import (
	"strconv"

	"slices"
	"strings"

	"github.com/cespare/xxhash/v2"

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
	endpointSliceName            = "__meta_kubernetes_endpointslice_name"
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

func (h ItemHash) String() string {
	return strconv.FormatUint(uint64(h), 10)
}

// Item represents a target to be scraped.
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
		t.hash = ItemHash(LabelsHashWithJobName(t.Labels, t.JobName))
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

// GetEndpointSliceName returns the name of the EndpointSlice that the target is part of.
// If the target is not part of an EndpointSlice, it returns an empty string.
func (t *Item) GetEndpointSliceName() string {
	return t.Labels.Get(endpointSliceName)
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

// LabelsHashWithJobName computes a hash of the labels and the job name.
// Same logic as Prometheus labels.Hash: https://github.com/prometheus/prometheus/blob/8fd46f74aa0155e4d5aa30654f9c02e564e03743/model/labels/labels.go#L72
// but adds in the job name since this is not in the labelset from the discovery manager.
// The scrape manager adds it later. Address is already included in the labels, so it is not needed here.
func LabelsHashWithJobName(ls labels.Labels, jobName string) uint64 {
	var sep byte = '\xff'
	var seps = []byte{sep}

	// Use xxhash.Sum64(b) for fast path as it's faster.
	b := make([]byte, 0, 1024)

	// Differs from Prometheus implementation by adding job name.
	b = append(b, jobName...)
	b = append(b, sep)

	for i, v := range ls {
		if len(b)+len(v.Name)+len(v.Value)+2 >= cap(b) {
			// If labels entry is 1KB+ do not allocate whole entry.
			h := xxhash.New()
			_, _ = h.Write(b)
			for _, v := range ls[i:] {
				_, _ = h.WriteString(v.Name)
				_, _ = h.Write(seps)
				_, _ = h.WriteString(v.Value)
				_, _ = h.Write(seps)
			}
			return h.Sum64()
		}

		b = append(b, v.Name...)
		b = append(b, sep)
		b = append(b, v.Value...)
		b = append(b, sep)
	}

	return xxhash.Sum64(b)
}
