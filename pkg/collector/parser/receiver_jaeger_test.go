package parser_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/parser"
)

var _ = Describe("Jaeger receiver", func() {
	logger := logf.Log.WithName("unit-tests")

	It("should have self registered", func() {
		// verify
		Expect(parser.IsRegistered("jaeger")).To(BeTrue())
	})

	It("should be found via its parser name", func() {
		// test
		p := parser.For(logger, "jaeger", map[interface{}]interface{}{})

		// verify
		Expect(p.ParserName()).To(Equal("__jaeger"))
	})

	It("should build with a minimal configuration", func() {
		// prepare
		builder := parser.NewJaegerReceiverParser(logger, "jaeger", map[interface{}]interface{}{
			"protocols": map[interface{}]interface{}{
				"grpc": map[interface{}]interface{}{},
			},
		})

		// test
		ports, err := builder.Ports()

		// verify
		Expect(err).ToNot(HaveOccurred())
		Expect(ports).To(HaveLen(1))
		Expect(ports[0].Port).To(BeEquivalentTo(14250))
	})

	It("should allow ports to be overridden", func() {
		// prepare
		builder := parser.NewJaegerReceiverParser(logger, "jaeger", map[interface{}]interface{}{
			"protocols": map[interface{}]interface{}{
				"grpc": map[interface{}]interface{}{
					"endpoint": "0.0.0.0:1234",
				},
			},
		})

		// test
		ports, err := builder.Ports()

		// verify
		Expect(err).ToNot(HaveOccurred())
		Expect(ports).To(HaveLen(1))
		Expect(ports[0].Port).To(BeEquivalentTo(1234))
	})

	It("should expose the default ports", func() {
		// prepare
		builder := parser.NewJaegerReceiverParser(logger, "jaeger", map[interface{}]interface{}{
			"protocols": map[interface{}]interface{}{
				"grpc":           map[interface{}]interface{}{},
				"thrift_http":    map[interface{}]interface{}{},
				"thrift_compact": map[interface{}]interface{}{},
				"thrift_binary":  map[interface{}]interface{}{},
			},
		})

		expectedResults := map[string]struct {
			portNumber int32
			seen       bool
		}{
			"jaeger-grpc":           {portNumber: 14250},
			"jaeger-thrift-http":    {portNumber: 14268},
			"jaeger-thrift-compact": {portNumber: 6831},
			"jaeger-thrift-binary":  {portNumber: 6832},
		}

		// test
		ports, err := builder.Ports()

		// verify
		Expect(err).ToNot(HaveOccurred())
		Expect(ports).To(HaveLen(4))

		for _, port := range ports {
			r := expectedResults[port.Name]
			r.seen = true
			expectedResults[port.Name] = r
			Expect(port.Port).To(BeEquivalentTo(r.portNumber))
		}
		for k, v := range expectedResults {
			Expect(v.seen).To(BeTrue(), "the port %s wasn't included in the service ports", k)
		}
	})
})
