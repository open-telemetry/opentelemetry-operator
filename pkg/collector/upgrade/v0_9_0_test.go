package upgrade_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/upgrade"
)

var _ = Describe("Upgrade to v0.9.0", func() {
	logger := logf.Log.WithName("unit-tests")

	It("should remove reconnection_delay", func() {
		// prepare
		nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
		existing := v1alpha1.OpenTelemetryCollector{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nsn.Name,
				Namespace: nsn.Namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "opentelemetry-operator",
				},
			},
			Spec: v1alpha1.OpenTelemetryCollectorSpec{
				Config: `exporters:
  opencensus:
    compression: "on"
    reconnection_delay: 15
    num_workers: 123`,
			},
		}
		existing.Status.Version = "0.8.0"

		// sanity check
		Expect(existing.Spec.Config).To(ContainSubstring("reconnection_delay"))

		// test
		res, err := upgrade.ManagedInstance(context.Background(), logger, version.Get(), nil, existing)
		Expect(err).To(Succeed())

		// verify
		Expect(res.Spec.Config).To(ContainSubstring("opencensus:"))
		Expect(res.Spec.Config).To(ContainSubstring(`compression: "on"`))
		Expect(res.Spec.Config).ToNot(ContainSubstring("reconnection_delay"))
		Expect(res.Spec.Config).To(ContainSubstring("num_workers: 123"))
		Expect(res.Status.Messages[0]).To(ContainSubstring("upgrade to v0.9.0 removed the property reconnection_delay for exporter"))
	})
})
