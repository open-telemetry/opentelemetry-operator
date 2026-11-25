// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package collectormetrics

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"github.com/open-telemetry/opentelemetry-operator/internal/testing/e2e"
)

const (
	samplePort     = 9100 // the port node-exporter listens on in the sample app
	sampleReplicas = 2    // replicas in testdata/sample-app-dual-stack.yaml
)

// dualStackService builds the sample app's Service with the given IP family policy
// and families. With no families, the Service's primary family is the cluster's.
func dualStackService(policy corev1.IPFamilyPolicy, families ...corev1.IPFamily) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: sampleApp, Labels: map[string]string{"app": sampleApp}},
		Spec: corev1.ServiceSpec{
			Selector:       map[string]string{"app": sampleApp},
			IPFamilyPolicy: &policy,
			IPFamilies:     families,
			Ports: []corev1.ServicePort{{
				Name:       "metrics",
				Port:       samplePort,
				TargetPort: intstr.FromInt32(samplePort),
			}},
		},
	}
}

// TestServiceMonitorDualStack validates that a ServiceMonitor selecting a dual-stack
// Service yields one target per pod, at the pod's address in the Service's primary IP
// family. Kubernetes publishes one EndpointSlice per address family for such a
// Service, and the allocator discovers ServiceMonitor targets with the endpointslice
// role, which must use only the slice of the primary family. See
// open-telemetry/opentelemetry-operator#4494.
func TestServiceMonitorDualStack(t *testing.T) {
	testenv.Test(t,
		dualStackFeature("PreferDualStack Service with the cluster's primary family",
			dualStackService(corev1.IPFamilyPolicyPreferDualStack)),
		dualStackFeature("RequireDualStack Service with IPv6 as primary family",
			dualStackService(corev1.IPFamilyPolicyRequireDualStack, corev1.IPv6Protocol, corev1.IPv4Protocol)),
	)
}

// dualStackFeature deploys the sample app behind svc with the ServiceMonitor
// differential setup and asserts on the allocator's targets for it.
func dualStackFeature(name string, svc *corev1.Service) features.Feature {
	return features.New(name).
		Setup(setup(dualStackAppManifest, smCollectorManifest, svc, serviceMonitor(pipelineTA), serviceMonitor(pipelineProm))).
		Assess("the Service has an EndpointSlice per IP family", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			// Guards against a vacuous pass on a cluster that is not dual-stack, where a
			// PreferDualStack Service silently gets a single family.
			ns := e2e.Namespace(t, ctx)
			cs := e2e.ClientSet(t, cfg)
			got, err := cs.CoreV1().Services(ns).Get(ctx, sampleApp, metav1.GetOptions{})
			require.NoError(t, err, "get Service %s", sampleApp)
			require.Len(t, got.Spec.IPFamilies, 2, "Service %s is not dual-stack; is the cluster dual-stack?", sampleApp)

			require.EventuallyWithT(t, func(c *assert.CollectT) {
				list, err := cs.DiscoveryV1().EndpointSlices(ns).List(ctx, metav1.ListOptions{
					LabelSelector: discoveryv1.LabelServiceName + "=" + sampleApp,
				})
				if !assert.NoError(c, err, "list EndpointSlices") {
					return
				}
				types := map[discoveryv1.AddressType]bool{}
				for _, s := range list.Items {
					types[s.AddressType] = true
				}
				assert.True(c, types[discoveryv1.AddressTypeIPv4], "no IPv4 EndpointSlice")
				assert.True(c, types[discoveryv1.AddressTypeIPv6], "no IPv6 EndpointSlice")
			}, time.Minute, 2*time.Second)
			return ctx
		}).
		Assess("one target per pod, at the address of the Service's primary family", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			want := primaryFamilyInstances(ctx, t, cfg)
			prom(t, ctx, cfg).Eventually(ctx, t, fmt.Sprintf(`up{%s=%q}`, pipelineLabel, pipelineTA), func(v model.Vector) error {
				got := make([]string, 0, len(v))
				for _, s := range v {
					if s.Value != 1 {
						return fmt.Errorf("target %s is down", s.Metric[model.InstanceLabel])
					}
					got = append(got, string(s.Metric[model.InstanceLabel]))
				}
				slices.Sort(got)
				if !slices.Equal(got, want) {
					return fmt.Errorf("instances %v, want %v", got, want)
				}
				return nil
			})
			return ctx
		}).
		Assess("target identity matches prometheus-operator (up)", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			prom(t, ctx, cfg).SameLabelsAcross(ctx, t, e2e.Differential{Query: `up`, PartitionLabel: pipelineLabel, WantPartitions: 2})
			return ctx
		}).
		Feature()
}

// primaryFamilyInstances returns the sorted `instance` values expected for the sample
// app: each pod's IP in the Service's primary family, joined with the scraped port.
func primaryFamilyInstances(ctx context.Context, t *testing.T, cfg *envconf.Config) []string {
	t.Helper()
	ns := e2e.Namespace(t, ctx)
	cs := e2e.ClientSet(t, cfg)

	svc, err := cs.CoreV1().Services(ns).Get(ctx, sampleApp, metav1.GetOptions{})
	require.NoError(t, err, "get Service %s", sampleApp)
	require.NotEmpty(t, svc.Spec.IPFamilies, "Service %s has no IP families", sampleApp)
	primary := svc.Spec.IPFamilies[0]

	pods, err := cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: "app=" + sampleApp})
	require.NoError(t, err, "list %s pods", sampleApp)
	require.Len(t, pods.Items, sampleReplicas, "%s pods", sampleApp)

	var instances []string
	for _, pod := range pods.Items {
		idx := slices.IndexFunc(pod.Status.PodIPs, func(ip corev1.PodIP) bool {
			addr, err := netip.ParseAddr(ip.IP)
			return err == nil && addr.Is6() == (primary == corev1.IPv6Protocol)
		})
		require.GreaterOrEqual(t, idx, 0, "pod %s has no %s address: %v", pod.Name, primary, pod.Status.PodIPs)
		instances = append(instances, net.JoinHostPort(pod.Status.PodIPs[idx].IP, strconv.Itoa(samplePort)))
	}
	slices.Sort(instances)
	return instances
}
