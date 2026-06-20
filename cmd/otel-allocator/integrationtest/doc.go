// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package integrationtest wires the target allocator and the OpenTelemetry
// Collector prometheus receiver together in-process: a mock scrape target, the
// real allocator pipeline (discovery + relabel + allocation + HTTP SD server),
// and a real receiver pointed at the allocator. Scraped metrics are captured in
// an in-memory OTLP sink for assertions — no Kubernetes cluster, no external
// backend.
//
// It is its own module so the heavy collector/receiver dependency graph stays
// out of the target allocator binary's module.
package integrationtest
