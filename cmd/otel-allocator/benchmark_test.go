// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	"github.com/prometheus/prometheus/model/labels"
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
	numTargets := 10000
	targetsPerGroup := 5
	groupsPerJob := 20
	tsets := prepareBenchmarkData(numTargets, targetsPerGroup, groupsPerJob)
	labelsBuilder := labels.NewBuilder(labels.EmptyLabels())

	b.ResetTimer()
	for _, strategy := range allocation.GetRegisteredAllocatorNames() {
		b.Run(strategy, func(b *testing.B) {
			targetDiscoverer, allocator := createTestDiscoverer(strategy, map[string][]*relabel.Config{})
			for i := 0; i < b.N; i++ {
				targetDiscoverer.ProcessTargets(labelsBuilder, tsets, allocator.SetTargets)
			}
		})
	}
}

// BenchmarkProcessTargetsWithRelabelConfig is BenchmarkProcessTargets with a relabel config set. The relabel config
// does not actually modify any records, but does force the prehook to perform any necessary conversions along the way.
func BenchmarkProcessTargetsWithRelabelConfig(b *testing.B) {
	numTargets := 10000
	targetsPerGroup := 5
	groupsPerJob := 20
	tsets := prepareBenchmarkData(numTargets, targetsPerGroup, groupsPerJob)
	labelsBuilder := labels.NewBuilder(labels.EmptyLabels())
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

	b.ResetTimer()
	for _, strategy := range allocation.GetRegisteredAllocatorNames() {
		b.Run(strategy, func(b *testing.B) {
			targetDiscoverer, allocator := createTestDiscoverer(strategy, prehookConfig)
			for i := 0; i < b.N; i++ {
				targetDiscoverer.ProcessTargets(labelsBuilder, tsets, allocator.SetTargets)
			}
		})
	}
}

