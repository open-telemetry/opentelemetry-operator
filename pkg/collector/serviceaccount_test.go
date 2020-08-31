package collector_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	. "github.com/open-telemetry/opentelemetry-operator/pkg/collector"
)

var _ = Describe("Serviceaccount", func() {
	It("should default to an instance-specific service account name", func() {
		// prepare
		otelcol := v1alpha1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-instance",
			},
		}

		// test
		sa := ServiceAccountName(otelcol)

		// verify
		Expect(sa).To(Equal("my-instance-collector"))
	})

	It("should allow the serviceaccount to be overridden", func() {
		// prepare
		otelcol := v1alpha1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-instance",
			},
			Spec: v1alpha1.OpenTelemetryCollectorSpec{
				ServiceAccount: "my-special-sa",
			},
		}

		// test
		sa := ServiceAccountName(otelcol)

		// verify
		Expect(sa).To(Equal("my-special-sa"))
	})

	It("should build a new default service account", func() {
		// prepare
		otelcol := v1alpha1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-instance",
			},
		}

		// test
		sa := ServiceAccount(otelcol)

		// verify
		Expect(sa.Name).To(Equal("my-instance-collector"))
	})
})
