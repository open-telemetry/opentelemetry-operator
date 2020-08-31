package collector_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	. "github.com/open-telemetry/opentelemetry-operator/pkg/collector"
)

var _ = Describe("Labels", func() {
	It("should build a common set of labels", func() {
		// prepare
		otelcol := v1alpha1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-instance",
				Namespace: "my-ns",
			},
		}

		// test
		labels := Labels(otelcol)

		// verify
		Expect(labels).To(HaveLen(4))
		Expect(labels["app.kubernetes.io/managed-by"]).To(Equal("opentelemetry-operator"))
		Expect(labels["app.kubernetes.io/instance"]).To(Equal("my-ns.my-instance"))
		Expect(labels["app.kubernetes.io/part-of"]).To(Equal("opentelemetry"))
		Expect(labels["app.kubernetes.io/component"]).To(Equal("opentelemetry-collector"))
	})

	It("should propagate down the instance's labels", func() {
		// prepare
		otelcol := v1alpha1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"myapp": "mycomponent"},
			},
		}

		// test
		labels := Labels(otelcol)

		// verify
		Expect(labels).To(HaveLen(5))
		Expect(labels["myapp"]).To(Equal("mycomponent"))
	})
})
