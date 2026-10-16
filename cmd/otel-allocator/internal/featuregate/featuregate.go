// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package featuregate

import (
	"flag"

	"go.opentelemetry.io/collector/featuregate"
)

// TargetsRemainingAttributes is the feature gate that records the opentelemetry_allocator_targets_remaining
// metric per job and namespace.
var TargetsRemainingAttributes = featuregate.GlobalRegistry().MustRegister(
	"targetallocator.targetsremainingattributes",
	featuregate.StageAlpha,
	featuregate.WithRegisterDescription("records the opentelemetry_allocator_targets_remaining metric with job.name and k8s.namespace.name attributes"),
	featuregate.WithRegisterFromVersion("v0.161.0"),
	featuregate.WithRegisterReferenceURL("https://github.com/open-telemetry/opentelemetry-operator/issues/4637"),
)

// Flags creates a new FlagSet that represents the available featuregate flags using the supplied featuregate registry.
func Flags(reg *featuregate.Registry) *flag.FlagSet {
	flagSet := new(flag.FlagSet)
	reg.RegisterFlags(flagSet)
	return flagSet
}
