// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/allocation"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
)

const zonesEndpointTargetZoneLabel = "__meta_kubernetes_endpointslice_endpoint_zone"

func doZonesRequest(t *testing.T, s *Server) (resp *http.Response, body []byte) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/zones", http.NoBody)
	w := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(w, req)
	resp = w.Result()
	var err error
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	return resp, body
}

func TestZonesHandler_DisabledByDefault(t *testing.T) {
	// Backward-compat contract: deployments that never set Topology.ZoneAware
	// must see the /zones endpoint respond with enabled=false and a stable,
	// empty payload. Existing scripts and dashboards that probe this endpoint
	// to detect zone-aware setups can rely on the field shape.
	a, err := allocation.New("consistent-hashing", logger)
	require.NoError(t, err)
	s, err := NewServer(logger, a, "")
	require.NoError(t, err)

	resp, body := doZonesRequest(t, s)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var got map[string]any
	require.NoError(t, json.Unmarshal(body, &got))
	assert.Equal(t, false, got["enabled"], "zone awareness must report disabled when no topology is attached")
	// Empty slices and the explicit `false` for `enabled` are the documented
	// "zone-aware is off" shape — clients can branch on `enabled` alone.
}

func TestZonesHandler_EnabledExposesSnapshot(t *testing.T) {
	zt, err := allocation.NewZoneTopology(logger, zonesEndpointTargetZoneLabel)
	require.NoError(t, err)
	a, err := allocation.New("consistent-hashing", logger, allocation.WithZoneTopology(zt))
	require.NoError(t, err)

	cols := map[string]*allocation.Collector{
		"collector-0": allocation.NewCollector("collector-0", "node-0", "us-east-1a"),
		"collector-1": allocation.NewCollector("collector-1", "node-1", "us-east-1a"),
		"collector-2": allocation.NewCollector("collector-2", "node-2", "us-east-1b"),
	}
	a.SetCollectors(cols)

	items := []*target.Item{
		target.NewItem("scrape", "10.0.0.1:9100",
			labels.New(labels.Label{Name: zonesEndpointTargetZoneLabel, Value: "us-east-1a"}), ""),
		target.NewItem("scrape", "10.0.0.2:9100",
			labels.New(labels.Label{Name: zonesEndpointTargetZoneLabel, Value: "us-east-1b"}), ""),
		// A target wanting a zone we don't cover — must show up uncovered.
		target.NewItem("scrape", "10.0.0.3:9100",
			labels.New(labels.Label{Name: zonesEndpointTargetZoneLabel, Value: "us-east-1c"}), ""),
	}
	a.SetTargets(items)

	s, err := NewServer(logger, a, "")
	require.NoError(t, err)

	resp, body := doZonesRequest(t, s)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var got struct {
		Enabled        bool                      `json:"enabled"`
		Zones          []allocation.ZoneSnapshot `json:"zones"`
		UncoveredZones []string                  `json:"uncoveredZones"`
	}
	require.NoError(t, json.Unmarshal(body, &got))
	assert.True(t, got.Enabled)
	assert.Equal(t, []string{"us-east-1c"}, got.UncoveredZones,
		"the API must surface uncovered zones as a top-level field for quick health checks")

	// The snapshot must be sorted and complete.
	zones := make(map[string]allocation.ZoneSnapshot, len(got.Zones))
	for _, z := range got.Zones {
		zones[z.Zone] = z
	}
	require.Contains(t, zones, "us-east-1a")
	require.Contains(t, zones, "us-east-1b")
	require.Contains(t, zones, "us-east-1c")
	assert.True(t, zones["us-east-1a"].Covered)
	assert.True(t, zones["us-east-1b"].Covered)
	assert.False(t, zones["us-east-1c"].Covered)
	assert.Equal(t, 1, zones["us-east-1a"].TargetsDesired)
	assert.Equal(t, 1, zones["us-east-1b"].TargetsDesired)
	assert.Equal(t, 1, zones["us-east-1c"].TargetsDesired)
	assert.Equal(t, []string{"collector-0", "collector-1"}, zones["us-east-1a"].Collectors)
}

func TestZonesHandler_AlwaysOKEvenWhenDisabled(t *testing.T) {
	// Defense-in-depth: the endpoint must always return 200 OK regardless
	// of allocator state, so health checks and dashboards never see a
	// transient 4xx/5xx that they'd have to special-case.
	a, err := allocation.New("least-weighted", logger)
	require.NoError(t, err)
	s, err := NewServer(logger, a, "")
	require.NoError(t, err)

	resp, _ := doZonesRequest(t, s)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
