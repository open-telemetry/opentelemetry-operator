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

package receivers

import (
	"github.com/open-telemetry/opentelemetry-operator/internal/parsers"
)

var _ parsers.ComponentPortParser = &GenericMultiPortReceiver{}

var (
	grpc = "grpc"
	http = "http"
)

// NewOTLPReceiverParser builds a new parser for OTLP receivers.
func NewOTLPReceiverParser(name string, config interface{}) (parsers.ComponentPortParser, error) {
	return createMultiPortParser(WithPortMapping(
		"grpc",
		4317,
		WithAppProtocol(&grpc),
		WithTargetPort(4317),
	), WithPortMapping(
		"http",
		4318,
		WithAppProtocol(&http),
		WithTargetPort(4318),
	))(name, config)
}
func init() {
	Register("otlp", NewOTLPReceiverParser)
}
