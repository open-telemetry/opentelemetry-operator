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

package flags

import (
	"flag"

	"go.opentelemetry.io/collector/featuregate"
)

const (
	featureGatesFlag = "feature-gates"
)

func Flags() *flag.FlagSet {
	flagSet := new(flag.FlagSet)
	flagSet.Var(featuregate.FlagValue{}, featureGatesFlag,
		"Comma-delimited list of feature gate identifiers. Prefix with '-' to disable the feature. '+' or no prefix will enable the feature.")
	return flagSet
}

func GetFeatureGatesFlag(flagSet *flag.FlagSet) featuregate.FlagValue {
	return flagSet.Lookup(featureGatesFlag).Value.(featuregate.FlagValue)
}
