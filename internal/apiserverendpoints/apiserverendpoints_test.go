// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package apiserverendpoints

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func endpointSlice(name, namespace, service string, addressType discoveryv1.AddressType, port int32, addresses ...string) *discoveryv1.EndpointSlice {
	slice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{discoveryv1.LabelServiceName: service},
		},
		AddressType: addressType,
		Ports: []discoveryv1.EndpointPort{
			{Name: new("https"), Port: new(port)},
		},
	}
	for _, address := range addresses {
		slice.Endpoints = append(slice.Endpoints, discoveryv1.Endpoint{Addresses: []string{address}})
	}
	return slice
}

func TestEndpointSliceEndpoints(t *testing.T) {
	for _, tc := range []struct {
		name     string
		objects  []client.Object
		expected []Endpoint
	}{
		{
			name: "single slice",
			objects: []client.Object{
				endpointSlice("kubernetes", Namespace, ServiceName, discoveryv1.AddressTypeIPv4, 6443, "172.18.0.3", "172.18.0.2"),
			},
			expected: []Endpoint{
				{IP: "172.18.0.2", Port: 6443},
				{IP: "172.18.0.3", Port: 6443},
			},
		},
		{
			name: "dual stack slices are merged",
			objects: []client.Object{
				endpointSlice("kubernetes", Namespace, ServiceName, discoveryv1.AddressTypeIPv4, 443, "172.18.0.2"),
				endpointSlice("kubernetes-ipv6", Namespace, ServiceName, discoveryv1.AddressTypeIPv6, 443, "fd00::2"),
			},
			expected: []Endpoint{
				{IP: "172.18.0.2", Port: 443},
				{IP: "fd00::2", Port: 443},
			},
		},
		{
			name: "unrelated and unusable slices are ignored",
			objects: []client.Object{
				endpointSlice("other", Namespace, "other", discoveryv1.AddressTypeIPv4, 6443, "172.18.0.9"),
				endpointSlice("kubernetes", "other-namespace", ServiceName, discoveryv1.AddressTypeIPv4, 6443, "172.18.0.9"),
				endpointSlice("kubernetes-fqdn", Namespace, ServiceName, discoveryv1.AddressTypeFQDN, 6443, "api.example.com"),
				func() *discoveryv1.EndpointSlice {
					slice := endpointSlice("kubernetes-no-https", Namespace, ServiceName, discoveryv1.AddressTypeIPv4, 6443, "172.18.0.9")
					slice.Ports[0].Name = new("http")
					return slice
				}(),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := fake.NewClientBuilder().WithObjects(tc.objects...).Build()
			actual, err := endpointSliceEndpoints(t.Context(), reader)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, sortEndpoints(actual))
		})
	}
}

func TestServiceFromEnv(t *testing.T) {
	for _, tc := range []struct {
		name     string
		host     string
		port     string
		expected Endpoint
		ok       bool
	}{
		{name: "ipv4", host: "10.96.0.1", port: "443", expected: Endpoint{IP: "10.96.0.1", Port: 443}, ok: true},
		{name: "ipv6", host: "fd00:10:96::1", port: "443", expected: Endpoint{IP: "fd00:10:96::1", Port: 443}, ok: true},
		{name: "unset"},
		{name: "hostname", host: "kubernetes.default.svc", port: "443"},
		{name: "invalid port", host: "10.96.0.1", port: "https"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("KUBERNETES_SERVICE_HOST", tc.host)
			t.Setenv("KUBERNETES_SERVICE_PORT", tc.port)
			actual, ok := serviceFromEnv()
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestEndpointCIDR(t *testing.T) {
	assert.Equal(t, "10.0.0.1/32", Endpoint{IP: "10.0.0.1"}.CIDR())
	assert.Equal(t, "fd00::1/128", Endpoint{IP: "fd00::1"}.CIDR())
}

func TestPredicate(t *testing.T) {
	p := (&Tracker{}).Predicate()
	assert.True(t, p.Create(event.CreateEvent{
		Object: endpointSlice("kubernetes", Namespace, ServiceName, discoveryv1.AddressTypeIPv4, 6443),
	}))
	assert.False(t, p.Create(event.CreateEvent{
		Object: endpointSlice("kubernetes", "other-namespace", ServiceName, discoveryv1.AddressTypeIPv4, 6443),
	}))
	assert.False(t, p.Create(event.CreateEvent{
		Object: endpointSlice("other", Namespace, "other", discoveryv1.AddressTypeIPv4, 6443),
	}))
}
