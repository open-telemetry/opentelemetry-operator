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

type (
	// SamplerType represents sampler type.
	// +kubebuilder:validation:Enum=always_on;always_off;traceidratio;parentbased_always_on;parentbased_always_off;parentbased_traceidratio;jaeger_remote;xray
	SamplerType string
)

const (
	// AlwaysOn represents AlwaysOnSampler.
	AlwaysOn SamplerType = "always_on"
	// AlwaysOff represents AlwaysOffSampler.
	AlwaysOff SamplerType = "always_off"
	// TraceIDRatio represents TraceIdRatioBased.
	TraceIDRatio SamplerType = "traceidratio"
	// ParentBasedAlwaysOn represents ParentBased(root=AlwaysOnSampler).
	ParentBasedAlwaysOn SamplerType = "parentbased_always_on"
	// ParentBasedAlwaysOff represents ParentBased(root=AlwaysOffSampler).
	ParentBasedAlwaysOff SamplerType = "parentbased_always_off"
	// ParentBasedTraceIDRatio represents ParentBased(root=TraceIdRatioBased).
	ParentBasedTraceIDRatio SamplerType = "parentbased_traceidratio"
	// JaegerRemote represents JaegerRemoteSampler.
	JaegerRemote SamplerType = "jaeger_remote"
	// ParentBasedJaegerRemote represents ParentBased(root=JaegerRemoteSampler).
	ParentBasedJaegerRemote SamplerType = "parentbased_jaeger_remote"
	// XRay represents AWS X-Ray Centralized Sampling.
	XRaySampler SamplerType = "xray"
)
