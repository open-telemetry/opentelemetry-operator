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

package config_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/autodetect"
)

func TestNewConfig(t *testing.T) {
	// prepare
	cfg := config.New(
		config.WithCollectorImage("some-image"),
		config.WithCollectorConfigMapEntry("some-config.yaml"),
		config.WithPlatform(autodetect.OpenShiftRoutesNotAvailable),
	)

	// test
	assert.Equal(t, "some-image", cfg.CollectorImage())
	assert.Equal(t, "some-config.yaml", cfg.CollectorConfigMapEntry())
	assert.Equal(t, autodetect.OpenShiftRoutesNotAvailable, cfg.OpenShiftRoutes())
	assert.Equal(t, autodetect.AutoscalingVersionUnknown, cfg.AutoscalingVersion())
}

func TestOnPlatformChangeCallback(t *testing.T) {
	// prepare
	calledBack := false
	mock := &mockAutoDetect{
		OpenShiftRoutesAvailabilityFunc: func() (autodetect.OpenShiftRoutesAvailability, error) {
			return autodetect.OpenShiftRoutesAvailable, nil
		},
	}
	cfg := config.New(
		config.WithAutoDetect(mock),
		config.WithOnOpenShiftRoutesChangeCallback(func() error {
			calledBack = true
			return nil
		}),
	)

	// sanity check
	require.Equal(t, autodetect.OpenShiftRoutesNotAvailable, cfg.OpenShiftRoutes())

	// test
	err := cfg.AutoDetect()
	require.NoError(t, err)

	// verify
	assert.Equal(t, autodetect.OpenShiftRoutesAvailable, cfg.OpenShiftRoutes())
	assert.True(t, calledBack)
}

func TestAutoDetectInBackground(t *testing.T) {
	// prepare
	wg := &sync.WaitGroup{}
	wg.Add(2)
	mock := &mockAutoDetect{
		OpenShiftRoutesAvailabilityFunc: func() (autodetect.OpenShiftRoutesAvailability, error) {
			wg.Done()
			return autodetect.OpenShiftRoutesNotAvailable, nil
		},
		HPAVersionFunc: func() (autodetect.AutoscalingVersion, error) {
			wg.Done()
			return autodetect.AutoscalingVersionV2, nil
		},
	}
	cfg := config.New(
		config.WithAutoDetect(mock),
		config.WithAutoDetectFrequency(100*time.Millisecond),
	)

	// sanity check
	require.Equal(t, autodetect.OpenShiftRoutesNotAvailable, cfg.OpenShiftRoutes())
	require.Equal(t, autodetect.AutoscalingVersionUnknown, cfg.AutoscalingVersion())

	// test
	err := cfg.StartAutoDetect()
	require.NoError(t, err)

	// verify
	wg.Wait()
}

var _ autodetect.AutoDetect = (*mockAutoDetect)(nil)

type mockAutoDetect struct {
	OpenShiftRoutesAvailabilityFunc func() (autodetect.OpenShiftRoutesAvailability, error)
	HPAVersionFunc                  func() (autodetect.AutoscalingVersion, error)
}

func (m *mockAutoDetect) HPAVersion() (autodetect.AutoscalingVersion, error) {
	if m.HPAVersionFunc != nil {
		return m.HPAVersionFunc()
	}
	return autodetect.DefaultAutoscalingVersion, nil
}

func (m *mockAutoDetect) OpenShiftRoutesAvailability() (autodetect.OpenShiftRoutesAvailability, error) {
	if m.OpenShiftRoutesAvailabilityFunc != nil {
		return m.OpenShiftRoutesAvailabilityFunc()
	}
	return autodetect.OpenShiftRoutesNotAvailable, nil
}
