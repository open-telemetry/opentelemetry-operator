// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package target

import (
	"strconv"
	"strings"
	"sync"

	"github.com/cespare/xxhash/v2"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
)

// seps is the separator used between label name/value pairs in hash computation.
// This matches Prometheus's label hashing approach.
var seps = []byte{'\xff'}

// hasherPool is a pool of xxhash digesters for efficient hash computation.
var hasherPool = sync.Pool{
	New: func() any {
		return xxhash.New()
	},
}

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

type ItemHash uint64

func (h ItemHash) String() string {
	return strconv.FormatUint(uint64(h), 10)
}

// Item represents a target to be scraped.
type Item struct {
	JobName       string
	TargetURL     string
	Labels        labels.Labels
	CollectorName string
	hash          ItemHash
}

func (t *Item) Hash() ItemHash {
	return t.hash
}

// HashLabels computes the item hash for a fully materialized label set and job name.
// It delegates to HashFromBuilder so the result is identical to the hash computed while
// relabeling targets during discovery. Callers that already hold a labels.Builder (e.g. the
// discoverer's hot path) should use HashFromBuilder directly to avoid allocating a builder.
func HashLabels(ls labels.Labels, jobName string) ItemHash {
	return HashFromBuilder(labels.NewBuilder(ls), jobName)
}

// HashFromBuilder computes a hash from a labels.Builder, skipping meta labels.
// Meta labels are skipped because Prometheus discards them after relabeling, so two targets
// that differ only in meta labels are the same scrape target and must hash identically.
func HashFromBuilder(builder *labels.Builder, jobName string) ItemHash {
	hash := hasherPool.Get().(*xxhash.Digest)
	hash.Reset()
	builder.Range(func(l labels.Label) {
		// Skip meta labels - they are discarded after relabeling in Prometheus.
		// For details, see https://github.com/prometheus/prometheus/blob/e6cfa720fbe6280153fab13090a483dbd40bece3/scrape/target.go#L534
		if strings.HasPrefix(l.Name, model.MetaLabelPrefix) {
			return
		}
		_, _ = hash.WriteString(l.Name)
		_, _ = hash.Write(seps)
		_, _ = hash.WriteString(l.Value)
		_, _ = hash.Write(seps)
	})
	_, _ = hash.WriteString(jobName)
	result := hash.Sum64()
	hasherPool.Put(hash)
	return ItemHash(result)
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

// GetEndpointSliceName returns the name of the EndpointSlice that the target is part of.
// If the target is not part of an EndpointSlice, it returns an empty string.
func (t *Item) GetEndpointSliceName() string {
	return t.Labels.Get(endpointSliceName)
}

// NewItem Creates a new target item.
// The hash must be computed by the caller (see HashFromBuilder/HashLabels); it identifies the
// target for allocation and deduplication.
// INVARIANTS:
// * Item fields must not be modified after creation.
func NewItem(jobName, targetURL string, itemLabels labels.Labels, collectorName string, hash ItemHash) *Item {
	return &Item{
		JobName:       jobName,
		TargetURL:     targetURL,
		Labels:        itemLabels,
		CollectorName: collectorName,
		hash:          hash,
	}
}
