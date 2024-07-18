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

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

// Deprecated use upgradeFuncV1beta1.
type upgradeFunc func(u VersionUpgrade, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error)
type upgradeFuncV1beta1 func(u VersionUpgrade, otelcol *v1beta1.OpenTelemetryCollector) (*v1beta1.OpenTelemetryCollector, error)

type otelcolVersion struct {
	// deprecated use upgradeV1beta1.
	upgrade        upgradeFunc
	upgradeV1beta1 upgradeFuncV1beta1
	semver.Version
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
		{
			Version: *semver.MustParse("0.36.0"),
			upgrade: upgrade0_36_0,
		},
		{
			Version: *semver.MustParse("0.38.0"),
			upgrade: upgrade0_38_0,
		},
		{
			Version: *semver.MustParse("0.39.0"),
			upgrade: upgrade0_39_0,
		},
		{
			Version: *semver.MustParse("0.41.0"),
			upgrade: upgrade0_41_0,
		},
		{
			Version: *semver.MustParse("0.43.0"),
			upgrade: upgrade0_43_0,
		},
		{
			Version: *semver.MustParse("0.56.0"),
			upgrade: upgrade0_56_0,
		},
		{
			Version: *semver.MustParse("0.57.2"),
			upgrade: upgrade0_57_2,
		},
		{
			Version: *semver.MustParse("0.61.0"),
			upgrade: upgrade0_61_0,
		},
		{
			Version:        *semver.MustParse("0.104.0"),
			upgradeV1beta1: upgrade0_104_0_TA,
		},
		{
			Version:        *semver.MustParse("0.104.0"),
			upgradeV1beta1: upgrade0_104_0,
		},
	}

	// Latest represents the latest version that we need to upgrade. This is not necessarily the latest known version.
	Latest = versions[len(versions)-1]
)
