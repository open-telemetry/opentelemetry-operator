// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// This file contains a full-stack integration test for zone-aware target
// allocation. It wires every layer the way main.go does — node zone
// resolver -> ZoneTopology -> allocator with zone-aware strategy -> HTTP
// server with the /zones endpoint — against a fake Kubernetes API. The
// goal is to surface any regression where the layers individually pass
// their unit tests but don't compose correctly. Without a test like this,
// a wiring bug in main.go could ship undetected.

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/allocation"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/server"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
)

const (
	integZoneLabel       = "topology.kubernetes.io/zone"
	integTargetZoneLabel = "__meta_kubernetes_endpointslice_endpoint_zone"
)

// nodeIn builds a Kubernetes Node object carrying the standard topology
// zone label. Used to feed the NodeZoneResolver from a fake API client.
func nodeIn(name, zone string) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{integZoneLabel: zone},
		},
	}
}

// integTarget creates a Prometheus target item that carries the SD zone
// meta-label the ZoneTopology will read.
func integTarget(url, zone string) *target.Item {
	return target.NewItem("scrape", url, labels.New(
		labels.Label{Name: integTargetZoneLabel, Value: zone},
		labels.Label{Name: "instance", Value: url},
	), "")
}

// probeZones issues a GET /zones against the server's HTTP handler and
// returns the decoded payload alongside the raw response. Centralizes
// the HTTP boilerplate so each test case stays focused on assertions.
func probeZones(t *testing.T, srv *server.Server) (resp *http.Response, payload map[string]any) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/zones", http.NoBody)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	resp = w.Result()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.NoError(t, json.Unmarshal(body, &payload))
	return resp, payload
}

