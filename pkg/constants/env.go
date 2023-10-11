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

package constants

const (
	EnvOTELServiceName          = "OTEL_SERVICE_NAME"
	EnvOTELExporterOTLPEndpoint = "OTEL_EXPORTER_OTLP_ENDPOINT"
	EnvOTELResourceAttrs        = "OTEL_RESOURCE_ATTRIBUTES"
	EnvOTELPropagators          = "OTEL_PROPAGATORS"
	EnvOTELTracesSampler        = "OTEL_TRACES_SAMPLER"
	EnvOTELTracesSamplerArg     = "OTEL_TRACES_SAMPLER_ARG"

	InstrumentationPrefix                           = "instrumentation.opentelemetry.io/"
	AnnotationDefaultAutoInstrumentationJava        = InstrumentationPrefix + "default-auto-instrumentation-java-image"
	AnnotationDefaultAutoInstrumentationNodeJS      = InstrumentationPrefix + "default-auto-instrumentation-nodejs-image"
	AnnotationDefaultAutoInstrumentationPython      = InstrumentationPrefix + "default-auto-instrumentation-python-image"
	AnnotationDefaultAutoInstrumentationDotNet      = InstrumentationPrefix + "default-auto-instrumentation-dotnet-image"
	AnnotationDefaultAutoInstrumentationGo          = InstrumentationPrefix + "default-auto-instrumentation-go-image"
	AnnotationDefaultAutoInstrumentationApacheHttpd = InstrumentationPrefix + "default-auto-instrumentation-apache-httpd-image"
	AnnotationDefaultAutoInstrumentationNginx       = InstrumentationPrefix + "default-auto-instrumentation-nginx-image"

	EnvPodName  = "OTEL_RESOURCE_ATTRIBUTES_POD_NAME"
	EnvPodUID   = "OTEL_RESOURCE_ATTRIBUTES_POD_UID"
	EnvNodeName = "OTEL_RESOURCE_ATTRIBUTES_NODE_NAME"
)
