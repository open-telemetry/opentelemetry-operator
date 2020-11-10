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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true

// OpenTelemetryCollectorDistribution is the Schema for the opentelemetrycollectordistributions API
type OpenTelemetryCollectorDistribution struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Image specifies the default image to use for this distribution. The image specified in the consuming resource should take precedence. Required.
	// +required
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Image string `json:"image,omitempty"`

	// Command specifies which command should be used to start the distribution, such as "/otelcontribcol". Ideally, this would be empty, as the container image would specify an appropriate Entrypoint. Optional.
	// See v1.Container#Command for
	// +optional
	// +listType=atomic
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	Command []string `json:"cmd,omitempty"`
}

// +kubebuilder:object:root=true

// OpenTelemetryCollectorDistributionList contains a list of OpenTelemetryCollectorDistribution
type OpenTelemetryCollectorDistributionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenTelemetryCollectorDistribution `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenTelemetryCollectorDistribution{}, &OpenTelemetryCollectorDistributionList{})
}
