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
	"sync"
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

const (
	// defaultPromTimeout bounds how long the retrying query helpers wait for a check
	// to pass — long enough for a freshly deployed pipeline to scrape and export.
	defaultPromTimeout = 2 * time.Minute
	// defaultPromInterval is the delay between attempts of the retrying query helpers.
	defaultPromInterval = 2 * time.Second
)

// Prom is a handle to a Prometheus backend reachable through the API server's
// service proxy (which works for headless Services, so no port-forward is needed).
// It carries everything that stays constant for a test — the environment plus the
// namespace, Service and port of the backend — so assertions only name the query.
type Prom struct {
	cfg       *envconf.Config
	namespace string
	service   string
	port      int

	once  sync.Once
	cs    *kubernetes.Clientset
	csErr error
}

// NewProm returns a handle to the Prometheus served by the named Service and port in
// namespace. Build it once the namespace is known (see Namespace) and reuse it for
// every assertion against that backend.
func NewProm(cfg *envconf.Config, namespace, service string, port int) *Prom {
	return &Prom{cfg: cfg, namespace: namespace, service: service, port: port}
}

// clientSet lazily builds (and caches) the clientset used for the service proxy. It
// returns the error instead of failing the test because it runs inside retry loops.
func (p *Prom) clientSet() (*kubernetes.Clientset, error) {
	p.once.Do(func() { p.cs, p.csErr = clientSet(p.cfg) })
	return p.cs, p.csErr
}

// Query runs a single instant PromQL query and decodes the result vector. It does not
// retry — use Eventually for assertions that have to wait for data to arrive.
func (p *Prom) Query(ctx context.Context, t *testing.T, query string) (model.Vector, error) {
	t.Helper()
	cs, err := p.clientSet()
	if err != nil {
		return nil, fmt.Errorf("create clientset: %w", err)
	}
	raw, err := cs.CoreV1().Services(p.namespace).
		ProxyGet("http", p.service, strconv.Itoa(p.port), "/api/v1/query", map[string]string{"query": query}).
		DoRaw(ctx)
	if err != nil {
		return nil, err
	}
	return parsePromVector(raw)
}

// Eventually runs an instant query and retries until check passes. The check is plain
// Go over a model.Vector — assert on values and labels of the queried series. On
// timeout the test fails with the last query or check error.
func (p *Prom) Eventually(ctx context.Context, t *testing.T, query string, check func(model.Vector) error) {
	t.Helper()
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		vec, err := p.Query(ctx, t, query)
		if err != nil {
			c.Errorf("promql %s: %v", query, err)
			return
		}
		if err := check(vec); err != nil {
			c.Errorf("promql %s: %v", query, err)
		}
	}, defaultPromTimeout, defaultPromInterval)
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
func (p *Prom) SameLabelsAcross(ctx context.Context, t *testing.T, d Differential) {
	t.Helper()
	desc := fmt.Sprintf("differential %q across %q", d.Query, d.PartitionLabel)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		vec, err := p.Query(ctx, t, d.Query)
		if err != nil {
			c.Errorf("%s: %v", desc, err)
			return
		}
		if err := samePartitionLabels(vec, d); err != nil {
			c.Errorf("%s: %v", desc, err)
		}
	}, defaultPromTimeout, defaultPromInterval)
}

// samePartitionLabels is the pure check behind SameLabelsAcross: it partitions vec by
// d.PartitionLabel and returns nil when at least d.WantPartitions partitions are
// present and all of them carry the same set of label-sets.
func samePartitionLabels(vec model.Vector, d Differential) error {
	drop := map[model.LabelName]bool{model.MetricNameLabel: true, model.LabelName(d.PartitionLabel): true}
	for _, l := range d.Ignore {
		drop[model.LabelName(l)] = true
	}

	parts := map[string]map[string]bool{} // partition value -> set of canonical label-sets
	for _, s := range vec {
		pv := string(s.Metric[model.LabelName(d.PartitionLabel)])
		if parts[pv] == nil {
			parts[pv] = map[string]bool{}
		}
		parts[pv][canonicalLabels(s.Metric, drop)] = true
	}
	if len(parts) < d.WantPartitions {
		return fmt.Errorf("waiting for %d partitions of %q, have %d: %v", d.WantPartitions, d.PartitionLabel, len(parts), slices.Sorted(maps.Keys(parts)))
	}
	// Compare against the lexically first partition so the message is deterministic.
	names := slices.Sorted(maps.Keys(parts))
	refName := names[0]
	for _, name := range names[1:] {
		if diff := diffSets(parts[refName], parts[name]); diff != "" {
			return fmt.Errorf("label sets differ between %s=%q (A) and %s=%q (B):\n%s", d.PartitionLabel, refName, d.PartitionLabel, name, diff)
		}
	}
	return nil
}

// canonicalLabels renders a sample's labels, minus the dropped ones, as a single
// deterministic string (`{name="value", ...}`, sorted by label name) so that label
// sets can be used as map keys and compared for set membership.
func canonicalLabels(metric model.Metric, drop map[model.LabelName]bool) string {
	ls := make(model.LabelSet, len(metric))
	for k, v := range metric {
		if drop[k] {
			continue
		}
		ls[k] = v
	}
	return ls.String()
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
		fmt.Fprintf(&sb, "  only in A: %s\n", s)
	}
	for _, s := range onlyB {
		fmt.Fprintf(&sb, "  only in B: %s\n", s)
	}
	return sb.String()
}

// parsePromVector decodes a Prometheus instant-query JSON response into a
// model.Vector, leaving the sample decoding (label sets, timestamps, native
// histograms) to prometheus/common's own unmarshaller.
func parsePromVector(raw []byte) (model.Vector, error) {
	var resp struct {
		Status string `json:"status"`
		Error  string `json:"error"`
		Data   struct {
			ResultType string          `json:"resultType"`
			Result     json.RawMessage `json:"result"`
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
	if err := json.Unmarshal(resp.Data.Result, &vec); err != nil {
		return nil, fmt.Errorf("decode prometheus vector: %w", err)
	}
	return vec, nil
}
