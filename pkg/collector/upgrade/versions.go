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

package upgrade

import (
	"github.com/Masterminds/semver/v3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/signalfx/splunk-otel-operator/api/v1alpha1"
)

type upgradeFunc func(cl client.Client, otelcol *v1alpha1.SplunkOtelAgent) (*v1alpha1.SplunkOtelAgent, error)

type otelcolVersion struct {
	semver.Version
	upgrade upgradeFunc
}

var (
	versions = []otelcolVersion{
		{
			Version: *semver.MustParse("0.2.10"),
			upgrade: upgrade0_2_10,
		},
		{
			Version: *semver.MustParse("0.9.0"),
			upgrade: upgrade0_9_0,
		},
		{
			Version: *semver.MustParse("0.15.0"),
			upgrade: upgrade0_15_0,
		},
		{
			Version: *semver.MustParse("0.19.0"),
			upgrade: upgrade0_19_0,
		},
		{
			Version: *semver.MustParse("0.24.0"),
			upgrade: upgrade0_24_0,
		},
		{
			Version: *semver.MustParse("0.31.0"),
			upgrade: upgrade0_31_0,
		},
	}

	// Latest represents the latest version that we need to upgrade. This is not necessarily the latest known version.
	Latest = versions[len(versions)-1]
)
