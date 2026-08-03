// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package integrationtest wires the target allocator and the OpenTelemetry
// Collector prometheus receiver together in-process: a mock scrape target, the
// real allocator pipeline (discovery + relabel + allocation + HTTP SD server),
// and a real receiver pointed at the allocator. Scraped metrics are captured in
// an in-memory OTLP sink for assertions — no Kubernetes cluster, no external
// backend.
//
// Most tests point a scrape config's static targets at the mock. Tests that need
// discovery to change over time (a pod whose labels are updated, say) instead
// register their own discovery.Config and hand target groups to Prometheus's
// discovery manager themselves; see fakeKubeSD.
//
// It is its own module so the heavy collector/receiver dependency graph stays
// out of the target allocator binary's module.
package integrationtest
