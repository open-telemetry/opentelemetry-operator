package collector_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	. "github.com/open-telemetry/opentelemetry-operator/pkg/collector"
)

var _ = Describe("Daemonset", func() {
	logger := logf.Log.WithName("unit-tests")

	It("should build a default new daemonset", func() {
		// prepare
		otelcol := v1alpha1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-instance",
			},
		}
		cfg := config.New()

		// test
		d := DaemonSet(cfg, logger, otelcol)

		// verify
		Expect(d.Name).To(Equal("my-instance-collector"))
		Expect(d.Labels["app.kubernetes.io/name"]).To(Equal("my-instance-collector"))
		Expect(d.Annotations["prometheus.io/scrape"]).To(Equal("true"))
		Expect(d.Annotations["prometheus.io/port"]).To(Equal("8888"))
		Expect(d.Annotations["prometheus.io/path"]).To(Equal("/metrics"))

		Expect(d.Spec.Template.Spec.Containers).To(HaveLen(1))

		// none of the default annotations should propagate down to the pod
		Expect(d.Spec.Template.Annotations).To(BeEmpty())

		// the pod selector should match the pod spec's labels
		Expect(d.Spec.Template.Labels).To(Equal(d.Spec.Selector.MatchLabels))
	})
})
