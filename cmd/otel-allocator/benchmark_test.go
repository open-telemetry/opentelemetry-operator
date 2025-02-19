// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	gokitlog "github.com/go-kit/log"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/model/relabel"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/prehook"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/server"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

// BenchmarkProcessTargets benchmarks the whole target allocation pipeline. It starts with data the prometheus
// discovery manager would normally output, and pushes it all the way into the allocator. It notably doe *not* check
// the HTTP server afterward. Test data is chosen to be reasonably representative of what the Prometheus service discovery
// outputs in the real world.
func BenchmarkProcessTargets(b *testing.B) {
	numTargets := 800000
	targetsPerGroup := 5
	groupsPerJob := 20
	tsets := prepareBenchmarkData(numTargets, targetsPerGroup, groupsPerJob)
	for _, strategy := range allocation.GetRegisteredAllocatorNames() {
		b.Run(strategy, func(b *testing.B) {
			targetDiscoverer := createTestDiscoverer(strategy, map[string][]*relabel.Config{})
			targetDiscoverer.UpdateTsets(tsets)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				targetDiscoverer.Reload()
			}
		})
	}
}

// BenchmarkProcessTargetsWithRelabelConfig is BenchmarkProcessTargets with a relabel config set. The relabel config
// does not actually modify any records, but does force the prehook to perform any necessary conversions along the way.
func BenchmarkProcessTargetsWithRelabelConfig(b *testing.B) {
	numTargets := 800000
	targetsPerGroup := 5
	groupsPerJob := 20
	tsets := prepareBenchmarkData(numTargets, targetsPerGroup, groupsPerJob)
	prehookConfig := make(map[string][]*relabel.Config, len(tsets))
	for jobName := range tsets {
		// keep all targets in half the jobs, drop the rest
		jobNrStr := strings.Split(jobName, "-")[1]
		jobNr, err := strconv.Atoi(jobNrStr)
		require.NoError(b, err)
		var action relabel.Action
		if jobNr%2 == 0 {
			action = "keep"
		} else {
			action = "drop"
		}
		prehookConfig[jobName] = []*relabel.Config{
			{
				Action:       action,
				Regex:        relabel.MustNewRegexp(".*"),
				SourceLabels: model.LabelNames{"__address__"},
			},
		}
	}

	for _, strategy := range allocation.GetRegisteredAllocatorNames() {
		b.Run(strategy, func(b *testing.B) {
			targetDiscoverer := createTestDiscoverer(strategy, prehookConfig)
			targetDiscoverer.UpdateTsets(tsets)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				targetDiscoverer.Reload()
			}
		})
	}
}

