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

func TestSkywalkingSelfRegisters(t *testing.T) {
	// verify
	assert.True(t, IsRegistered("skywalking"))
}

func TestSkywalkingIsFoundByName(t *testing.T) {
	// test
	p, err := For(logger, "skywalking", map[string]interface{}{})
	assert.NoError(t, err)

	// verify
	assert.Equal(t, "__skywalking", p.ParserName())
}

func TestSkywalkingPortsOverridden(t *testing.T) {
	// prepare
	builder := NewSkywalkingReceiverParser(logger, "skywalking", map[string]interface{}{
		"protocols": map[string]interface{}{
			"grpc": map[string]interface{}{
				"endpoint": "0.0.0.0:1234",
			},
			"http": map[string]interface{}{
				"endpoint": "0.0.0.0:1235",
			},
		},
	})

	expectedResults := map[string]struct {
		portNumber int32
		seen       bool
	}{
		"skywalking-grpc": {portNumber: 1234},
		"skywalking-http": {portNumber: 1235},
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

func TestSkywalkingExposeDefaultPorts(t *testing.T) {
	// prepare
	builder := NewSkywalkingReceiverParser(logger, "skywalking", map[string]interface{}{
		"protocols": map[string]interface{}{
			"grpc": map[string]interface{}{},
			"http": map[string]interface{}{},
		},
	})

	expectedResults := map[string]struct {
		portNumber int32
		seen       bool
	}{
		"skywalking-grpc": {portNumber: 11800},
		"skywalking-http": {portNumber: 12800},
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
