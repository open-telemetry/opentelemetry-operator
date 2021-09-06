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

// Package controllers contains the main controller, where the reconciliation starts.
package controllers

import (
	"github.com/signalfx/splunk-otel-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

func setAgentDefaults(instance *v1alpha1.SplunkOtelAgent) {
	// agent must be daemonset.
	// TODO(splunk): add different API/controller for clusterreceiver (deployment)
	// and gateway (deployment).
	instance.Spec.Mode = "daemonset"
	instance.Spec.HostNetwork = true

	// TODO(splunk): OpenShift uses externally configured SecurityContextConstraints so doesn't need SecuirtyContext here
	// Investigate if we need SecuirtyContext for non-openshift platforms

	if instance.Spec.Volumes == nil {
		instance.Spec.Volumes = []v1.Volume{
			{
				Name: "hostfs",
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{Path: "/"},
				},
			},
			{
				Name: "etc-passwd",
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{Path: "/etc/passwd"},
				},
			},
		}
	}

	if instance.Spec.VolumeMounts == nil {
		hostToContainer := v1.MountPropagationHostToContainer
		instance.Spec.VolumeMounts = []v1.VolumeMount{
			{
				Name:             "hostfs",
				MountPath:        "/hostfs",
				ReadOnly:         true,
				MountPropagation: &hostToContainer,
			},
			{
				Name:      "etc-passwd",
				MountPath: "/etc/passwd",
				ReadOnly:  true,
			},
		}
	}

	if instance.Spec.Tolerations == nil {
		instance.Spec.Tolerations = []v1.Toleration{
			{
				Key:      "node.alpha.kubernetes.io/role",
				Effect:   v1.TaintEffectNoSchedule,
				Operator: v1.TolerationOpExists,
			},
			{
				Key:      "node-role.kubernetes.io/master",
				Effect:   v1.TaintEffectNoSchedule,
				Operator: v1.TolerationOpExists,
			},
		}
	}

	if instance.Spec.Env == nil {
		instance.Spec.Env = []v1.EnvVar{
			{
				Name: "SPLUNK_ACCESS_TOKEN",
				ValueFrom: &v1.EnvVarSource{
					SecretKeyRef: &v1.SecretKeySelector{
						LocalObjectReference: v1.LocalObjectReference{Name: "splunk-access-token"},
						Key:                  "access-token",
					},
				},
			},
			newEnvVarWithFieldRef("MY_NODE_NAME", "spec.nodeName"),
			newEnvVarWithFieldRef("MY_NODE_IP", "status.hostIP"),
			newEnvVarWithFieldRef("MY_POD_IP", "status.podIP"),
			newEnvVarWithFieldRef("MY_POD_NAME", "metadata.name"),
			newEnvVarWithFieldRef("MY_POD_UID", "metadata.uid"),
			newEnvVarWithFieldRef("MY_NAMESPACE", "metadata.namespace"),
			newEnvVar("HOST_PROC", "/hostfs/proc"),
			newEnvVar("HOST_SYS", "/hostfs/sys"),
			newEnvVar("HOST_ETC", "/hostfs/etc"),
			newEnvVar("HOST_VAR", "/hostfs/var"),
			newEnvVar("HOST_RUN", "/hostfs/run"),
			newEnvVar("HOST_DEV", "/hostfs/dev"),

			// TODO(splunk): add Realm and Cluster config and use it them here
			newEnvVar("SPLUNK_REALM", "replace-with-signalfx-realm"),
			newEnvVar("MY_CLUSTER_NAME", "replace-with-cluster-name"),
		}
	}

	if instance.Spec.Config == "" {
		instance.Spec.Config = defaultAgentConfig
	}
}

func newEnvVar(name, value string) v1.EnvVar {
	return v1.EnvVar{
		Name:  name,
		Value: value,
	}
}

func newEnvVarWithFieldRef(name, path string) v1.EnvVar {
	return v1.EnvVar{
		Name: name,
		ValueFrom: &v1.EnvVarSource{
			FieldRef: &v1.ObjectFieldSelector{
				APIVersion: "v1",
				FieldPath:  path,
			},
		},
	}
}