func prepareBenchmarkData(numTargets, targetsPerGroup, groupsPerJob int) map[string][]*targetgroup.Group {
	numGroups := numTargets / targetsPerGroup
	numJobs := numGroups / groupsPerJob
	jobNamePrefix := "test-"
	groupLabels := model.LabelSet{
		"__meta_kubernetes_endpointslice_address_target_kind":                                                "Pod",
		"__meta_kubernetes_endpointslice_address_target_name":                                                "parquet-test-job-gen-ax-28878990-dbmj2",
		"__meta_kubernetes_endpointslice_address_type":                                                       "IPv4",
		"__meta_kubernetes_endpointslice_annotation_endpoints_kubernetes_io_last_change_trigger_time":        "2024-11-27T20:30:03Z",
		"__meta_kubernetes_endpointslice_annotationpresent_endpoints_kubernetes_io_last_change_trigger_time": "true",
		"__meta_kubernetes_endpointslice_endpoint_conditions_ready":                                          "false",
		"__meta_kubernetes_endpointslice_endpoint_conditions_serving":                                        "false",
		"__meta_kubernetes_endpointslice_endpoint_conditions_terminating":                                    "false",
		"__meta_kubernetes_endpointslice_endpoint_node_name":                                                 "f-03d1abced6fcd44e1",
		"__meta_kubernetes_endpointslice_endpoint_zone":                                                      "eu-west-32a",
		"__meta_kubernetes_endpointslice_label_AZ":                                                           "eu-west-32a",
		"__meta_kubernetes_endpointslice_label_CLOUDFLARE_SITE_ID":                                           "qddf2d5372648c1e877835b0c4af7f811123",
		"__meta_kubernetes_endpointslice_label_CLUSTER_DNS":                                                  "staging.eu-west-32.k8s-staging",
		"__meta_kubernetes_endpointslice_label_DEPLOYABLE":                                                   "parquet-test-job",
		"__meta_kubernetes_endpointslice_label_DOMAIN":                                                       "app.staging.xablau.com",
		"__meta_kubernetes_endpointslice_label_ENV_CLASS":                                                    "staging",
		"__meta_kubernetes_endpointslice_label_ENV_ID":                                                       "staging",
		"__meta_kubernetes_endpointslice_label_ENV_TYPE":                                                     "staging",
		"__meta_kubernetes_endpointslice_label_HOST_PROJECT":                                                 "staging",
		"__meta_kubernetes_endpointslice_label_ORG":                                                          "contoso-corp",
		"__meta_kubernetes_endpointslice_label_PRODUCT":                                                      "xablauax-structure",
		"__meta_kubernetes_endpointslice_label_REGION":                                                       "eu-west-32",
		"__meta_kubernetes_endpointslice_label_REPOSITORY":                                                   "test-ax-service",
		"__meta_kubernetes_endpointslice_label_SERVICE_NAME":                                                 "parquet-test-job-gen-ax",
		"__meta_kubernetes_endpointslice_label_SERVICE_TYPE":                                                 "application",
		"__meta_kubernetes_endpointslice_label_TEAM":                                                         "xablauax-structure",
		"__meta_kubernetes_endpointslice_label_app_kubernetes_io_instance":                                   "parquet-test-job-gen-ax",
		"__meta_kubernetes_endpointslice_label_app_kubernetes_io_managed_by":                                 "Helm",
		"__meta_kubernetes_endpointslice_label_app_kubernetes_io_name":                                       "gen-ax",
		"__meta_kubernetes_endpointslice_label_argocd_argoproj_io_instance":                                  "parquet-test-job-gen-ax-staging",
		"__meta_kubernetes_endpointslice_label_endpointslice_kubernetes_io_managed_by":                       "endpointslice-controller.k8s.io",
		"__meta_kubernetes_endpointslice_label_kubernetes_io_service_name":                                   "parquet-test-job-gen-ax",
		"__meta_kubernetes_endpointslice_label_pipelines_contoso-corp_net_managed_by":                        "abc-pipeline",
		"__meta_kubernetes_endpointslice_label_prometheus_io_scrape":                                         "true",
		"__meta_kubernetes_endpointslice_labelpresent_AZ":                                                    "true",
		"__meta_kubernetes_endpointslice_labelpresent_CLOUDFLARE_SITE_ID":                                    "true",
		"__meta_kubernetes_endpointslice_labelpresent_CLUSTER_DNS":                                           "true",
		"__meta_kubernetes_endpointslice_labelpresent_DEPLOYABLE":                                            "true",
		"__meta_kubernetes_endpointslice_labelpresent_DOMAIN":                                                "true",
		"__meta_kubernetes_endpointslice_labelpresent_ENV_CLASS":                                             "true",
		"__meta_kubernetes_endpointslice_labelpresent_ENV_ID":                                                "true",
		"__meta_kubernetes_endpointslice_labelpresent_ENV_TYPE":                                              "true",
		"__meta_kubernetes_endpointslice_labelpresent_HOST_PROJECT":                                          "true",
		"__meta_kubernetes_endpointslice_labelpresent_ORG":                                                   "true",
		"__meta_kubernetes_endpointslice_labelpresent_PRODUCT":                                               "true",
		"__meta_kubernetes_endpointslice_labelpresent_REGION":                                                "true",
		"__meta_kubernetes_endpointslice_labelpresent_REPOSITORY":                                            "true",
		"__meta_kubernetes_endpointslice_labelpresent_SERVICE_NAME":                                          "true",
		"__meta_kubernetes_endpointslice_labelpresent_SERVICE_TYPE":                                          "true",
		"__meta_kubernetes_endpointslice_labelpresent_TEAM":                                                  "true",
		"__meta_kubernetes_endpointslice_labelpresent_app_kubernetes_io_instance":                            "true",
		"__meta_kubernetes_endpointslice_labelpresent_app_kubernetes_io_managed_by":                          "true",
		"__meta_kubernetes_endpointslice_labelpresent_app_kubernetes_io_name":                                "true",
		"__meta_kubernetes_endpointslice_labelpresent_argocd_argoproj_io_instance":                           "true",
		"__meta_kubernetes_endpointslice_labelpresent_endpointslice_kubernetes_io_managed_by":                "true",
		"__meta_kubernetes_endpointslice_labelpresent_kubernetes_io_service_name":                            "true",
		"__meta_kubernetes_endpointslice_labelpresent_pipelines_contoso-corp_net_managed_by":                 "true",
		"__meta_kubernetes_endpointslice_labelpresent_prometheus_io_scrape":                                  "true",
		"__meta_kubernetes_endpointslice_name":                                                               "parquet-test-job-gen-ax-hggc8",
		"__meta_kubernetes_endpointslice_port":                                                               "8080",
		"__meta_kubernetes_endpointslice_port_name":                                                          "http",
		"__meta_kubernetes_endpointslice_port_protocol":                                                      "TCP",
		"__meta_kubernetes_namespace":                                                                        "xablauax-structure",
		"__meta_kubernetes_pod_annotation_kubectl_kubernetes_io_default_container":                           "gen-ax",
		"__meta_kubernetes_pod_annotation_meta_helm_sh_release_name":                                         "parquet-test-job-gen-ax",
		"__meta_kubernetes_pod_annotation_meta_helm_sh_release_namespace":                                    "xablauax-structure",
		"__meta_kubernetes_pod_annotation_reloader_stakater_com_auto":                                        "true",
		"__meta_kubernetes_pod_annotationpresent_kubectl_kubernetes_io_default_container":                    "true",
		"__meta_kubernetes_pod_annotationpresent_meta_helm_sh_release_name":                                  "true",
		"__meta_kubernetes_pod_annotationpresent_meta_helm_sh_release_namespace":                             "true",
		"__meta_kubernetes_pod_annotationpresent_reloader_stakater_com_auto":                                 "true",
		"__meta_kubernetes_pod_container_image":                                                              "449726762866.dkr.ecr.eu-west-32.amazonaws.com/gen-ax:f2b348e7",
		"__meta_kubernetes_pod_container_name":                                                               "gen-ax",
		"__meta_kubernetes_pod_container_port_name":                                                          "http",
		"__meta_kubernetes_pod_container_port_number":                                                        "8080",
		"__meta_kubernetes_pod_container_port_protocol":                                                      "TCP",
		"__meta_kubernetes_pod_controller_kind":                                                              "Job",
		"__meta_kubernetes_pod_controller_name":                                                              "parquet-test-job-gen-ax-28878990",
		"__meta_kubernetes_pod_host_ip":                                                                      "10.9.99.99",
		"__meta_kubernetes_pod_ip":                                                                           "10.8.110.131",
		"__meta_kubernetes_pod_label_AZ":                                                                     "eu-west-32a",
		"__meta_kubernetes_pod_label_CLOUDFLARE_SITE_ID":                                                     "qddf2d5372648c1e877835b0c4af7f811123",
		"__meta_kubernetes_pod_label_CLUSTER_DNS":                                                            "staging.eu-west-32.k8s-staging",
		"__meta_kubernetes_pod_label_DEPLOYABLE":                                                             "parquet-test-job",
		"__meta_kubernetes_pod_label_DOMAIN":                                                                 "app.staging.xablau.com",
		"__meta_kubernetes_pod_label_ENV_CLASS":                                                              "staging",
		"__meta_kubernetes_pod_label_ENV_ID":                                                                 "staging",
		"__meta_kubernetes_pod_label_ENV_TYPE":                                                               "staging",
		"__meta_kubernetes_pod_label_FEATURE_GROUP_ID":                                                       "unknown-xablau-group-id",
		"__meta_kubernetes_pod_label_FEATURE_ID":                                                             "unknown-xablau-id",
		"__meta_kubernetes_pod_label_HOST_PROJECT":                                                           "staging",
		"__meta_kubernetes_pod_label_ORG":                                                                    "contoso-corp",
		"__meta_kubernetes_pod_label_PRODUCT":                                                                "xablauax-structure",
		"__meta_kubernetes_pod_label_REGION":                                                                 "eu-west-32",
		"__meta_kubernetes_pod_label_REPOSITORY":                                                             "test-ax-service",
		"__meta_kubernetes_pod_label_SERVICE_NAME":                                                           "parquet-test-job-gen-ax",
		"__meta_kubernetes_pod_label_SERVICE_TYPE":                                                           "application",
		"__meta_kubernetes_pod_label_TEAM":                                                                   "xablauax-structure",
		"__meta_kubernetes_pod_label_app_kubernetes_io_instance":                                             "parquet-test-job-gen-ax",
		"__meta_kubernetes_pod_label_app_kubernetes_io_managed_by":                                           "Helm",
		"__meta_kubernetes_pod_label_app_kubernetes_io_name":                                                 "gen-ax",
		"__meta_kubernetes_pod_label_batch_kubernetes_io_controller_uid":                                     "df32b32c-ea37-43e5-917b-73ced043b021",
	}
	exampleTarget := model.LabelSet{
		"__meta_kubernetes_pod_label_batch_kubernetes_io_job_name":                                     "parquet-test-job-gen-ax-28878990",
		"__meta_kubernetes_pod_label_controller_uid":                                                   "df32b32c-ea37-43e5-917b-73ced043b021",
		"__meta_kubernetes_pod_label_job_name":                                                         "parquet-test-job-gen-ax-28878990",
		"__meta_kubernetes_pod_label_pipelines_contoso_corp_net_managed_by":                            "abc-pipeline",
		"__meta_kubernetes_pod_labelpresent_AZ":                                                        "true",
		"__meta_kubernetes_pod_labelpresent_CLOUDFLARE_SITE_ID":                                        "true",
		"__meta_kubernetes_pod_labelpresent_CLUSTER_DNS":                                               "true",
		"__meta_kubernetes_pod_labelpresent_DEPLOYABLE":                                                "true",
		"__meta_kubernetes_pod_labelpresent_DOMAIN":                                                    "true",
		"__meta_kubernetes_pod_labelpresent_ENV_CLASS":                                                 "true",
		"__meta_kubernetes_pod_labelpresent_ENV_ID":                                                    "true",
		"__meta_kubernetes_pod_labelpresent_ENV_TYPE":                                                  "true",
		"__meta_kubernetes_pod_labelpresent_FEATURE_GROUP_ID":                                          "true",
		"__meta_kubernetes_pod_labelpresent_FEATURE_ID":                                                "true",
		"__meta_kubernetes_pod_labelpresent_HOST_PROJECT":                                              "true",
		"__meta_kubernetes_pod_labelpresent_ORG":                                                       "true",
		"__meta_kubernetes_pod_labelpresent_PRODUCT":                                                   "true",
		"__meta_kubernetes_pod_labelpresent_REGION":                                                    "true",
		"__meta_kubernetes_pod_labelpresent_REPOSITORY":                                                "true",
		"__meta_kubernetes_pod_labelpresent_SERVICE_NAME":                                              "true",
		"__meta_kubernetes_pod_labelpresent_SERVICE_TYPE":                                              "true",
		"__meta_kubernetes_pod_labelpresent_TEAM":                                                      "true",
		"__meta_kubernetes_pod_labelpresent_app_kubernetes_io_instance":                                "true",
		"__meta_kubernetes_pod_labelpresent_app_kubernetes_io_managed_by":                              "true",
		"__meta_kubernetes_pod_labelpresent_app_kubernetes_io_name":                                    "true",
		"__meta_kubernetes_pod_labelpresent_batch_kubernetes_io_controller_uid":                        "true",
		"__meta_kubernetes_pod_labelpresent_batch_kubernetes_io_job_name":                              "true",
		"__meta_kubernetes_pod_labelpresent_controller_uid":                                            "true",
		"__meta_kubernetes_pod_labelpresent_job_name":                                                  "true",
		"__meta_kubernetes_pod_labelpresent_pipelines_contoso-corp_net_managed_by":                     "true",
		"__meta_kubernetes_pod_name":                                                                   "parquet-test-job-gen-ax-28878990-dbmj2",
		"__meta_kubernetes_pod_node_name":                                                              "f-03d1abced6fcd44e1",
		"__meta_kubernetes_pod_phase":                                                                  "Succeeded",
		"__meta_kubernetes_pod_ready":                                                                  "false",
		"__meta_kubernetes_pod_uid":                                                                    "5cc88535-3fb6-4923-8b9c-70b2eca47c11",
		"__meta_kubernetes_service_annotation_kubectl_kubernetes_io_last_applied_configuration":        "{\"apiVersion\":\"v1\",\"kind\":\"Service\",\"metadata\":{\"annotations\":{\"meta.helm.sh/release-name\":\"parquet-test-job-gen-ax\",\"meta.helm.sh/release-namespace\":\"xablauax-structure\"},\"labels\":{\"AZ\":\"eu-west-32a\",\"CLOUDFLARE_SITE_ID\":\"qddf2d5372648c1e877835b0c4af7f811123\",\"CLUSTER_DNS\":\"staging.eu-west-32.k8s-staging\",\"DEPLOYABLE\":\"parquet-test-job\",\"DOMAIN\":\"app.staging.xablau.com\",\"ENV_CLASS\":\"staging\",\"ENV_ID\":\"staging\",\"ENV_TYPE\":\"staging\",\"HOST_PROJECT\":\"staging\",\"ORG\":\"contoso-corp\",\"PRODUCT\":\"xablauax-structure\",\"REGION\":\"eu-west-32\",\"REPOSITORY\":\"test-ax-service\",\"SERVICE_NAME\":\"parquet-test-job-gen-ax\",\"SERVICE_TYPE\":\"application\",\"TEAM\":\"xablauax-structure\",\"app.kubernetes.io/instance\":\"parquet-test-job-gen-ax\",\"app.kubernetes.io/managed-by\":\"Helm\",\"app.kubernetes.io/name\":\"gen-ax\",\"argocd.argoproj.io/instance\":\"parquet-test-job-gen-ax-staging\",\"pipelines.contoso-corp.net/managed-by\":\"abc-pipeline\",\"prometheus.io/scrape\":\"true\"},\"name\":\"parquet-test-job-gen-ax\",\"namespace\":\"xablauax-structure\"},\"spec\":{\"ports\":[{\"name\":\"http\",\"port\":8080,\"protocol\":\"TCP\",\"targetPort\":8080}],\"selector\":{\"app.kubernetes.io/instance\":\"parquet-test-job-gen-ax\",\"app.kubernetes.io/name\":\"gen-ax\"},\"type\":\"ClusterIP\"}}\n",
		"__meta_kubernetes_service_annotation_meta_helm_sh_release_name":                               "parquet-test-job-gen-ax",
		"__meta_kubernetes_service_annotation_meta_helm_sh_release_namespace":                          "xablauax-structure",
		"__meta_kubernetes_service_annotationpresent_kubectl_kubernetes_io_last_applied_configuration": "true",
		"__meta_kubernetes_service_annotationpresent_meta_helm_sh_release_name":                        "true",
		"__meta_kubernetes_service_annotationpresent_meta_helm_sh_release_namespace":                   "true",
		"__meta_kubernetes_service_label_AZ":                                                           "eu-west-32a",
		"__meta_kubernetes_service_label_CLOUDFLARE_SITE_ID":                                           "qddf2d5372648c1e877835b0c4af7f811123",
		"__meta_kubernetes_service_label_CLUSTER_DNS":                                                  "staging.eu-west-32.k8s-staging",
		"__meta_kubernetes_service_label_DEPLOYABLE":                                                   "parquet-test-job",
		"__meta_kubernetes_service_label_DOMAIN":                                                       "app.staging.xablau.com",
		"__meta_kubernetes_service_label_ENV_CLASS":                                                    "staging",
		"__meta_kubernetes_service_label_ENV_ID":                                                       "staging",
		"__meta_kubernetes_service_label_ENV_TYPE":                                                     "staging",
		"__meta_kubernetes_service_label_HOST_PROJECT":                                                 "staging",
		"__meta_kubernetes_service_label_ORG":                                                          "contoso-corp",
		"__meta_kubernetes_service_label_PRODUCT":                                                      "xablauax-structure",
		"__meta_kubernetes_service_label_REGION":                                                       "eu-west-32",
		"__meta_kubernetes_service_label_REPOSITORY":                                                   "test-ax-service",
		"__meta_kubernetes_service_label_SERVICE_NAME":                                                 "parquet-test-job-gen-ax",
		"__meta_kubernetes_service_label_SERVICE_TYPE":                                                 "application",
		"__meta_kubernetes_service_label_TEAM":                                                         "xablauax-structure",
		"__meta_kubernetes_service_label_app_kubernetes_io_instance":                                   "parquet-test-job-gen-ax",
		"__meta_kubernetes_service_label_app_kubernetes_io_managed_by":                                 "Helm",
		"__meta_kubernetes_service_label_app_kubernetes_io_name":                                       "gen-ax",
		"__meta_kubernetes_service_label_argocd_argoproj_io_instance":                                  "parquet-test-job-gen-ax-staging",
		"__meta_kubernetes_service_label_pipelines_contoso-corp_net_managed_by":                        "abc-pipeline",
		"__meta_kubernetes_service_label_prometheus_io_scrape":                                         "true",
		"__meta_kubernetes_service_labelpresent_AZ":                                                    "true",
		"__meta_kubernetes_service_labelpresent_CLOUDFLARE_SITE_ID":                                    "true",
		"__meta_kubernetes_service_labelpresent_CLUSTER_DNS":                                           "true",
		"__meta_kubernetes_service_labelpresent_DEPLOYABLE":                                            "true",
		"__meta_kubernetes_service_labelpresent_DOMAIN":                                                "true",
		"__meta_kubernetes_service_labelpresent_ENV_CLASS":                                             "true",
		"__meta_kubernetes_service_labelpresent_ENV_ID":                                                "true",
		"__meta_kubernetes_service_labelpresent_ENV_TYPE":                                              "true",
		"__meta_kubernetes_service_labelpresent_HOST_PROJECT":                                          "true",
		"__meta_kubernetes_service_labelpresent_ORG":                                                   "true",
		"__meta_kubernetes_service_labelpresent_PRODUCT":                                               "true",
		"__meta_kubernetes_service_labelpresent_REGION":                                                "true",
		"__meta_kubernetes_service_labelpresent_REPOSITORY":                                            "true",
		"__meta_kubernetes_service_labelpresent_SERVICE_NAME":                                          "true",
		"__meta_kubernetes_service_labelpresent_SERVICE_TYPE":                                          "true",
		"__meta_kubernetes_service_labelpresent_TEAM":                                                  "true",
		"__meta_kubernetes_service_labelpresent_app_kubernetes_io_instance":                            "true",
		"__meta_kubernetes_service_labelpresent_app_kubernetes_io_managed_by":                          "true",
		"__meta_kubernetes_service_labelpresent_app_kubernetes_io_name":                                "true",
		"__meta_kubernetes_service_labelpresent_argocd_argoproj_io_instance":                           "true",
		"__meta_kubernetes_service_labelpresent_pipelines_contoso-corp_net_managed_by":                 "true",
		"__meta_kubernetes_service_labelpresent_prometheus":                                            "true",
		"__meta_kubernetes_service_name":                                                               "parquet-test-job-gen-ax",
	}
	targets := []model.LabelSet{}
	for i := 0; i < numTargets; i++ {
		exampleTarget["__address__"] = model.LabelValue(fmt.Sprintf("10.8.110.%d:8080", i))
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

func createTestDiscoverer(allocationStrategy string, prehookConfig map[string][]*relabel.Config) (*target.Discoverer, allocation.Allocator) {
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
	targetDiscoverer := target.NewDiscoverer(logger, discoveryManager, allocatorPrehook, srv)
	return targetDiscoverer, allocator
}
