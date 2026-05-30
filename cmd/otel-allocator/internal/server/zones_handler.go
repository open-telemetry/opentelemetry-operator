// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"cmp"
	"net/http"
	"slices"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/allocation"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/target"
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

// ZonesHTMLHandler renders the zone topology as a human-readable HTML page
// served at /debug/zones. It mirrors the data exposed by ZonesHandler
// (and the /zones JSON endpoint) but formats it the way operators expect
// from the existing /debug/* pages — clickable collector names, sorted
// rows, an uncovered-zones table when applicable.
//
// When zone-aware allocation is disabled this renders a single-line
// notice explaining how to enable it, rather than a 404, so curious
// operators clicking the link from the index page get a useful answer
// instead of an error.
func (s *Server) ZonesHTMLHandler(c *gin.Context) {
	c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")

	WriteHTMLPageHeader(c.Writer, HeaderData{
		Title: "OpenTelemetry Target Allocator - Zones",
	})

	zt := s.allocator.ZoneTopology()
	if zt == nil {
		// Zone awareness is off — render an empty page (just the
		// navigation chrome from header + footer). The Zones nav link
		// always stays visible so the operator can land here, but we
		// don't show any content because there is none to show.
		WriteHTMLPageFooter(c.Writer)
		return
	}

	// Top-level zone topology table. Each zone row links to /debug/zone
	// where operators can drill into the collectors in that zone with
	// their job/target counts — the same view the index page shows for
	// the global pool, scoped to one zone.
	snap := zt.Snapshot()
	WriteHTMLPropertiesTable(c.Writer, PropertiesTableData{
		Headers: []string{"Zone", "Collectors", "Targets Desired", "Status"},
		Rows: func() [][]Cell {
			var rows [][]Cell
			for _, z := range snap {
				zoneLabel := z.Zone
				if zoneLabel == "" {
					zoneLabel = "(zone-less)"
				}
				status := "covered"
				if !z.Covered {
					status = "UNCOVERED — targets failover to global pool"
				}
				rows = append(rows, []Cell{
					zoneAnchorLink(zoneLabel),
					NewCell(strconv.Itoa(len(z.Collectors))),
					NewCell(strconv.Itoa(z.TargetsDesired)),
					NewCell(status),
				})
			}
			return rows
		}(),
	})

	// Dedicated uncovered-zones callout so operators don't have to scan
	// the Status column to find problems.
	if uncovered := zt.UncoveredZones(); len(uncovered) > 0 {
		WriteHTMLPropertiesTable(c.Writer, PropertiesTableData{
			Headers: []string{"Uncovered Zones (targets failing over)"},
			Rows: func() [][]Cell {
				var rows [][]Cell
				for _, z := range uncovered {
					rows = append(rows, []Cell{zoneAnchorLink(z)})
				}
				return rows
			}(),
		})
	}

	WriteHTMLPageFooter(c.Writer)
}

