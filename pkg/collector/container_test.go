package collector_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	. "github.com/open-telemetry/opentelemetry-operator/pkg/collector"
)

var _ = Describe("Container", func() {
	logger := logf.Log.WithName("unit-tests")

	It("should generate a new default container", func() {
		// prepare
		otelcol := v1alpha1.OpenTelemetryCollector{}
		cfg := config.New(config.WithCollectorImage("default-image"))

		// test
		c := Container(cfg, logger, otelcol)

		// verify
		Expect(c.Image).To(Equal("default-image"))
	})

	It("should allow image to be overridden", func() {
		// prepare
		otelcol := v1alpha1.OpenTelemetryCollector{
			Spec: v1alpha1.OpenTelemetryCollectorSpec{
				Image: "overridden-image",
			},
		}
		cfg := config.New(config.WithCollectorImage("default-image"))

		// test
		c := Container(cfg, logger, otelcol)

		// verify
		Expect(c.Image).To(Equal("overridden-image"))
	})

	It("config flag is ignored", func() {
		// prepare
		otelcol := v1alpha1.OpenTelemetryCollector{
			Spec: v1alpha1.OpenTelemetryCollectorSpec{
				Args: map[string]string{
					"key":    "value",
					"config": "/some-custom-file.yaml",
				},
			},
		}
		cfg := config.New()

		// test
		c := Container(cfg, logger, otelcol)

		// verify
		Expect(c.Args).To(HaveLen(2))
		Expect(c.Args).To(ContainElement("--key=value")) // sanity check
		Expect(c.Args).ToNot(ContainElement("--config=/some-custom-file.yaml"))
	})

	It("custom volumes are mounted", func() {
		// prepare
		otelcol := v1alpha1.OpenTelemetryCollector{
			Spec: v1alpha1.OpenTelemetryCollectorSpec{
				VolumeMounts: []corev1.VolumeMount{{
					Name: "custom-volume-mount",
				}},
			},
		}
		cfg := config.New()

		// test
		c := Container(cfg, logger, otelcol)

		// verify
		Expect(c.VolumeMounts).To(HaveLen(2))
		Expect(c.VolumeMounts[1].Name).To(Equal("custom-volume-mount"))
	})

	It("should allow for env vars to be overridden", func() {
		otelcol := v1alpha1.OpenTelemetryCollector{
			Spec: v1alpha1.OpenTelemetryCollectorSpec{
				Env: []corev1.EnvVar{
					{
						Name:  "foo",
						Value: "bar",
					},
				},
			},
		}

		cfg := config.New()

		// test
		c := Container(cfg, logger, otelcol)

		Expect(c.Env).To(HaveLen(1))
		Expect(c.Env[0].Name).To(Equal("foo"))
		Expect(c.Env[0].Value).To(Equal("bar"))
	})

	It("should allow for empty env vars by default", func() {
		otelcol := v1alpha1.OpenTelemetryCollector{
			Spec: v1alpha1.OpenTelemetryCollectorSpec{},
		}

		cfg := config.New()

		// test
		c := Container(cfg, logger, otelcol)

		Expect(c.Env).To(BeEmpty())
	})
})
