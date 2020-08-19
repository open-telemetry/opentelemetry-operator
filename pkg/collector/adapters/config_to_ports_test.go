package adapters_test

import (
	"errors"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/parser"
)

var _ = Describe("ConfigToReceiverPorts", func() {
	logger := logf.Log.WithName("unit-tests")

	Describe("Extract ports from config", func() {
		It("extract all known ports", func() {
			configStr := `receivers:
  examplereceiver:
    endpoint: "0.0.0.0:12345"
  examplereceiver/settings:
    endpoint: "0.0.0.0:12346"
  examplereceiver/invalid-ignored:
    endpoint: "0.0.0.0"
  examplereceiver/invalid-not-number:
    endpoint: "0.0.0.0:not-number"
  examplereceiver/without-endpoint:
    notendpoint: "0.0.0.0:12347"
  jaeger:
    protocols:
      grpc:
      thrift_compact:
      thrift_binary:
        endpoint: 0.0.0.0:6833
  jaeger/custom:
    protocols:
      thrift_http:
        endpoint: 0.0.0.0:15268
`

			// prepare
			config, err := adapters.ConfigFromString(configStr)
			Expect(config).ToNot(BeEmpty())
			Expect(err).ToNot(HaveOccurred())

			// test
			ports, err := adapters.ConfigToReceiverPorts(logger, config)
			Expect(ports).To(HaveLen(6))
			Expect(err).ToNot(HaveOccurred())

			// verify
			expectedPorts := map[int32]bool{}
			expectedPorts[int32(12345)] = false
			expectedPorts[int32(12346)] = false
			expectedPorts[int32(14250)] = false
			expectedPorts[int32(6831)] = false
			expectedPorts[int32(6833)] = false
			expectedPorts[int32(15268)] = false

			expectedNames := map[string]bool{}
			expectedNames["examplereceiver"] = false
			expectedNames["examplereceiver-settings"] = false
			expectedNames["jaeger-grpc"] = false
			expectedNames["jaeger-thrift-compact"] = false
			expectedNames["jaeger-thrift-binary"] = false
			expectedNames["jaeger-custom-thrift-http"] = false

			// make sure we only have the ports in the set
			for _, port := range ports {
				Expect(expectedPorts).To(HaveKey(port.Port))
				Expect(expectedNames).To(HaveKey(port.Name))
				expectedPorts[port.Port] = true
				expectedNames[port.Name] = true
			}

			// and make sure all the ports from the set are there
			for _, val := range expectedPorts {
				Expect(val).To(BeTrue())
			}

		})
	})

	DescribeTable("No ports are parsed",
		func(configStr string, expected error) {
			// prepare
			config, err := adapters.ConfigFromString(configStr)
			Expect(err).ToNot(HaveOccurred())

			// test
			ports, err := adapters.ConfigToReceiverPorts(logger, config)

			// verify
			Expect(ports).To(BeNil())
			Expect(err).To(MatchError(expected))
		},
		Entry("empty", "", adapters.ErrNoReceivers),
		Entry("not a map", "receivers: some-string", adapters.ErrReceiversNotAMap),
	)

	DescribeTable("Invalid receivers",
		func(configStr string) {
			// prepare
			config, err := adapters.ConfigFromString(configStr)
			Expect(err).ToNot(HaveOccurred())

			// test
			ports, err := adapters.ConfigToReceiverPorts(logger, config)

			// verify
			Expect(ports).To(HaveLen(0))
			Expect(err).ToNot(HaveOccurred())
		},
		Entry("receiver isn't a map", "receivers:\n  some-receiver: string"),
		Entry("receiver's endpoint isn't string", "receivers:\n  some-receiver:\n    endpoint: 123"),
	)

	Describe("Parser failed", func() {
		It("should return an empty list of ports", func() {
			// prepare
			mockParserCalled := false
			mockParser := &mockParser{
				portsFunc: func() ([]v1.ServicePort, error) {
					mockParserCalled = true
					return nil, errors.New("mocked error")
				},
			}
			parser.Register("mock", func(logger logr.Logger, name string, config map[interface{}]interface{}) parser.ReceiverParser {
				return mockParser
			})

			config := map[interface{}]interface{}{
				"receivers": map[interface{}]interface{}{
					"mock": map[interface{}]interface{}{},
				},
			}

			// test
			ports, err := adapters.ConfigToReceiverPorts(logger, config)

			// verify
			Expect(ports).To(HaveLen(0))
			Expect(err).ToNot(HaveOccurred())
			Expect(mockParserCalled).To(BeTrue())
		})
	})

})

type mockParser struct {
	portsFunc func() ([]corev1.ServicePort, error)
}

func (m *mockParser) Ports() ([]corev1.ServicePort, error) {
	if m.portsFunc != nil {
		return m.portsFunc()
	}

	return nil, nil
}

func (m *mockParser) ParserName() string {
	return "__mock-adapters"
}
