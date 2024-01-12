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

package receiver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLokiSelfRegisters(t *testing.T) {
	// verify
	assert.True(t, IsRegistered("loki"))
}

func TestLokiIsFoundByName(t *testing.T) {
	// test
	p, err := For(logger, "loki", map[interface{}]interface{}{})
	assert.NoError(t, err)

	// verify
	assert.Equal(t, "__loki", p.ParserName())
}

func TestLokiPortsOverridden(t *testing.T) {
	// prepare
	builder := NewLokiReceiverParser(logger, "loki", map[interface{}]interface{}{
		"protocols": map[interface{}]interface{}{
			"grpc": map[interface{}]interface{}{
				"endpoint": "0.0.0.0:1234",
			},
			"http": map[interface{}]interface{}{
				"endpoint": "0.0.0.0:1235",
			},
		},
	})

	expectedResults := map[string]struct {
		portNumber int32
		seen       bool
	}{
		"loki-grpc": {portNumber: 1234},
		"loki-http": {portNumber: 1235},
	}

	// test
	ports, err := builder.Ports()

	// verify
	assert.NoError(t, err)
	assert.Len(t, ports, len(expectedResults))

	for _, port := range ports {
		r := expectedResults[port.Name]
		r.seen = true
		expectedResults[port.Name] = r
		assert.EqualValues(t, r.portNumber, port.Port)
	}
	for k, v := range expectedResults {
		assert.True(t, v.seen, "the port %s wasn't included in the service ports", k)
	}
}

func TestLokiExposeDefaultPorts(t *testing.T) {
	// prepare
	builder := NewLokiReceiverParser(logger, "loki", map[interface{}]interface{}{
		"protocols": map[interface{}]interface{}{
			"grpc": map[interface{}]interface{}{},
			"http": map[interface{}]interface{}{},
		},
	})

	expectedResults := map[string]struct {
		portNumber int32
		seen       bool
	}{
		"loki-grpc": {portNumber: 9095},
		"loki-http": {portNumber: 3100},
	}

	// test
	ports, err := builder.Ports()

	// verify
	assert.NoError(t, err)
	assert.Len(t, ports, len(expectedResults))

	for _, port := range ports {
		r := expectedResults[port.Name]
		r.seen = true
		expectedResults[port.Name] = r
		assert.EqualValues(t, r.portNumber, port.Port)
	}
	for k, v := range expectedResults {
		assert.True(t, v.seen, "the port %s wasn't included in the service ports", k)
	}
}
