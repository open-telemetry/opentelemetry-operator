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

package distros

var K8sReceivers = []string{
	"filelog",
	"fluentforward",
	"hostmetrics",
	"httpcheck",
	"jaeger",
	"journald",
	"k8s_cluster",
	"k8s_events",
	"k8sobjects",
	"kubeletstats",
	"opencensus",
	"otlp",
	"prometheus",
	"receiver_creator",
	"zipkin",
}
var K8sProcessors = []string{
	"attributes",
	"batch",
	"cumulativetodelta",
	"deltatorate",
	"filter",
	"groupbyattrs",
	"groupbytrace",
	"k8sattributes",
	"memory_limiter",
	"metricstransform",
	"probabilistic_sampler",
	"redaction",
	"remotetap",
	"resource",
	"resourcedetection",
	"tail_sampling",
	"transform",
}
var K8sExporters = []string{
	"debug",
	"file",
	"loadbalancing",
	"otlp",
	"otlphttp",
}

var K8sExtensions = []string{
	"basicauth",
	"bearertokenauth",
	"file_storage",
	"headers_setter",
	"health_check",
	"host_observer",
	"k8s_observer",
	"oauth2client",
	"oidc",
	"pprof",
	"zpages",
}

var K8sConnectors = []string{
	"count",
	"forward",
	"routing",
	"servicegraph",
	"spanmetrics",
}
