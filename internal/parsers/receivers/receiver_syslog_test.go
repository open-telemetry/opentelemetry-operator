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
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestSyslogSelfRegisters(t *testing.T) {
	// verify
	assert.True(t, IsRegistered("syslog"))
}

func TestSyslogIsFoundByName(t *testing.T) {
	// test
	p, err := For("syslog", map[string]interface{}{})
	assert.ErrorContains(t, err, "must set either udp or tcp")
	good := map[string]interface{}{"udp": map[string]interface{}{"listen_address": "0.0.0.0:1234"}}
	p, err = For("syslog", good)
	assert.NoError(t, err)
	// verify
	assert.Equal(t, "__syslog", p.ParserName())
}

func TestSyslogConfiguration(t *testing.T) {
	for _, tt := range []struct {
		desc     string
		config   map[string]interface{}
		expected []corev1.ServicePort
	}{
		{"UDP port configuration",
			map[string]interface{}{"udp": map[string]interface{}{"listen_address": "0.0.0.0:1234"}},
			[]corev1.ServicePort{{Name: "syslog", Port: 1234, Protocol: corev1.ProtocolUDP}},
		},
		{"TCP port configuration",
			map[string]interface{}{"tcp": map[string]interface{}{"listen_address": "0.0.0.0:1234"}},
			[]corev1.ServicePort{{Name: "syslog", Port: 1234, Protocol: corev1.ProtocolTCP}},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			// prepare
			builder, err := NewSyslogReceiverParser("syslog", tt.config)
			assert.NoError(t, err)

			// test
			ports, err := builder.Ports(logr.Discard())

			// verify
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, ports)
		})
	}
}
