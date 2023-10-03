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

package collector

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

var (
	// backoffLimit is set to one because we don't need to retry this job, it either fails or succeeds.
	backoffLimit int32 = 1
)

func Job(params manifests.Params) *batchv1.Job {
	confMapSha := GetConfigMapSHA(params.OtelCol.Spec.Config)
	name := naming.Job(params.OtelCol.Name, confMapSha)
	labels := Labels(params.OtelCol, name, params.Config.LabelsFilter())

	annotations := Annotations(params.OtelCol)
	podAnnotations := PodAnnotations(params.OtelCol)
	// manualSelector is explicitly false because we don't want to cause a potential conflict between the job
	// and the replicaset
	manualSelector := false

	c := Container(params.Config, params.Log, params.OtelCol, true)
	c.Args = append([]string{"validate"}, c.Args...)

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   params.OtelCol.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: batchv1.JobSpec{
			ManualSelector: &manualSelector,
			BackoffLimit:   &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: podAnnotations,
				},
				Spec: corev1.PodSpec{
					RestartPolicy:                 corev1.RestartPolicyNever,
					ServiceAccountName:            ServiceAccountName(params.OtelCol),
					InitContainers:                params.OtelCol.Spec.InitContainers,
					Containers:                    append(params.OtelCol.Spec.AdditionalContainers, c),
					Volumes:                       Volumes(params.Config, params.OtelCol, naming.VersionedConfigMap(params.OtelCol.Name, confMapSha)),
					DNSPolicy:                     getDNSPolicy(params.OtelCol),
					HostNetwork:                   params.OtelCol.Spec.HostNetwork,
					Tolerations:                   params.OtelCol.Spec.Tolerations,
					NodeSelector:                  params.OtelCol.Spec.NodeSelector,
					SecurityContext:               params.OtelCol.Spec.PodSecurityContext,
					PriorityClassName:             params.OtelCol.Spec.PriorityClassName,
					Affinity:                      params.OtelCol.Spec.Affinity,
					TerminationGracePeriodSeconds: params.OtelCol.Spec.TerminationGracePeriodSeconds,
					TopologySpreadConstraints:     params.OtelCol.Spec.TopologySpreadConstraints,
				},
			},
		},
	}
}
