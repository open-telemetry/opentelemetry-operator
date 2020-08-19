package upgrade_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/upgrade"
)

var _ = Describe("Upgrade", func() {
	logger := logf.Log.WithName("unit-tests")

	It("should upgrade to the latest", func() {
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
		}
		existing.Status.Version = "0.0.1" // this is the first version we have an upgrade function
		err := k8sClient.Create(context.Background(), &existing)
		Expect(err).To(Succeed())

		err = k8sClient.Status().Update(context.Background(), &existing)
		Expect(err).To(Succeed())

		// sanity check
		persisted := &v1alpha1.OpenTelemetryCollector{}
		err = k8sClient.Get(context.Background(), nsn, persisted)
		Expect(err).To(Succeed())
		Expect(persisted.Status.Version).To(Equal("0.0.1"))

		// test
		err = upgrade.ManagedInstances(context.Background(), logger, k8sClient)
		Expect(err).To(Succeed())

		// verify
		err = k8sClient.Get(context.Background(), nsn, persisted)
		Expect(err).To(Succeed())
		Expect(persisted.Status.Version).To(Equal(upgrade.Latest.Tag))

		// cleanup
		Expect(k8sClient.Delete(context.Background(), &existing))
	})
})
