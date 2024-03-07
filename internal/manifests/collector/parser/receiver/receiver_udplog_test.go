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
	corev1 "k8s.io/api/core/v1"
)

func TestUdpLogSelfRegisters(t *testing.T) {
	// verify
	assert.True(t, IsRegistered("udplog"))
}

func TestUdpLogIsFoundByName(t *testing.T) {
	// test
	p, err := For(logger, "udplog", map[string]interface{}{})
	assert.NoError(t, err)

	// verify
	assert.Equal(t, "__udplog", p.ParserName())
}

func TestUdpLogConfiguration(t *testing.T) {
	for _, tt := range []struct {
		desc     string
		config   map[string]interface{}
		expected []corev1.ServicePort
	}{
		{"Empty configuration", map[string]interface{}{}, []corev1.ServicePort{}},
		{"UDP port configuration",
			map[string]interface{}{"listen_address": "0.0.0.0:1234"},
			[]corev1.ServicePort{{Name: "udplog", Port: 1234, Protocol: corev1.ProtocolUDP}}},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			// prepare
			builder := NewUdpLogReceiverParser(logger, "udplog", tt.config)

			// test
			ports, err := builder.Ports()

			// verify
			assert.NoError(t, err)
			assert.Equal(t, ports, tt.expected)
		})
	}
}
