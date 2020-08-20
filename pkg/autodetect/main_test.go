package autodetect_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/open-telemetry/opentelemetry-operator/pkg/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/pkg/platform"
)

var _ = Describe("Autodetect", func() {
	DescribeTable("detect platform based on available API groups",
		func(expected platform.Platform, apiGroupList *metav1.APIGroupList) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				output, err := json.Marshal(apiGroupList)
				Expect(err).ToNot(HaveOccurred())

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write(output)
			}))
			defer server.Close()

			autoDetect, err := autodetect.New(&rest.Config{Host: server.URL})
			Expect(err).ToNot(HaveOccurred())

			// test
			plt, err := autoDetect.Platform()

			// verify
			Expect(err).ToNot(HaveOccurred())
			Expect(plt).To(Equal(expected))
		},

		Entry("kubernetes", platform.Kubernetes, &metav1.APIGroupList{}),
		Entry("openshift", platform.OpenShift, &metav1.APIGroupList{
			Groups: []metav1.APIGroup{
				{
					Name: "route.openshift.io",
				},
			},
		}),
	)

	It("should return unknown platform when errors occur", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		autoDetect, err := autodetect.New(&rest.Config{Host: server.URL})
		Expect(err).ToNot(HaveOccurred())

		// test
		plt, err := autoDetect.Platform()

		// verify
		Expect(err).To(HaveOccurred())
		Expect(plt).To(Equal(platform.Unknown))
	})
})
