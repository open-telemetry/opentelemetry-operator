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
	EnableDotnetAutoInstrumentationSupport = featuregate.GlobalRegistry().MustRegister(
		"operator.autoinstrumentation.dotnet",
		featuregate.StageBeta,
		featuregate.WithRegisterDescription("controls whether the operator supports .NET auto-instrumentation"))
	EnablePythonAutoInstrumentationSupport = featuregate.GlobalRegistry().MustRegister(
		"operator.autoinstrumentation.python",
		featuregate.StageBeta,
		featuregate.WithRegisterDescription("controls whether the operator supports Python auto-instrumentation"))
	EnableJavaAutoInstrumentationSupport = featuregate.GlobalRegistry().MustRegister(
		"operator.autoinstrumentation.java",
		featuregate.StageBeta,
		featuregate.WithRegisterDescription("controls whether the operator supports Java auto-instrumentation"))
)

// Flags creates a new FlagSet that represents the available featuregate flags using the supplied featuregate registry.
func Flags(reg *featuregate.Registry) *flag.FlagSet {
	flagSet := new(flag.FlagSet)
	flagSet.Var(featuregate.NewFlag(reg), FeatureGatesFlag,
		"Comma-delimited list of feature gate identifiers. Prefix with '-' to disable the feature. '+' or no prefix will enable the feature.")
	return flagSet
}
