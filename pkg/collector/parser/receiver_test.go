package parser

import (
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var _ = Describe("Receiver", func() {
	logger := logf.Log.WithName("unit-tests")

	DescribeTable("port names",
		func(candidate string, port int, expected string) {
			// test and verify
			Expect(portName(candidate, int32(port))).To(Equal(expected))
		},
		Entry("regular case", "my-receiver", 123, "my-receiver"),
		Entry("name too long", "long-name-long-name-long-name-long-name-long-name-long-name-long-name-long-name", 123, "port-123"),
		Entry("name with invalid chars", "my-🦄-receiver", 123, "port-123"),
		Entry("name starting with invalid char", "-my-receiver", 123, "port-123"),
	)

	DescribeTable("receiver type",
		func(name string, expected string) {
			// test and verify
			Expect(receiverType(name)).To(Equal(expected))
		},
		Entry("regular case", "myreceiver", "myreceiver"),
		Entry("named instance", "myreceiver/custom", "myreceiver"),
	)

	DescribeTable("parse port from endpoint",
		func(endpoint string, expected int, errorExpected bool) {
			// test
			val, err := portFromEndpoint(endpoint)
			if errorExpected {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
			}

			Expect(val).To(BeEquivalentTo(expected), "wrong port from endpoint %s: %d", endpoint, val)
		},
		Entry("regular case", "http://localhost:1234", 1234, false),
		Entry("no protocol", "0.0.0.0:1234", 1234, false),
		Entry("just port", ":1234", 1234, false),
		Entry("no port at all", "http://localhost", 0, true),
	)

	It("should fail when the port isn't a string", func() {
		// prepare
		config := map[interface{}]interface{}{
			"endpoint": 123,
		}

		// test
		p := singlePortFromConfigEndpoint(logger, "myreceiver", config)

		// verify
		Expect(p).To(BeNil())
	})

	It("should fallback to generic parser when receiver isn't registered", func() {
		// test
		p := For(logger, "myreceiver", map[interface{}]interface{}{})

		// test
		Expect(p.ParserName()).To(Equal("__generic"))
	})

	It("should find a registered parser", func() {
		// prepare
		builderCalled := false
		Register("mock", func(logger logr.Logger, name string, config map[interface{}]interface{}) ReceiverParser {
			builderCalled = true
			return &mockParser{}
		})

		// test
		For(logger, "mock", map[interface{}]interface{}{})

		// verify
		Expect(builderCalled).To(BeTrue())
	})
})

type mockParser struct {
}

func (m *mockParser) Ports() ([]corev1.ServicePort, error) {
	return nil, nil
}

func (m *mockParser) ParserName() string {
	return "__mock"
}
