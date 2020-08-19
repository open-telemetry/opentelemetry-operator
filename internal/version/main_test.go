package version

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Version management", func() {
	It("should have a fallback version", func() {
		Expect(OpenTelemetryCollector()).To(Equal("0.0.0"))
	})

	It("should use a version set during the build", func() {
		// prepare
		otelCol = "0.0.2" // set during the build
		defer func() {
			otelCol = ""
		}()

		Expect(OpenTelemetryCollector()).To(Equal(otelCol))
		Expect(Get()).To(ContainSubstring(otelCol))
	})
})
