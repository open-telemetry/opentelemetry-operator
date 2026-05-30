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

// htmlGet drives a GET against the server's HTTP handler and returns the
// response + body. Centralizes the HTML test boilerplate so each test
// case stays focused on its assertions.
func htmlGet(t *testing.T, s *Server, path string) (*http.Response, string) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, http.NoBody)
	w := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(w, req)
	resp := w.Result()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	return resp, string(body)
}

func newZoneAwareServerWithFixture(t *testing.T) *Server {
	t.Helper()
	zt, err := allocation.NewZoneTopology(logger, zonesEndpointTargetZoneLabel)
	require.NoError(t, err)
	a, err := allocation.New("least-weighted", logger, allocation.WithZoneTopology(zt))
	require.NoError(t, err)
	a.SetCollectors(map[string]*allocation.Collector{
		"collector-a-1": allocation.NewCollector("collector-a-1", "node-a-1", "us-east-1a"),
		"collector-a-2": allocation.NewCollector("collector-a-2", "node-a-2", "us-east-1a"),
		"collector-b-1": allocation.NewCollector("collector-b-1", "node-b-1", "us-east-1b"),
	})
	a.SetTargets([]*target.Item{
		target.NewItem("node-exporter", "10.0.1.1:9100",
			labels.New(labels.Label{Name: zonesEndpointTargetZoneLabel, Value: "us-east-1a"}), ""),
		target.NewItem("node-exporter", "10.0.2.1:9100",
			labels.New(labels.Label{Name: zonesEndpointTargetZoneLabel, Value: "us-east-1b"}), ""),
		// Uncovered zone target — should appear in /debug/zone?zone_id=us-east-1c.
		target.NewItem("cadvisor", "10.0.3.1:4194",
			labels.New(labels.Label{Name: zonesEndpointTargetZoneLabel, Value: "us-east-1c"}), ""),
	})
	s, err := NewServer(logger, a, "")
	require.NoError(t, err)
	return s
}

func TestZonesHTMLHandler_RendersTopologyAndUncoveredCallout(t *testing.T) {
	// The HTML view at /debug/zones must mirror the same picture the JSON
	// API exposes, plus a dedicated uncovered-zones callout at the
	// bottom. Zone names must be clickable so operators can drill into
	// each zone without manually editing the URL.
	s := newZoneAwareServerWithFixture(t)
	resp, body := htmlGet(t, s, "/debug/zones")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// Zone rows: each zone has a clickable link to /debug/zone.
	assert.Contains(t, body, `href="/debug/zone?zone_id=us-east-1a"`)
	assert.Contains(t, body, `href="/debug/zone?zone_id=us-east-1b"`)
	assert.Contains(t, body, `href="/debug/zone?zone_id=us-east-1c"`)
	// Status text — case-sensitive UNCOVERED so operators can grep for it.
	assert.Contains(t, body, "UNCOVERED")
	// Uncovered callout at the bottom must list the uncovered zone.
	assert.Contains(t, body, "Uncovered Zones")
}

func TestZonesHTMLHandler_DisabledRendersEmptyBody(t *testing.T) {
	// With zone-aware off, /debug/zones must still return 200 (so the
	// nav link never 404s) but render no body content beyond the header
	// + footer chrome. This is the documented "empty" state.
	a, err := allocation.New("consistent-hashing", logger)
	require.NoError(t, err)
	s, err := NewServer(logger, a, "")
	require.NoError(t, err)
	resp, body := htmlGet(t, s, "/debug/zones")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// No data tables, no notice — just chrome.
	assert.NotContains(t, body, "Targets Desired")
	assert.NotContains(t, body, "Uncovered Zones")
}

func TestZoneHTMLHandler_CoveredZoneShowsCollectorsTable(t *testing.T) {
	// /debug/zone?zone_id=<covered> must show a Summary block + a
	// collectors-in-zone table with the same columns the global view
	// uses (Collector / Zone / Job Count / Target Count).
	s := newZoneAwareServerWithFixture(t)
	resp, body := htmlGet(t, s, "/debug/zone?zone_id=us-east-1a")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, body, "Zone: us-east-1a")
	assert.Contains(t, body, "Targets Desired")
	assert.Contains(t, body, "Job Count")
	assert.Contains(t, body, "collector-a-1")
	assert.Contains(t, body, "collector-a-2")
	assert.NotContains(t, body, "Failover Collector",
		"covered zones must not render the failover-targets table")
}

func TestZoneHTMLHandler_UncoveredZoneShowsTargetsTable(t *testing.T) {
	// For an uncovered zone, the page must replace the empty collectors
	// table with a list of targets that desired this zone, including
	// the collector each one was failed over to. This is the actual
	// operator value the user explicitly asked for.
	s := newZoneAwareServerWithFixture(t)
	resp, body := htmlGet(t, s, "/debug/zone?zone_id=us-east-1c")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, body, "Zone: us-east-1c")
	assert.Contains(t, body, "UNCOVERED")
	// Failover targets table — name, columns, and target URL all present.
	assert.Contains(t, body, "Failover Collector")
	assert.Contains(t, body, "cadvisor")
	assert.Contains(t, body, "10.0.3.1:4194")
}

func TestZoneHTMLHandler_UnknownZoneReturns404(t *testing.T) {
	// Typos in the URL must not return an empty page — that would look
	// like a working zone with no data. Return a proper 404 instead.
	s := newZoneAwareServerWithFixture(t)
	resp, _ := htmlGet(t, s, "/debug/zone?zone_id=nonexistent")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestZoneHTMLHandler_MissingZoneIDReturns400(t *testing.T) {
	// Reaching /debug/zone without a zone_id is a programmer error
	// (every link from /debug/zones includes one), but we still want a
	// clean 400 with the example URL so it's obvious what went wrong.
	s := newZoneAwareServerWithFixture(t)
	resp, body := htmlGet(t, s, "/debug/zone")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, body, "zone_id")
}
