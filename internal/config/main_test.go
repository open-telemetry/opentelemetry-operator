package config_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
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

func TestOverrideVersion(t *testing.T) {
	// prepare
	v := version.Version{
		OpenTelemetryCollector: "the-version",
	}
	cfg := config.New(config.WithVersion(v))

	// test
	assert.Contains(t, cfg.CollectorImage(), "the-version")
}

func TestCallbackOnChanges(t *testing.T) {
	// prepare
	calledBack := false
	mock := &mockAutoDetect{
		PlatformFunc: func() (platform.Platform, error) {
			return platform.OpenShift, nil
		},
	}
	cfg := config.New(
		config.WithAutoDetect(mock),
		config.WithOnChange(func() error {
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

func TestDistributionFound(t *testing.T) {
	// prepare
	cfg := config.New()
	expected := v1alpha1.OpenTelemetryCollectorDistribution{
		ObjectMeta: v1.ObjectMeta{Namespace: "my-ns", Name: "my-dist"},
		Command:    []string{"some-custom-command"},
		Image:      "some-custom-image",
	}
	cfg.SetDistributions([]v1alpha1.OpenTelemetryCollectorDistribution{expected})

	// test
	dist := cfg.Distribution("my-ns", "my-dist")

	// verify
	assert.Equal(t, expected, *dist)
}

func TestDistributionNotFound(t *testing.T) {
	// prepare
	cfg := config.New()

	// test
	dist := cfg.Distribution("my-ns", "my-dist")

	// verify
	assert.Nil(t, dist)
}

var _ autodetect.AutoDetect = (*mockAutoDetect)(nil)

type mockAutoDetect struct {
	PlatformFunc func() (platform.Platform, error)
}

func (m *mockAutoDetect) Platform() (platform.Platform, error) {
	if m.PlatformFunc != nil {
		return m.PlatformFunc()
	}
	return platform.Unknown, nil
}
