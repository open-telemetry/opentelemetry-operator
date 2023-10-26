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

package v1alpha2

import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

// ConvertTo converts this OpenTelemetryCollector to the Hub version (v1alpha1).
func (src *OpenTelemetryCollector) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1alpha1.OpenTelemetryCollector)
	dst.Spec = src.Spec
	return nil
}

// ConvertFrom converts from the Hub version (v1alpha1) to this version.
func (dst *OpenTelemetryCollector) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1alpha1.OpenTelemetryCollector)
	src.Spec = dst.Spec
	return nil
}
