// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// PromTarget identifies a Prometheus backend by its in-namespace Service and port.
type PromTarget struct {
	Service string
	Port    int
}

// queryProm runs an instant PromQL query against a Prometheus Service via the API
// server's service proxy (works for headless Services, so no port-forward is needed).
func queryProm(ctx context.Context, cs *kubernetes.Clientset, ns string, target PromTarget, query string) (model.Vector, error) {
	raw, err := cs.CoreV1().Services(ns).
		ProxyGet("http", target.Service, strconv.Itoa(target.Port), "/api/v1/query", map[string]string{"query": query}).
		DoRaw(ctx)
	if err != nil {
		return nil, err
	}
	return parsePromVector(raw)
}

// parsePromVector decodes a Prometheus instant-query JSON response into a model.Vector.
func parsePromVector(raw []byte) (model.Vector, error) {
	var resp struct {
		Status string `json:"status"`
		Error  string `json:"error"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Value  []any             `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("decode prometheus response: %w", err)
	}
	if resp.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed: %s", resp.Error)
	}
	if resp.Data.ResultType != "vector" {
		return nil, fmt.Errorf("query returned %q, want a vector", resp.Data.ResultType)
	}
	var vec model.Vector
	for _, r := range resp.Data.Result {
		metric := model.Metric{}
		for k, v := range r.Metric {
			metric[model.LabelName(k)] = model.LabelValue(v)
		}
		var val model.SampleValue
		if len(r.Value) == 2 {
			s, _ := r.Value[1].(string)
			f, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return nil, fmt.Errorf("parse sample value %q: %w", s, err)
			}
			val = model.SampleValue(f)
		}
		vec = append(vec, &model.Sample{Metric: metric, Value: val})
	}
	return vec, nil
}

// eventuallyInterval is the delay between eventually's attempts.
const eventuallyInterval = 2 * time.Second

// eventually retries fn until it returns nil, ctx is done, or attempts are exhausted.
func eventually(ctx context.Context, t *testing.T, desc string, attempts int, fn func() error) {
	t.Helper()
	var err error
	for range attempts {
		if err = fn(); err == nil {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("%s: %v (last error: %v)", desc, ctx.Err(), err)
		case <-time.After(eventuallyInterval):
		}
	}
	t.Fatalf("%s: gave up after %d attempts: %v", desc, attempts, err)
}

// EventuallyPromQL runs an instant query against a Prometheus Service and retries until
// check passes. The check is plain Go over a model.Vector — assert on values and labels
// of the queried series.
func EventuallyPromQL(ctx context.Context, t *testing.T, cfg *envconf.Config, ns string, target PromTarget, query string, check func(model.Vector) error) {
	t.Helper()
	cs := ClientSet(t, cfg)
	eventually(ctx, t, "promql: "+query, 60, func() error {
		vec, err := queryProm(ctx, cs, ns, target, query)
		if err != nil {
			return err
		}
		return check(vec)
	})
}

// Differential describes a SameLabelsAcross comparison: partition the result of
// Query by the value of PartitionLabel, require at least WantPartitions distinct
// partitions, and ignore the labels in Ignore (they legitimately differ between
// pipelines).
type Differential struct {
	Query          string
	PartitionLabel string
	WantPartitions int
	Ignore         []string
}

// SameLabelsAcross is a differential within one Prometheus: it queries a single
// backend, partitions the result series by the value of d.PartitionLabel, and asserts
// every partition exposes the same set of label-sets (after dropping the partition
// label and any ignored labels). Use it to compare two pipelines that write to one
// Prometheus distinguished by a label — e.g. the target allocator pipeline
// (pipeline=ta) versus prometheus-operator scraping the same target natively
// (pipeline=oracle). Identical sets mean the allocator labeled the target exactly as
// prometheus-operator does. It retries until at least d.WantPartitions distinct
// partitions are present and they agree.
//
// Query `up` to compare pure target identity.
func SameLabelsAcross(ctx context.Context, t *testing.T, cfg *envconf.Config, ns string, target PromTarget, d Differential) {
	t.Helper()
	cs := ClientSet(t, cfg)
	drop := map[string]bool{"__name__": true, d.PartitionLabel: true}
	for _, l := range d.Ignore {
		drop[l] = true
	}

	desc := fmt.Sprintf("differential %q across %q", d.Query, d.PartitionLabel)
	eventually(ctx, t, desc, 60, func() error {
		vec, err := queryProm(ctx, cs, ns, target, d.Query)
		if err != nil {
			return err
		}
		parts := map[string]map[string]bool{} // partition value -> set of canonical label-sets
		for _, s := range vec {
			pv := string(s.Metric[model.LabelName(d.PartitionLabel)])
			labels := map[string]string{}
			for k, v := range s.Metric {
				if drop[string(k)] {
					continue
				}
				labels[string(k)] = string(v)
			}
			if parts[pv] == nil {
				parts[pv] = map[string]bool{}
			}
			parts[pv][canonicalLabels(labels)] = true
		}
		if len(parts) < d.WantPartitions {
			return fmt.Errorf("waiting for %d partitions of %q, have %d: %v", d.WantPartitions, d.PartitionLabel, len(parts), slices.Sorted(maps.Keys(parts)))
		}
		var refName string
		var ref map[string]bool
		for name, set := range parts {
			if ref == nil {
				refName, ref = name, set
				continue
			}
			if diff := diffSets(ref, set); diff != "" {
				return fmt.Errorf("label sets differ between %s=%q (A) and %s=%q (B):\n%s", d.PartitionLabel, refName, d.PartitionLabel, name, diff)
			}
		}
		return nil
	})
}

// canonicalLabels renders a label set as a single deterministic string (labels sorted
// by name, each rendered as name="value") so that label sets can be used as map keys
// and compared for set membership.
func canonicalLabels(m map[string]string) string {
	var b strings.Builder
	for i, k := range slices.Sorted(maps.Keys(m)) {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, "%s=%q", k, m[k])
	}
	return b.String()
}

// diffSets returns a human-readable description of the symmetric difference between
// two sets of canonical label-sets — the members only in a ("only in A") and only in
// b ("only in B"). It returns the empty string when the sets are equal.
func diffSets(a, b map[string]bool) string {
	var onlyA, onlyB []string
	for k := range a {
		if !b[k] {
			onlyA = append(onlyA, k)
		}
	}
	for k := range b {
		if !a[k] {
			onlyB = append(onlyB, k)
		}
	}
	if len(onlyA) == 0 && len(onlyB) == 0 {
		return ""
	}
	slices.Sort(onlyA)
	slices.Sort(onlyB)
	var sb strings.Builder
	for _, s := range onlyA {
		fmt.Fprintf(&sb, "  only in A: {%s}\n", s)
	}
	for _, s := range onlyB {
		fmt.Fprintf(&sb, "  only in B: {%s}\n", s)
	}
	return sb.String()
}
