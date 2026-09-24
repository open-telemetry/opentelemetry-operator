// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package apiserverendpoints resolves the addresses through which the Kubernetes API server is reachable
// from inside the cluster, for use in NetworkPolicy egress rules.
//
// The addresses come from the EndpointSlices backing the `default/kubernetes` Service, which change over the
// cluster's lifetime (for example when a managed control plane is scaled or upgraded). The Tracker is the single
// source of truth for which of those addresses are considered live: components read them from it on every
// reconcile, and watch EndpointSlices, filtered with Tracker.Predicate, only as a trigger to reconcile.
// CacheByObject restricts the manager cache to only the relevant EndpointSlices.
package apiserverendpoints

import (
	"cmp"
	"context"
	"fmt"
	"net"
	"net/netip"
	"os"
	"slices"
	"strconv"

	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Namespace is the namespace of the Kubernetes API server Service.
	Namespace = metav1.NamespaceDefault
	// ServiceName is the name of the Kubernetes API server Service.
	ServiceName = "kubernetes"
	// portName is the name of the API server port in the Service and its EndpointSlices.
	portName = "https"
)

// Endpoint is a single address and port through which the API server can be reached.
type Endpoint struct {
	IP   string
	Port int32
}

// CIDR returns the single-address CIDR for the endpoint's IP, suitable for a NetworkPolicy IPBlock.
func (e Endpoint) CIDR() string {
	addr, err := netip.ParseAddr(e.IP)
	if err != nil {
		return e.IP
	}
	return netip.PrefixFrom(addr, addr.BitLen()).String()
}

// CacheByObject returns the cache configuration restricting EndpointSlices to the ones backing the API server
// Service. It overrides the manager's default namespaces, so the API server EndpointSlices are available even
// when the operator only watches a subset of namespaces.
func CacheByObject() (client.Object, cache.ByObject) {
	return &discoveryv1.EndpointSlice{}, cache.ByObject{
		Namespaces: map[string]cache.Config{Namespace: {}},
		Label:      labels.SelectorFromSet(labels.Set{discoveryv1.LabelServiceName: ServiceName}),
	}
}

func isAPIServerEndpointSlice(obj client.Object) bool {
	return obj.GetNamespace() == Namespace && obj.GetLabels()[discoveryv1.LabelServiceName] == ServiceName
}

// endpointSliceEndpoints returns the endpoints listed in the API server's EndpointSlices.
func endpointSliceEndpoints(ctx context.Context, reader client.Reader) ([]Endpoint, error) {
	var sliceList discoveryv1.EndpointSliceList
	if err := reader.List(ctx, &sliceList,
		client.InNamespace(Namespace),
		client.MatchingLabels{discoveryv1.LabelServiceName: ServiceName},
	); err != nil {
		return nil, fmt.Errorf("failed to list the Kubernetes API server EndpointSlices: %w", err)
	}

	var endpoints []Endpoint
	for i := range sliceList.Items {
		endpoints = append(endpoints, fromEndpointSlice(&sliceList.Items[i])...)
	}
	return endpoints, nil
}

// sortEndpoints sorts the endpoints by port and IP, and removes duplicates.
func sortEndpoints(endpoints []Endpoint) []Endpoint {
	slices.SortFunc(endpoints, func(a, b Endpoint) int {
		return cmp.Or(cmp.Compare(a.Port, b.Port), cmp.Compare(a.IP, b.IP))
	})
	return slices.Compact(endpoints)
}

func fromEndpointSlice(slice *discoveryv1.EndpointSlice) []Endpoint {
	if slice.AddressType != discoveryv1.AddressTypeIPv4 && slice.AddressType != discoveryv1.AddressTypeIPv6 {
		return nil
	}
	portIndex := slices.IndexFunc(slice.Ports, func(p discoveryv1.EndpointPort) bool {
		return p.Port != nil && p.Name != nil && *p.Name == portName
	})
	if portIndex == -1 {
		return nil
	}
	port := *slice.Ports[portIndex].Port

	// Endpoints are included regardless of their conditions: a terminating endpoint can still be serving
	// connections, and a not-yet-ready one is about to.
	var endpoints []Endpoint
	for _, endpoint := range slice.Endpoints {
		for _, address := range endpoint.Addresses {
			endpoints = append(endpoints, Endpoint{IP: address, Port: port})
		}
	}
	return endpoints
}

// serviceFromEnv returns the API server Service's ClusterIP endpoint, from the environment variables Kubernetes
// sets in every container.
func serviceFromEnv() (Endpoint, bool) {
	host := os.Getenv("KUBERNETES_SERVICE_HOST")
	if net.ParseIP(host) == nil {
		return Endpoint{}, false
	}
	port, err := strconv.ParseInt(os.Getenv("KUBERNETES_SERVICE_PORT"), 10, 32)
	if err != nil || port <= 0 {
		return Endpoint{}, false
	}
	return Endpoint{IP: host, Port: int32(port)}, true
}