// ZoneHTMLHandler renders a per-zone detail page at /debug/zone?zone_id=<zone>.
// It mirrors the index page's collectors table but scopes the rows to
// collectors that live in the requested zone, with their per-collector
// job count and target count. This is the drill-down operators reach for
// when /debug/zones says a zone has N collectors and they want to know
// how those collectors are loaded.
//
// Returns 404 with a helpful page when the requested zone is unknown
// (covers typos in the URL) and a "zone awareness disabled" notice when
// zone tracking is off entirely.
func (s *Server) ZoneHTMLHandler(c *gin.Context) {
	c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")

	zoneIDValues := c.Request.URL.Query()["zone_id"]
	if len(zoneIDValues) != 1 {
		c.Status(http.StatusBadRequest)
		WriteHTMLBadRequest(c.Writer, BadRequestData{
			Error:   "Expected zone_id in the query string",
			Example: "/debug/zone?zone_id=us-east-1a",
		})
		return
	}
	zoneID := zoneIDValues[0]

	zt := s.allocator.ZoneTopology()
	if zt == nil {
		WriteHTMLPageHeader(c.Writer, HeaderData{Title: "Zone: " + zoneID})
		WriteHTMLPropertiesTable(c.Writer, PropertiesTableData{
			Headers: []string{"Zone awareness"},
			Rows: [][]Cell{
				{NewCell("disabled — set topology.zone_aware: true in the allocator config to enable")},
			},
		})
		WriteHTMLPageFooter(c.Writer)
		return
	}

	// Locate the zone in the snapshot so we can render Covered status and
	// total desiring targets alongside the collector breakdown.
	var (
		found          bool
		snapshot       allocation.ZoneSnapshot
		fullSnapshot   = zt.Snapshot()
		lookupKey      = zoneID
		emptyZoneInURL = zoneID == "(zone-less)"
	)
	if emptyZoneInURL {
		// /debug/zones surfaces the zone-less bucket as "(zone-less)";
		// the underlying key in the topology is "".
		lookupKey = ""
	}
	for _, z := range fullSnapshot {
		if z.Zone == lookupKey {
			snapshot = z
			found = true
			break
		}
	}
	if !found {
		c.Status(http.StatusNotFound)
		WriteHTMLNotFound(c.Writer, NotFoundData{
			ResourceType: "Zone",
			ResourceName: zoneID,
		})
		return
	}

	WriteHTMLPageHeader(c.Writer, HeaderData{Title: "Zone: " + zoneID})

	// Zone summary so operators see the same key facts the /debug/zones
	// table shows, plus a direct backlink to that page.
	status := "covered"
	if !snapshot.Covered {
		status = "UNCOVERED — targets failover to global pool"
	}
	WriteHTMLPropertiesTable(c.Writer, PropertiesTableData{
		Headers: []string{"Property", "Value"},
		Rows: [][]Cell{
			{NewCell("Zone"), NewCell(zoneID)},
			{NewCell("Collectors"), NewCell(strconv.Itoa(len(snapshot.Collectors)))},
			{NewCell("Targets Desired"), NewCell(strconv.Itoa(snapshot.TargetsDesired))},
			{NewCell("Status"), NewCell(status)},
			{NewCell("All Zones"), zonesAnchorLink()},
		},
	})

	if snapshot.Covered {
		// Covered zone — show the collectors that live here, with the
		// same shape the index page uses for the global collectors
		// table so operators see a familiar layout. We pull the
		// collectors via CollectorsSnapshot rather than Collectors() so
		// the read is lock-free and race-free against concurrent
		// SetTargets / SetCollectors calls — the alternative would
		// expose Collector.NumTargets and Collector.TargetsPerJob to
		// torn reads.
		collectors := s.allocator.CollectorsSnapshot()
		WriteHTMLPropertiesTable(c.Writer, PropertiesTableData{
			Headers: []string{"Collector", "Zone", "Job Count", "Target Count"},
			Rows: func() [][]Cell {
				var rows [][]Cell
				for _, colName := range snapshot.Collectors {
					col, ok := collectors[colName]
					if !ok {
						// Topology shows a collector the allocator no
						// longer knows about (transient state between
						// SetCollectors calls). Render the row so the
						// page doesn't lie about emptiness; counts are 0.
						rows = append(rows, []Cell{
							collectorAnchorLink(colName),
							NewCell(zoneID),
							NewCell("0"),
							NewCell("0"),
						})
						continue
					}
					zoneCell := col.Zone
					if zoneCell == "" {
						zoneCell = "(zone-less)"
					}
					rows = append(rows, []Cell{
						collectorAnchorLink(colName),
						NewCell(zoneCell),
						NewCell(strconv.Itoa(len(col.TargetsPerJob))),
						NewCell(strconv.Itoa(col.NumTargets)),
					})
				}
				return rows
			}(),
		})
	} else {
		// Uncovered zone — there are no in-zone collectors to list.
		// Instead, show the actual targets that desired this zone so
		// the operator can see who's affected by the missing-zone
		// failover (and which global collector is now scraping them).
		WriteHTMLPropertiesTable(c.Writer, PropertiesTableData{
			Headers: []string{"Job", "Target", "Failover Collector"},
			Rows: func() [][]Cell {
				var rows [][]Cell
				items := s.allocator.TargetItems()
				// Stable sort by job then URL so the page doesn't shuffle
				// between refreshes.
				keys := make([]target.ItemHash, 0, len(items))
				for k := range items {
					keys = append(keys, k)
				}
				slices.SortFunc(keys, func(a, b target.ItemHash) int {
					ai, bi := items[a], items[b]
					if ai.JobName != bi.JobName {
						return cmp.Compare(ai.JobName, bi.JobName)
					}
					return cmp.Compare(ai.TargetURL, bi.TargetURL)
				})
				for _, k := range keys {
					item := items[k]
					if zt.GetTargetZone(item) != lookupKey {
						continue
					}
					failoverCell := NewCell("(unassigned)")
					if item.CollectorName != "" {
						failoverCell = collectorAnchorLink(item.CollectorName)
					}
					rows = append(rows, []Cell{
						jobAnchorLink(item.JobName),
						targetAnchorLink(item),
						failoverCell,
					})
				}
				return rows
			}(),
		})
	}

	WriteHTMLPageFooter(c.Writer)
}
