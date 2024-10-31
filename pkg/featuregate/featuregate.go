// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package featuregate

import (
	"flag"

	"go.opentelemetry.io/collector/featuregate"
)

const (
	FeatureGatesFlag = "feature-gates"
)

var (
	// CollectorUsesTargetAllocatorCR is the feature gate that enables the OpenTelemetryCollector reconciler to generate
	// TargetAllocator CRs instead of generating the manifests for its resources directly.
	CollectorUsesTargetAllocatorCR = featuregate.GlobalRegistry().MustRegister(
		"operator.collector.targetallocatorcr",
		featuregate.StageAlpha,
		featuregate.WithRegisterDescription("causes collector reconciliation to create a target allocator CR instead of creating resources directly"),
		featuregate.WithRegisterFromVersion("v0.112.0"),
	)
	// EnableNativeSidecarContainers is the feature gate that controls whether a
	// sidecar should be injected as a native sidecar or the classic way.
	// Native sidecar containers have been available since kubernetes v1.28 in
	// alpha and v1.29 in beta.
	// It needs to be enabled with +featureGate=SidecarContainers.
	// See:
	// https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/#feature-gates-for-alpha-or-beta-features
	EnableNativeSidecarContainers = featuregate.GlobalRegistry().MustRegister(
		"operator.sidecarcontainers.native",
		featuregate.StageAlpha,
		featuregate.WithRegisterDescription("controls whether the operator supports sidecar containers as init containers. Should only be enabled on k8s v1.29+"),
	)
	// PrometheusOperatorIsAvailable is the feature gate that enables features associated to the Prometheus Operator.
	PrometheusOperatorIsAvailable = featuregate.GlobalRegistry().MustRegister(
		"operator.observability.prometheus",
		featuregate.StageBeta,
		featuregate.WithRegisterDescription("enables features associated to the Prometheus Operator"),
		featuregate.WithRegisterFromVersion("v0.82.0"),
	)
	// SetGolangFlags is the feature gate that enables automatically setting GOMEMLIMIT and GOMAXPROCS for the
	// collector, bridge, and target allocator.
	SetGolangFlags = featuregate.GlobalRegistry().MustRegister(
		"operator.golang.flags",
		featuregate.StageAlpha,
		featuregate.WithRegisterDescription("enables feature to set GOMEMLIMIT and GOMAXPROCS automatically"),
		featuregate.WithRegisterFromVersion("v0.100.0"),
	)
	// EnableTargetAllocatorMTLS is the feature gate that enables mTLS between the target allocator and the collector.
	EnableTargetAllocatorMTLS = featuregate.GlobalRegistry().MustRegister(
		"operator.targetallocator.mtls",
		featuregate.StageAlpha,
		featuregate.WithRegisterDescription("enables mTLS between the target allocator and the collector"),
		featuregate.WithRegisterFromVersion("v0.111.0"),
	)
	// EnableConfigDefaulting is the feature gate that enables the operator to default the endpoint for known components.
	EnableConfigDefaulting = featuregate.GlobalRegistry().MustRegister(
		"operator.collector.default.config",
		featuregate.StageBeta,
		featuregate.WithRegisterDescription("enables the operator to default the endpoint for known components"),
		featuregate.WithRegisterFromVersion("v0.110.0"),
	)
)

// Flags creates a new FlagSet that represents the available featuregate flags using the supplied featuregate registry.
func Flags(reg *featuregate.Registry) *flag.FlagSet {
	flagSet := new(flag.FlagSet)
	reg.RegisterFlags(flagSet)
	return flagSet
}
