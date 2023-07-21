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
func Container(cfg config.Config, logger logr.Logger, otelcol v1alpha1.OpenTelemetryCollector, addConfig bool) corev1.Container {
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

	var volumeMounts []corev1.VolumeMount
	argsMap := otelcol.Spec.Args
	if argsMap == nil {
		argsMap = map[string]string{}
	}
	// defines the output (sorted) array for final output
	var args []string
	// When adding a config via v1alpha1.OpenTelemetryCollectorSpec.Config, we ensure that it is always the
	// first item in the args. At the time of writing, although multiple configs are allowed in the
	// opentelemetry collector, the operator has yet to implement such functionality.  When multiple configs
	// are present they should be merged in a deterministic manner using the order given, and because
	// v1alpha1.OpenTelemetryCollectorSpec.Config is a required field we assume that it will always be the
	// "primary" config and in the future additional configs can be appended to the container args in a simple manner.
	if addConfig {
		// if key exists then delete key and excluded from the iteration after this block
		if _, exists := argsMap["config"]; exists {
			logger.Info("the 'config' flag isn't allowed and is being ignored")
			delete(argsMap, "config")
		}
		args = append(args, fmt.Sprintf("--config=/conf/%s", cfg.CollectorConfigMapEntry()))
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				Name:      naming.ConfigMapVolume(),
				MountPath: "/conf",
			})
	}

	// ensure that the v1alpha1.OpenTelemetryCollectorSpec.Args are ordered when moved to container.Args,
	// where iterating over a map does not guarantee, so that reconcile will not be fooled by different
	// ordering in args.
	var sortedArgs []string
	for k, v := range argsMap {
		sortedArgs = append(sortedArgs, fmt.Sprintf("--%s=%s", k, v))
	}
	sort.Strings(sortedArgs)
	args = append(args, sortedArgs...)

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
		if probe, err := getLivenessProbe(config, otelcol.Spec.LivenessProbe); err == nil {
			livenessProbe = probe
		} else {
			logger.Error(err, "Cannot create liveness probe.")
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
		Lifecycle:       otelcol.Spec.Lifecycle,
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

	promExporterPort, err := getPrometheusExporterPort(c)
	if err != nil {
		logger.V(2).Info("prometheus exporter port not detected")
	} else {
		ports["promexporter"] = corev1.ContainerPort{
			Name:          "promexporter",
			ContainerPort: promExporterPort,
			Protocol:      corev1.ProtocolTCP,
		}
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

func getPrometheusExporterPort(c map[interface{}]interface{}) (int32, error) {
	// we don't need to unmarshal the whole config, just follow the keys down to
	// the prometheus endpoint.
	type prometheusCfg struct {
		Endpoint string
	}
	type exportersCfg struct {
		Prometheus prometheusCfg
	}
	type cfg struct {
		Exporters exportersCfg
	}
	var cOut cfg
	err := mapstructure.Decode(c, &cOut)
	if err != nil {
		return 0, nil
	}

	_, port, err := net.SplitHostPort(cOut.Exporters.Prometheus.Endpoint)
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

func getLivenessProbe(config map[interface{}]interface{}, probeConfig *v1alpha1.Probe) (*corev1.Probe, error) {
	probe, err := adapters.ConfigToContainerProbe(config)
	if err != nil {
		return nil, err
	}
	if probeConfig != nil {
		if probeConfig.InitialDelaySeconds != nil {
			probe.InitialDelaySeconds = *probeConfig.InitialDelaySeconds
		}
		if probeConfig.PeriodSeconds != nil {
			probe.PeriodSeconds = *probeConfig.PeriodSeconds
		}
		if probeConfig.FailureThreshold != nil {
			probe.FailureThreshold = *probeConfig.FailureThreshold
		}
		if probeConfig.SuccessThreshold != nil {
			probe.SuccessThreshold = *probeConfig.SuccessThreshold
		}
		if probeConfig.TimeoutSeconds != nil {
			probe.TimeoutSeconds = *probeConfig.TimeoutSeconds
		}
		probe.TerminationGracePeriodSeconds = probeConfig.TerminationGracePeriodSeconds
	}
	return probe, nil
}
