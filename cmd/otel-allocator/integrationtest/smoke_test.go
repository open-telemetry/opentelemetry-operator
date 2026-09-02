// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package integrationtest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
)

// TestSmoke is an end-to-end smoke test: a mock target exposes a counter, the
// allocator discovers it and assigns it to the (single) collector, the receiver
// pulls the assignment from the allocator and scrapes the target, and the scraped
// metric lands in the sink.
func TestSmoke(t *testing.T) {
	const exposition = "# HELP my_counter A test counter.\n" +
		"# TYPE my_counter counter\n" +
		"my_counter_total{foo=\"bar\"} 42\n"

	target := startMockTarget(t, exposition)
	taEndpoint := startTargetAllocator(t, scrapeConfig(t, target.Host))

	sink := new(consumertest.MetricsSink)
	startReceiver(t, taEndpoint, sink)

	require.Eventuallyf(t, func() bool {
		_, ok := metricNames(sink)["my_counter_total"]
		return ok
	}, 30*time.Second, 250*time.Millisecond, "expected my_counter_total to be scraped via the allocator")

	// The receiver also synthesizes `up` for the scraped target.
	assert.Contains(t, metricNames(sink), "up")
}
