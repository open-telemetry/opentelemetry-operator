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

package opampbridge

import (
	"testing"

	"github.com/stretchr/testify/assert"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

var logger = logf.Log.WithName("unit-tests")

func TestContainerNewDefault(t *testing.T) {
	// prepare
	opampBridge := v1alpha1.OpAMPBridge{}
	cfg := config.New(config.WithOperatorOpAMPBridgeImage("default-image"))

	// test
	c := Container(cfg, logger, opampBridge)

	// verify
	assert.Equal(t, "default-image", c.Image)
}

func TestContainerWithImageOverridden(t *testing.T) {
	// prepare
	otelcol := v1alpha1.OpAMPBridge{
		Spec: v1alpha1.OpAMPBridgeSpec{
			Image: "overridden-image",
		},
	}

	cfg := config.New(config.WithOperatorOpAMPBridgeImage("default-image"))

	// test
	c := Container(cfg, logger, otelcol)

	// verify
	assert.Equal(t, "overridden-image", c.Image)
}

func TestContainerVolumes(t *testing.T) {
	// prepare
	opampBridge := v1alpha1.OpAMPBridge{
		Spec: v1alpha1.OpAMPBridgeSpec{
			Image: "default-image",
		},
	}
	cfg := config.New()

	// test
	c := Container(cfg, logger, opampBridge)

	// verify
	assert.Len(t, c.VolumeMounts, 1)
	assert.Equal(t, naming.OpAMPBridgeConfigMapVolume(), c.VolumeMounts[0].Name)
}
