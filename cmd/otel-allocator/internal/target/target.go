// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package target

import (
	"encoding/binary"
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

type ItemOption func(*Item)

// WithHash sets a precomputed hash on the item.
// Use this when the hash has been computed during relabeling to avoid recomputation.
func WithHash(hash ItemHash) ItemOption {
	return func(i *Item) {
		i.hash = hash
	}
}

func (t *Item) Hash() ItemHash {
	if t.hash == 0 {
		t.hash = ItemHash(LabelsHashWithJobName(t.Labels, t.JobName))
	}
	return t.hash
}

// HashFromBuilder computes a hash from a labels.Builder, skipping meta labels.
// This is used during relabeling to compute the hash efficiently without materializing
// the filtered labels.
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
// INVARIANTS:
// * Item fields must not be modified after creation.
func NewItem(jobName string, targetURL string, itemLabels labels.Labels, collectorName string, opts ...ItemOption) *Item {
	item := &Item{
		JobName:       jobName,
		TargetURL:     targetURL,
		Labels:        itemLabels,
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
	labelsHash := ls.Hash()
	var labelsHashBytes [8]byte
	binary.LittleEndian.PutUint64(labelsHashBytes[:], labelsHash)
	hash := hasherPool.Get().(*xxhash.Digest)
	hash.Reset()
	_, _ = hash.Write(labelsHashBytes[:]) // nolint: errcheck // xxhash.Write can't fail
	_, _ = hash.WriteString(jobName)      // nolint: errcheck // xxhash.Write can't fail
	result := hash.Sum64()
	hasherPool.Put(hash)
	return result
}