const defaultAgentConfig = `
extensions:
  health_check:
    endpoint: '0.0.0.0:13133'
  zpages:
    endpoint: '0.0.0.0:55679'
  k8s_observer:
    auth_type: serviceAccount
    node: '${MY_NODE_NAME}'
receivers:
  jaeger:
    protocols:
      grpc:
        endpoint: '0.0.0.0:14250'
      thrift_http:
        endpoint: '0.0.0.0:14268'
  otlp:
    protocols:
      grpc:
        endpoint: '0.0.0.0:4317'
      http:
        endpoint: '0.0.0.0:55681'
  zipkin:
    endpoint: '0.0.0.0:9411'
  smartagent/signalfx-forwarder:
    listenAddress: '0.0.0.0:9080'
    type: signalfx-forwarder
  signalfx:
    endpoint: '0.0.0.0:9943'
  hostmetrics:
    collection_interval: 10s
    scrapers:
      cpu: null
      disk: null
      filesystem: null
      load: null
      memory: null
      network: null
      paging: null
      processes: null
  kubeletstats:
    auth_type: serviceAccount
    collection_interval: 10s
    endpoint: '${MY_NODE_IP}:10250'
    extra_metadata_labels:
      - container.id
    metric_groups:
      - container
      - pod
      - node
  receiver_creator:
    receivers: null
    watch_observers:
      - k8s_observer
  prometheus/self:
    config:
      scrape_configs:
        - job_name: otel-agent
          scrape_interval: 10s
          static_configs:
            - targets:
                - '${MY_POD_IP}:8888'
exporters:
  sapm:
    access_token: '${SPLUNK_ACCESS_TOKEN}'
    endpoint: 'https://ingest.${SPLUNK_REALM}.signalfx.com/v2/trace'
  signalfx:
    access_token: '${SPLUNK_ACCESS_TOKEN}'
    api_url: 'https://api.${SPLUNK_REALM}.signalfx.com'
    ingest_url: 'https://ingest.${SPLUNK_REALM}.signalfx.com'
    sync_host_metadata: true
  splunk_hec:
    token: '${SPLUNK_ACCESS_TOKEN}'
    endpoint: 'https://ingest.${SPLUNK_REALM}.signalfx.com/v1/log'
  logging: null
  logging/debug:
    loglevel: debug
processors:
  k8s_tagger:
    extract:
      metadata:
        - k8s.namespace.name
        - k8s.node.name
        - k8s.pod.name
        - k8s.pod.uid
    filter:
      node: '${MY_NODE_NAME}'
  batch: null
  memory_limiter:
    ballast_size_mib: '${SPLUNK_BALLAST_SIZE_MIB}'
    check_interval: 2s
    limit_mib: '${SPLUNK_MEMORY_LIMIT_MIB}'
  resource:
    attributes:
      - action: insert
        key: k8s.node.name
        value: '${MY_NODE_NAME}'
      - action: insert
        key: k8s.cluster.name
        value: '${MY_CLUSTER_NAME}'
      - action: insert
        key: deployment.environment
        value: '${MY_CLUSTER_NAME}'
  resource/self:
    attributes:
      - action: insert
        key: k8s.pod.name
        value: '${MY_POD_NAME}'
      - action: insert
        key: k8s.pod.uid
        value: '${MY_POD_UID}'
      - action: insert
        key: k8s.namespace.name
        value: '${MY_NAMESPACE}'
  resourcedetection:
    override: false
    timeout: 10s
    detectors:
      - system
      - env
service:
  extensions:
    - health_check
    - k8s_observer
    - zpages
  pipelines:
    traces:
      receivers:
        - smartagent/signalfx-forwarder
        - otlp
        - jaeger
        - zipkin
      processors:
        - k8s_tagger
        - batch
        - resource
        - resourcedetection
      exporters:
        - sapm
        - signalfx
    metrics:
      receivers:
        - hostmetrics
        - kubeletstats
        - receiver_creator
        - signalfx
      processors:
        - batch
        - resource
        - resourcedetection
      exporters:
        - signalfx
    metrics/self:
      receivers:
        - prometheus/self
      processors:
        - batch
        - resource
        - resource/self
        - resourcedetection
      exporters:
        - signalfx
`
