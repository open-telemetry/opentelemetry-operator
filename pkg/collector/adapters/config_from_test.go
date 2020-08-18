package adapters_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
)

var _ = Describe("ConfigFromString", func() {
	Describe("Invalid YAML", func() {
		It("should return an error", func() {
			// test
			config, err := adapters.ConfigFromString("🦄")

			// verify
			Expect(config).To(BeNil())
			Expect(err).To(MatchError(adapters.ErrInvalidYAML))
		})
	})

	Describe("Empty string", func() {
		It("should return an empty config", func() {
			// test and verify
			Expect(adapters.ConfigFromString("")).To(BeEmpty())
		})
	})
})
