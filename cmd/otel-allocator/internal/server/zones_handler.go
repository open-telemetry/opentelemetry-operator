// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/allocation"
)

// zonesResponse is the JSON payload of the /zones endpoint. It mirrors what
// the allocator's ZoneTopology knows at the moment the request is served.
// The shape is intentionally flat so consumers (UI dashboards, scripts,
// dashboards generators) can render it without recursing.
type zonesResponse struct {
	// Enabled reports whether zone-aware allocation is currently active.
	// When false, the rest of the payload is a fixed empty/false response
	// — callers can detect non-zone-aware setups without having to inspect
	// the allocator configuration separately.
	Enabled bool `json:"enabled"`
	// Zones lists every zone seen by the allocator (either with at least
	// one collector or with at least one desiring target). Each entry
	// carries its collector membership and the number of targets that
	// want that zone, regardless of where those targets actually got
	// assigned (so failover and maxSkew spillover stay visible).
	Zones []allocation.ZoneSnapshot `json:"zones"`
	// UncoveredZones is the subset of named zones (excluding the
	// zone-less bucket) that currently have desiring targets but no
	// collectors. It duplicates information that can be derived from
	// Zones[].Covered == false, but is provided as a top-level field for
	// quick health checks ("is anything wrong with my zone coverage?").
	UncoveredZones []string `json:"uncoveredZones"`
}

// ZonesHandler serves a JSON snapshot of the current zone topology. When
// zone-aware allocation is disabled the endpoint still responds with HTTP
// 200 so that clients can rely on it being available, but the payload
// contains `enabled: false` and an empty zone list.
func (s *Server) ZonesHandler(c *gin.Context) {
	zt := s.allocator.ZoneTopology()
	resp := zonesResponse{}
	if zt != nil {
		resp.Enabled = true
		resp.Zones = zt.Snapshot()
		resp.UncoveredZones = zt.UncoveredZones()
	}
	// We always return 200 — an empty/disabled response is still a valid
	// answer to "what's the zone state right now?". Returning a 4xx/5xx
	// here would force every operator dashboard to special-case it.
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(http.StatusOK)
	s.jsonHandler(c.Writer, resp)
}
