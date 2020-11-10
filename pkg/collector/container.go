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
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
)

// Container builds a container for the given collector
func Container(cfg *config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) corev1.Container {
	var image string
	var cmd []string
	if len(otelcol.Spec.DistributionName) > 0 {
		distribution := cfg.Distribution(otelcol.Namespace, otelcol.Spec.DistributionName)
		if distribution != nil {
			image = distribution.Image
			cmd = distribution.Command
		} else {
			logger.V(1).Info("the requested distribution couldn't be found", "namespace", otelcol.Namespace, "distributionName", otelcol.Spec.DistributionName)
		}
	} else {
		image = otelcol.Spec.Image
	}

	if len(image) == 0 {
		image = cfg.CollectorImage()
	}

	argsMap := otelcol.Spec.Args
	if argsMap == nil {
		argsMap = map[string]string{}
	}

	if _, exists := argsMap["config"]; exists {
		logger.Info("the 'config' flag isn't allowed and is being ignored")
	}

	// this effectively overrides any 'config' entry that might exist in the CR
	argsMap["config"] = fmt.Sprintf("/conf/%s", cfg.CollectorConfigMapEntry())

	var args []string
	for k, v := range argsMap {
		args = append(args, fmt.Sprintf("--%s=%s", k, v))
	}

	volumeMounts := []corev1.VolumeMount{{
		Name:      naming.ConfigMapVolume(),
		MountPath: "/conf",
	}}

	if len(otelcol.Spec.VolumeMounts) > 0 {
		volumeMounts = append(volumeMounts, otelcol.Spec.VolumeMounts...)
	}

	var envVars = otelcol.Spec.Env
	if otelcol.Spec.Env == nil {
		envVars = []corev1.EnvVar{}
	}

	container := corev1.Container{
		Name:         naming.Container(),
		Image:        image,
		VolumeMounts: volumeMounts,
		Args:         args,
		Env:          envVars,
	}

	if len(cmd) > 0 {
		container.Command = cmd
	}

	return container
}
