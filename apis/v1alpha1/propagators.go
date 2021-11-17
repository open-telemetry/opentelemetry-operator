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
	// Propagator represents the propagation type.
	// +kubebuilder:validation:Enum=tracecontext;baggage;b3;b3multi;jaeger;xray;ottrace;none
	Propagator string
)

const (
	// TraceContext represents W3C Trace Context.
	TraceContext Propagator = "tracecontext"
	// Baggage represents W3C Baggage.
	Baggage Propagator = "baggage"
	// B3 represents B3 Single.
	B3 Propagator = "b3"
	// B3Multi represents B3 Multi.
	B3Multi Propagator = "b3multi"
	// Jaeger represents Jaeger.
	Jaeger Propagator = "jaeger"
	// XRay represents AWS X-Ray.
	XRay Propagator = "xray"
	// OTTrace represents OT Trace.
	OTTrace Propagator = "ottrace"
	// None represents automatically configured propagator.
	None Propagator = "none"
)
