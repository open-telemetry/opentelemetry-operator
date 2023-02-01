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
	"github.com/open-telemetry/opentelemetry-operator/pkg/platform"
)

func TestNewConfig(t *testing.T) {
	// prepare
	cfg := config.New(
		config.WithCollectorImage("some-image"),
		config.WithCollectorConfigMapEntry("some-config.yaml"),
		config.WithPlatform(platform.Kubernetes),
	)

	// test
	assert.Equal(t, "some-image", cfg.CollectorImage())
	assert.Equal(t, "some-config.yaml", cfg.CollectorConfigMapEntry())
	assert.Equal(t, platform.Kubernetes, cfg.Platform())
}

func TestOnPlatformChangeCallback(t *testing.T) {
	// prepare
	calledBack := false
	mock := &mockAutoDetect{
		PlatformFunc: func() (platform.Platform, error) {
			return platform.OpenShift, nil
		},
	}
	cfg := config.New(
		config.WithAutoDetect(mock),
		config.WithOnPlatformChangeCallback(func() error {
			calledBack = true
			return nil
		}),
	)

	// sanity check
	require.Equal(t, platform.Unknown, cfg.Platform())

	// test
	err := cfg.AutoDetect()
	require.NoError(t, err)

	// verify
	assert.Equal(t, platform.OpenShift, cfg.Platform())
	assert.True(t, calledBack)
}

func TestAutoDetectInBackground(t *testing.T) {
	// prepare
	wg := &sync.WaitGroup{}
	wg.Add(2)
	mock := &mockAutoDetect{
		PlatformFunc: func() (platform.Platform, error) {
			wg.Done()
			// returning Unknown will cause the auto-detection to keep trying to detect the platform
			return platform.Unknown, nil
		},
	}
	cfg := config.New(
		config.WithAutoDetect(mock),
		config.WithAutoDetectFrequency(100*time.Millisecond),
	)

	// sanity check
	require.Equal(t, platform.Unknown, cfg.Platform())

	// test
	err := cfg.StartAutoDetect()
	require.NoError(t, err)

	// verify
	wg.Wait()
}

var _ autodetect.AutoDetect = (*mockAutoDetect)(nil)

type mockAutoDetect struct {
	PlatformFunc func() (platform.Platform, error)
}

func (m *mockAutoDetect) HPAVersion() (autodetect.AutoscalingVersion, error) {
	return autodetect.DefaultAutoscalingVersion, nil
}

func (m *mockAutoDetect) Platform() (platform.Platform, error) {
	if m.PlatformFunc != nil {
		return m.PlatformFunc()
	}
	return platform.Unknown, nil
}
