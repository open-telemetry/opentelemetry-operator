package collector_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	. "github.com/open-telemetry/opentelemetry-operator/pkg/collector"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
)

var _ = Describe("Volume", func() {
	It("should build a new default volume", func() {
		// prepare
		otelcol := v1alpha1.OpenTelemetryCollector{}
		cfg := config.New()

		// test
		volumes := Volumes(cfg, otelcol)

		// verify
		Expect(volumes).To(HaveLen(1))

		// check that it's the otc-internal volume, with the config map
		Expect(volumes[0].Name).To(Equal(naming.ConfigMapVolume()))
	})

	It("should allow more volumes to be added", func() {
		// prepare
		otelcol := v1alpha1.OpenTelemetryCollector{
			Spec: v1alpha1.OpenTelemetryCollectorSpec{
				Volumes: []corev1.Volume{{
					Name: "my-volume",
				}},
			},
		}
		cfg := config.New()

		// test
		volumes := Volumes(cfg, otelcol)

		// verify
		Expect(volumes).To(HaveLen(2))

		// check that it's the otc-internal volume, with the config map
		Expect(volumes[1].Name).To(Equal("my-volume"))
	})

})
