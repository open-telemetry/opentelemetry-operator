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
	EnableJavaAutoInstrumentationSupport = featuregate.GlobalRegistry().MustRegister(
		"operator.autoinstrumentation.java",
		featuregate.StageBeta,
		featuregate.WithRegisterDescription("controls whether the operator supports Java auto-instrumentation"),
		featuregate.WithRegisterFromVersion("v0.76.1"),
	)
	EnableNodeJSAutoInstrumentationSupport = featuregate.GlobalRegistry().MustRegister(
		"operator.autoinstrumentation.nodejs",
		featuregate.StageBeta,
		featuregate.WithRegisterDescription("controls whether the operator supports NodeJS auto-instrumentation"),
		featuregate.WithRegisterFromVersion("v0.76.1"),
	)
	EnableGoAutoInstrumentationSupport = featuregate.GlobalRegistry().MustRegister(
		"operator.autoinstrumentation.go",
		featuregate.StageAlpha,
		featuregate.WithRegisterDescription("controls whether the operator supports Golang auto-instrumentation"),
		featuregate.WithRegisterFromVersion("v0.77.0"),
	)
	EnableNginxAutoInstrumentationSupport = featuregate.GlobalRegistry().MustRegister(
		"operator.autoinstrumentation.nginx",
		featuregate.StageAlpha,
		featuregate.WithRegisterDescription("controls whether the operator supports Nginx auto-instrumentation"),
		featuregate.WithRegisterFromVersion("v0.86.0"),
	)
	// EnableTargetAllocatorRewrite is the feature gate that controls whether the collector's configuration should
	// automatically be rewritten when the target allocator is enabled.
	EnableTargetAllocatorRewrite = featuregate.GlobalRegistry().MustRegister(
		"operator.collector.rewritetargetallocator",
		featuregate.StageBeta,
		featuregate.WithRegisterDescription("controls whether the operator should configure the collector's targetAllocator configuration"),
		featuregate.WithRegisterFromVersion("v0.76.1"),
	)

	// PrometheusOperatorIsAvailable is the feature gate that enables features associated to the Prometheus Operator.
	PrometheusOperatorIsAvailable = featuregate.GlobalRegistry().MustRegister(
		"operator.observability.prometheus",
		featuregate.StageBeta,
		featuregate.WithRegisterDescription("enables features associated to the Prometheus Operator"),
		featuregate.WithRegisterFromVersion("v0.82.0"),
	)
)

// Flags creates a new FlagSet that represents the available featuregate flags using the supplied featuregate registry.
func Flags(reg *featuregate.Registry) *flag.FlagSet {
	flagSet := new(flag.FlagSet)
	reg.RegisterFlags(flagSet)
	return flagSet
}