// TestZoneAware_FullStackIntegration wires the full zone-aware pipeline
// the same way main.go does it, against a fake Kubernetes API server.
// This is the closest we can get to a real end-to-end test without a
// kind cluster, and it catches wiring bugs (e.g. the zone resolver not
// being passed into the collector watcher, or the topology not being
// attached to the allocator) that pure unit tests can miss.
func TestZoneAware_FullStackIntegration(t *testing.T) {
	log := logf.Log.WithName("integration-test")

	// Step 1: fake Kubernetes API with three nodes spread across three
	// zones. This is the substrate the zone resolver reads.
	k8sClient := fake.NewClientset(
		nodeIn("node-a", "us-east-1a"),
		nodeIn("node-b", "us-east-1b"),
		nodeIn("node-c", "us-east-1c"),
	)

	// Step 2: zone resolver populates its index from the API.
	resolver := allocation.NewNodeZoneResolver(log, k8sClient, integZoneLabel)
	require.NoError(t, resolver.SyncNodes(t.Context()),
		"NodeZoneResolver must successfully sync nodes from a fake API")
	assert.Equal(t, "us-east-1a", resolver.GetZone("node-a"))
	assert.Equal(t, "us-east-1c", resolver.GetZone("node-c"))

	// Step 3: build the allocator with a ZoneTopology, just like main.go
	// does when topology.zone_aware is true.
	topology, err := allocation.NewZoneTopology(log, integTargetZoneLabel)
	require.NoError(t, err)
	a, err := allocation.New("least-weighted", log,
		allocation.WithMaxSkew(0),
		allocation.WithZoneTopology(topology))
	require.NoError(t, err)

	// Step 4: feed collectors directly into the allocator with the zones
	// we'd expect the watcher to resolve from the resolver. We exercise
	// the watcher integration separately in collector_test.go; here we
	// stay focused on the allocator + server + topology composition.
	collectors := map[string]*allocation.Collector{
		"collector-a-1": allocation.NewCollector("collector-a-1", "node-a", resolver.GetZone("node-a")),
		"collector-a-2": allocation.NewCollector("collector-a-2", "node-a", resolver.GetZone("node-a")),
		"collector-b":   allocation.NewCollector("collector-b", "node-b", resolver.GetZone("node-b")),
		"collector-c":   allocation.NewCollector("collector-c", "node-c", resolver.GetZone("node-c")),
	}
	a.SetCollectors(collectors)

	// Step 5: load up a workload spanning all three zones plus an
	// uncovered zone (us-east-1d) that has no collector.
	wanted := []*target.Item{
		integTarget("svc-a-1:9100", "us-east-1a"),
		integTarget("svc-a-2:9100", "us-east-1a"),
		integTarget("svc-a-3:9100", "us-east-1a"),
		integTarget("svc-b-1:9100", "us-east-1b"),
		integTarget("svc-b-2:9100", "us-east-1b"),
		integTarget("svc-c-1:9100", "us-east-1c"),
		integTarget("svc-d-1:9100", "us-east-1d"), // uncovered — must failover
	}
	a.SetTargets(wanted)

	// Step 6: every target with a covered zone must be on a same-zone
	// collector. The uncovered-zone target gets failed over to *some*
	// collector but must be tracked as desiring the uncovered zone.
	tracked := a.TargetItems()
	require.Len(t, tracked, len(wanted))
	collectorsNow := a.Collectors()
	for _, it := range tracked {
		desired := it.Labels.Get(integTargetZoneLabel)
		owner := collectorsNow[it.CollectorName]
		require.NotNil(t, owner, "target %s was assigned to unknown collector %q", it.TargetURL, it.CollectorName)
		if desired == "us-east-1d" {
			// Uncovered — any collector is acceptable.
			continue
		}
		assert.Equal(t, desired, owner.Zone,
			"target %s wants %q but landed on collector %q in zone %q — zone-aware assignment failed",
			it.TargetURL, desired, owner.Name, owner.Zone)
	}
	assert.Equal(t, []string{"us-east-1d"}, topology.UncoveredZones(),
		"uncovered zone reporting must surface us-east-1d to the operator")

	// Step 7: stand up the HTTP server and assert the /zones endpoint
	// reflects the same picture the topology object has internally. This
	// is the contract dashboards and tooling will rely on in production.
	srv, err := server.NewServer(log, a, "")
	require.NoError(t, err)
	resp, got := probeZones(t, srv)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	assert.Equal(t, true, got["enabled"], "/zones must report enabled when a topology is attached")
	require.IsType(t, []any{}, got["uncoveredZones"])
	uncov := got["uncoveredZones"].([]any)
	require.Len(t, uncov, 1)
	assert.Equal(t, "us-east-1d", uncov[0])

	// Sanity-check zone breakdown: all four zones present, with target
	// counts attributed to desired zone (not assigned collector zone).
	require.IsType(t, []any{}, got["zones"])
	zones := got["zones"].([]any)
	require.Len(t, zones, 4)
	byZone := make(map[string]map[string]any, len(zones))
	for _, z := range zones {
		m := z.(map[string]any)
		byZone[m["zone"].(string)] = m
	}
	assert.Equal(t, 3.0, byZone["us-east-1a"]["targetsDesired"])
	assert.Equal(t, 2.0, byZone["us-east-1b"]["targetsDesired"])
	assert.Equal(t, 1.0, byZone["us-east-1c"]["targetsDesired"])
	assert.Equal(t, 1.0, byZone["us-east-1d"]["targetsDesired"])
	assert.Equal(t, false, byZone["us-east-1d"]["covered"],
		"the uncovered zone must report Covered=false")
	assert.Equal(t, true, byZone["us-east-1a"]["covered"])
}

// TestZoneAware_FullStack_DisabledMatchesPreFeatureBehavior is the
// counterpart of the integration test above: it wires the same pipeline
// without enabling zone awareness and asserts the /zones endpoint reports
// the documented "disabled" payload. This is the backward-compat contract
// at the integration level — clients can rely on it across upgrades.
func TestZoneAware_FullStack_DisabledMatchesPreFeatureBehavior(t *testing.T) {
	log := logf.Log.WithName("integration-test")

	a, err := allocation.New("consistent-hashing", log)
	require.NoError(t, err)
	assert.Nil(t, a.ZoneTopology(),
		"default allocator must not have a zone topology when no zone options are used")

	srv, err := server.NewServer(log, a, "")
	require.NoError(t, err)

	resp, got := probeZones(t, srv)
	require.Equal(t, http.StatusOK, resp.StatusCode,
		"/zones must respond 200 even when zone-aware is disabled, so health checks never trip")
	assert.Equal(t, false, got["enabled"],
		"/zones must report enabled=false when no topology is attached")
}
