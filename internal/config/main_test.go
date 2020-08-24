package config_test

import (
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	"github.com/open-telemetry/opentelemetry-operator/pkg/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/pkg/platform"
)

var _ = Describe("Config", func() {

	It("should build new configuration with given options", func() {
		// prepare
		cfg := config.New(
			config.WithCollectorImage("some-image"),
			config.WithCollectorConfigMapEntry("some-config.yaml"),
			config.WithPlatform(platform.Kubernetes),
		)

		// test
		Expect(cfg.CollectorImage()).To(Equal("some-image"))
		Expect(cfg.CollectorConfigMapEntry()).To(Equal("some-config.yaml"))
		Expect(cfg.Platform()).To(Equal(platform.Kubernetes))
	})

	It("should use the version as part of the default image", func() {
		// prepare
		v := version.Version{
			OpenTelemetryCollector: "the-version",
		}
		cfg := config.New(config.WithVersion(v))

		// test
		Expect(cfg.CollectorImage()).To(ContainSubstring("the-version"))
	})

	It("should callback when configuration changes happen", func() {
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
		Expect(cfg.Platform()).To(Equal(platform.Unknown))

		// test
		cfg.AutoDetect()

		// verify
		Expect(cfg.Platform()).To(Equal(platform.OpenShift))
		Expect(calledBack).To(BeTrue())
	})

	It("should run the auto-detect routine in the background", func() {
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
		Expect(cfg.Platform()).To(Equal(platform.Unknown))

		// test
		cfg.StartAutoDetect()

		// verify
		wg.Wait()
	})
})

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
