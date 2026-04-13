// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package featuregate

import (
	"go.opentelemetry.io/collector/featuregate"
)

// EnableLabeledMetrics is the feature gate that enables labeled metrics for target allocator targets_remaining.
var EnableLabeledMetrics = featuregate.GlobalRegistry().MustRegister(
	"targetallocator.labeledmetrics",
	featuregate.StageAlpha,
	featuregate.WithRegisterDescription("enables labeled metrics (with job.name and k8s.namespace.name) for target allocator targets_remaining metric"),
	featuregate.WithRegisterFromVersion("v0.145.0"),
)