func prepareBenchmarkData(numTargets, targetsPerGroup, groupsPerJob int) map[string][]*targetgroup.Group {
	numGroups := numTargets / targetsPerGroup
	numJobs := numGroups / groupsPerJob
	jobNamePrefix := "test-"
	groupLabels := model.LabelSet{
		"__meta_kubernetes_pod_controller_name":                                                 "example",
		"__meta_kubernetes_pod_ip":                                                              "10.244.0.251",
		"__meta_kubernetes_pod_uid":                                                             "676ebee7-14f8-481e-a937-d2affaec4105",
		"__meta_kubernetes_endpointslice_port_protocol":                                         "TCP",
		"__meta_kubernetes_endpointslice_endpoint_conditions_ready":                             "true",
		"__meta_kubernetes_service_annotation_kubectl_kubernetes_io_last_applied_configuration": "{\"apiVersion\":\"v1\",\"kind\":\"Service\",\"metadata\":{\"annotations\":{},\"labels\":{\"app\":\"example\"},\"name\":\"example-svc\",\"namespace\":\"example\"},\"spec\":{\"clusterIP\":\"None\",\"ports\":[{\"name\":\"http-example\",\"port\":9006,\"targetPort\":9006}],\"selector\":{\"app\":\"example\"},\"type\":\"ClusterIP\"}}\n",
		"__meta_kubernetes_endpointslice_labelpresent_app":                                      "true",
		"__meta_kubernetes_endpointslice_name":                                                  "example-svc-qgwxf",
		"__address__":                                                                           "10.244.0.251:9006",
		"__meta_kubernetes_endpointslice_endpoint_conditions_terminating":                       "false",
		"__meta_kubernetes_pod_labelpresent_pod_template_hash":                                  "true",
		"__meta_kubernetes_endpointslice_label_kubernetes_io_service_name":                      "example-svc",
		"__meta_kubernetes_endpointslice_labelpresent_service_kubernetes_io_headless":           "true",
		"__meta_kubernetes_pod_label_pod_template_hash":                                         "6b549885f8",
		"__meta_kubernetes_endpointslice_address_target_name":                                   "example-6b549885f8-7tbcw",
		"__meta_kubernetes_pod_labelpresent_app":                                                "true",
		"somelabel":                                                                             "somevalue",
	}
	exampleTarget := model.LabelSet{
		"__meta_kubernetes_endpointslice_port":                                                               "9006",
		"__meta_kubernetes_service_label_app":                                                                "example",
		"__meta_kubernetes_endpointslice_port_name":                                                          "http-example",
		"__meta_kubernetes_pod_ready":                                                                        "true",
		"__meta_kubernetes_endpointslice_address_type":                                                       "IPv4",
		"__meta_kubernetes_endpointslice_label_endpointslice_kubernetes_io_managed_by":                       "endpointslice-controller.k8s.io",
		"__meta_kubernetes_endpointslice_labelpresent_endpointslice_kubernetes_io_managed_by":                "true",
		"__meta_kubernetes_endpointslice_label_app":                                                          "example",
		"__meta_kubernetes_endpointslice_endpoint_conditions_serving":                                        "true",
		"__meta_kubernetes_pod_phase":                                                                        "Running",
		"__meta_kubernetes_pod_controller_kind":                                                              "ReplicaSet",
		"__meta_kubernetes_service_annotationpresent_kubectl_kubernetes_io_last_applied_configuration":       "true",
		"__meta_kubernetes_service_labelpresent_app":                                                         "true",
		"__meta_kubernetes_endpointslice_labelpresent_kubernetes_io_service_name":                            "true",
		"__meta_kubernetes_endpointslice_annotation_endpoints_kubernetes_io_last_change_trigger_time":        "2023-09-27T16:01:29Z",
		"__meta_kubernetes_pod_name":                                                                         "example-6b549885f8-7tbcw",
		"__meta_kubernetes_service_name":                                                                     "example-svc",
		"__meta_kubernetes_namespace":                                                                        "example",
		"__meta_kubernetes_endpointslice_annotationpresent_endpoints_kubernetes_io_last_change_trigger_time": "true",
		"__meta_kubernetes_pod_node_name":                                                                    "kind-control-plane",
		"__meta_kubernetes_endpointslice_address_target_kind":                                                "Pod",
		"__meta_kubernetes_pod_host_ip":                                                                      "172.18.0.2",
		"__meta_kubernetes_endpointslice_label_service_kubernetes_io_headless":                               "",
		"__meta_kubernetes_pod_label_app":                                                                    "example",
	}
	targets := []model.LabelSet{}
	for i := 0; i < numTargets; i++ {
		targets = append(targets, exampleTarget.Clone())
	}
	groups := make([]*targetgroup.Group, numGroups)
	for i := 0; i < numGroups; i++ {
		groupTargets := targets[(i * targetsPerGroup):(i*targetsPerGroup + targetsPerGroup)]
		groups[i] = &targetgroup.Group{
			Labels:  groupLabels,
			Targets: groupTargets,
		}
	}
	tsets := make(map[string][]*targetgroup.Group, numJobs)
	for i := 0; i < numJobs; i++ {
		jobGroups := groups[(i * groupsPerJob):(i*groupsPerJob + groupsPerJob)]
		jobName := fmt.Sprintf("%s%d", jobNamePrefix, i)
		tsets[jobName] = jobGroups
	}
	return tsets
}

func createTestDiscoverer(allocationStrategy string, prehookConfig map[string][]*relabel.Config) *target.Discoverer {
	ctx := context.Background()
	logger := ctrl.Log.WithName(fmt.Sprintf("bench-%s", allocationStrategy))
	ctrl.SetLogger(logr.New(log.NullLogSink{}))
	allocatorPrehook := prehook.New("relabel-config", logger)
	allocatorPrehook.SetConfig(prehookConfig)
	allocator, err := allocation.New(allocationStrategy, logger, allocation.WithFilter(allocatorPrehook))
	srv := server.NewServer(logger, allocator, "localhost:0")
	if err != nil {
		setupLog.Error(err, "Unable to initialize allocation strategy")
		os.Exit(1)
	}
	registry := prometheus.NewRegistry()
	sdMetrics, _ := discovery.CreateAndRegisterSDMetrics(registry)
	discoveryManager := discovery.NewManager(ctx, gokitlog.NewNopLogger(), registry, sdMetrics)
	targetDiscoverer := target.NewDiscoverer(logger, discoveryManager, allocatorPrehook, srv, allocator.SetTargets)
	return targetDiscoverer
}
