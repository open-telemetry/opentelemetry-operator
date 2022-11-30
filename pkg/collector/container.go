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
	"net"
	"sort"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/mitchellh/mapstructure"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/collector/adapters"
	"github.com/open-telemetry/opentelemetry-operator/pkg/naming"
)

// maxPortLen allows us to truncate a port name according to what is considered valid port syntax:
// https://pkg.go.dev/k8s.io/apimachinery/pkg/util/validation#IsValidPortName
const maxPortLen = 15

// Container builds a container for the given collector.
func Container(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector) corev1.Container {
	image := otelcol.Spec.Image
	if len(image) == 0 {
		image = cfg.CollectorImage()
	}

	// build container ports from service ports
	ports := getConfigContainerPorts(logger, otelcol.Spec.Config)
	for _, p := range otelcol.Spec.Ports {
		ports[p.Name] = corev1.ContainerPort{
			Name:          p.Name,
			ContainerPort: p.Port,
			Protocol:      p.Protocol,
		}
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

	envVars = append(envVars, corev1.EnvVar{
		Name: "POD_NAME",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.name",
			},
		},
	})

	if otelcol.Spec.TargetAllocator.Enabled {
		// We need to add a SHARD here so the collector is able to keep targets after the hashmod operation which is
		// added by default by the Prometheus operator's config generator.
		// All collector instances use SHARD == 0 as they only receive targets
		// allocated to them and should not use the Prometheus hashmod-based
		// allocation.
		envVars = append(envVars, corev1.EnvVar{
			Name:  "SHARD",
			Value: "0",
		})
	}

	var livenessProbe *corev1.Probe
	if config, err := adapters.ConfigFromString(otelcol.Spec.Config); err == nil {
		if probe, err := adapters.ConfigToContainerProbe(config); err == nil {
			livenessProbe = probe
		}
	}

	return corev1.Container{
		Name:            naming.Container(),
		Image:           image,
		ImagePullPolicy: otelcol.Spec.ImagePullPolicy,
		Ports:           portMapToList(ports),
		VolumeMounts:    volumeMounts,
		Args:            args,
		Env:             envVars,
		EnvFrom:         otelcol.Spec.EnvFrom,
		Resources:       otelcol.Spec.Resources,
		SecurityContext: otelcol.Spec.SecurityContext,
		LivenessProbe:   livenessProbe,
	}
}

func getConfigContainerPorts(logger logr.Logger, cfg string) map[string]corev1.ContainerPort {
	ports := map[string]corev1.ContainerPort{}
	c, err := adapters.ConfigFromString(cfg)
	if err != nil {
		logger.Error(err, "couldn't extract the configuration")
		return ports
	}
	ps, err := adapters.ConfigToReceiverPorts(logger, c)
	if err != nil {
		logger.Error(err, "couldn't build container ports from configuration")
	} else {
		for _, p := range ps {
			truncName := naming.Truncate(p.Name, maxPortLen)
			if p.Name != truncName {
				logger.Info("truncating container port name",
					"port.name.prev", p.Name, "port.name.new", truncName)
			}
			nameErrs := validation.IsValidPortName(truncName)
			numErrs := validation.IsValidPortNum(int(p.Port))
			if len(nameErrs) > 0 || len(numErrs) > 0 {
				logger.Info("dropping invalid container port", "port.name", truncName, "port.num", p.Port,
					"port.name.errs", nameErrs, "num.errs", numErrs)
				continue
			}
			ports[truncName] = corev1.ContainerPort{
				Name:          truncName,
				ContainerPort: p.Port,
				Protocol:      p.Protocol,
			}
		}
	}

	metricsPort, err := getMetricsPort(c)
	if err != nil {
		logger.Info("couldn't determine metrics port from configuration, using 8888 default value", "error", err)
		metricsPort = 8888
	}
	ports["metrics"] = corev1.ContainerPort{
		Name:          "metrics",
		ContainerPort: metricsPort,
		Protocol:      corev1.ProtocolTCP,
	}
	return ports
}

// getMetricsPort gets the port number for the metrics endpoint from the collector config if it has been set.
func getMetricsPort(c map[interface{}]interface{}) (int32, error) {
	// we don't need to unmarshal the whole config, just follow the keys down to
	// the metrics address.
	type metricsCfg struct {
		Address string
	}
	type telemetryCfg struct {
		Metrics metricsCfg
	}
	type serviceCfg struct {
		Telemetry telemetryCfg
	}
	type cfg struct {
		Service serviceCfg
	}
	var cOut cfg
	err := mapstructure.Decode(c, &cOut)
	if err != nil {
		return 0, err
	}

	_, port, err := net.SplitHostPort(cOut.Service.Telemetry.Metrics.Address)
	if err != nil {
		return 0, err
	}
	i64, err := strconv.ParseInt(port, 10, 32)
	if err != nil {
		return 0, err
	}

	return int32(i64), nil
}

func portMapToList(portMap map[string]corev1.ContainerPort) []corev1.ContainerPort {
	ports := make([]corev1.ContainerPort, 0, len(portMap))
	for _, p := range portMap {
		ports = append(ports, p)
	}
	sort.Slice(ports, func(i, j int) bool {
		return ports[i].Name < ports[j].Name
	})
	return ports
}
