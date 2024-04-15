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

var CoreReceivers = []string{
	"hostmetrics",
	"jaeger",
	"kafka",
	"nop",
	"opencensus",
	"otlp",
	"prometheus",
	"zipkin",
}
var CoreProcessors = []string{
	"attributes",
	"batch",
	"filter",
	"memory_limiter",
	"probabilistic_sampler",
	"resource",
	"span",
}
var CoreExporters = []string{
	"debug",
	"file",
	"kafka",
	"logging",
	"nop",
	"opencensus",
	"otlp",
	"otlphttp",
	"prometheus",
	"prometheusremotewrite",
	"zipkin",
}

var CoreExtensions = []string{
	"health_check",
	"memory_ballast",
	"pprof",
	"zpages",
}

var CoreConnectors = []string{
	"forward",
}
