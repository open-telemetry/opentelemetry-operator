package parser_test

import (
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/parser"
)

var _ = Describe("Generic receivers", func() {
	logger := logf.Log.WithName("unit-tests")

	It("should parse the endpoint", func() {
		// prepare
		// there's no parser registered to handle "myreceiver", so, it falls back to the generic parser
		builder := parser.NewGenericReceiverParser(logger, "myreceiver", map[interface{}]interface{}{
			"endpoint": "0.0.0.0:1234",
		})

		// test
		ports, err := builder.Ports()

		// verify
		Expect(err).ToNot(HaveOccurred())
		Expect(ports).To(HaveLen(1))
		Expect(ports[0].Port).To(BeEquivalentTo(1234))
	})

	It("should have failed to parse the endpoint", func() {
		// prepare
		// there's no parser registered to handle "myreceiver", so, it falls back to the generic parser
		builder := parser.NewGenericReceiverParser(logger, "myreceiver", map[interface{}]interface{}{
			"endpoint": "0.0.0.0",
		})

		// test
		ports, err := builder.Ports()

		// verify
		Expect(err).ToNot(HaveOccurred())
		Expect(ports).To(HaveLen(0))
	})

	DescribeTable("downstream parsers",
		func(receiverName, parserName string, defaultPort int, builder func(logr.Logger, string, map[interface{}]interface{}) parser.ReceiverParser) {
			Describe("builds successfully", func() {
				// test
				builder := builder(logger, receiverName, map[interface{}]interface{}{})

				// verify
				Expect(builder.ParserName()).To(Equal(parserName))
			})

			Describe("assigns the expected port", func() {
				// prepare
				builder := builder(logger, receiverName, map[interface{}]interface{}{})

				// test
				ports, err := builder.Ports()

				// verify
				Expect(err).ToNot(HaveOccurred())
				Expect(ports).To(HaveLen(1))
				Expect(ports[0].Port).To(BeEquivalentTo(defaultPort))
				Expect(ports[0].Name).To(Equal(receiverName))
			})

			Describe("allows port to be overridden", func() {
				// prepare
				builder := builder(logger, receiverName, map[interface{}]interface{}{
					"endpoint": "0.0.0.0:65535",
				})

				// test
				ports, err := builder.Ports()

				// verify
				Expect(err).ToNot(HaveOccurred())
				Expect(ports).To(HaveLen(1))
				Expect(ports[0].Port).To(BeEquivalentTo(65535))
				Expect(ports[0].Name).To(Equal(receiverName))
			})
		},
		Entry("zipkin", "zipkin", "__zipkin", 9411, parser.NewZipkinReceiverParser),
		Entry("opencensus", "opencensus", "__opencensus", 55678, parser.NewOpenCensusReceiverParser),
		Entry("otlp", "otlp", "__otlp", 55680, parser.NewOTLPReceiverParser),

		// contrib receivers
		Entry("carbon", "carbon", "__carbon", 2003, parser.NewCarbonReceiverParser),
		Entry("collectd", "collectd", "__collectd", 8081, parser.NewCollectdReceiverParser),
		Entry("sapm", "sapm", "__sapm", 7276, parser.NewSAPMReceiverParser),
		Entry("signalfx", "signalfx", "__signalfx", 9943, parser.NewSignalFxReceiverParser),
		Entry("wavefront", "wavefront", "__wavefront", 2003, parser.NewWavefrontReceiverParser),
		Entry("zipkin-scribe", "zipkin-scribe", "__zipkinscribe", 9410, parser.NewZipkinScribeReceiverParser),
	)

})
