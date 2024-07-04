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

package exporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func Test_getService(t *testing.T) {
	lbcfgStr := `routing_key: "service"
protocol:
  otlp:
    timeout: 1s
resolver:
  # use k8s service resolver, if collector runs in kubernetes environment
  k8s:
    service: lb-svc.kube-public
    ports:
      - 15317
      - 16317
`
	var config = make(map[any]any)
	assert.NoError(t, yaml.Unmarshal([]byte(lbcfgStr), config))

	gotName, gotNamespace, gotOk := getService(config)
	assert.Equal(t, gotName, "lb-svc")
	assert.Equal(t, gotNamespace, "kube-public")
	assert.True(t, gotOk)

}
