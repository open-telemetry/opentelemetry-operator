// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"fmt"
	"path"
	"sort"

	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/proxy"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
	"github.com/open-telemetry/opentelemetry-operator/pkg/constants"
	"github.com/open-telemetry/opentelemetry-operator/pkg/featuregate"
)

// maxPortLen allows us to truncate a port name according to what is considered valid port syntax:
// https://pkg.go.dev/k8s.io/apimachinery/pkg/util/validation#IsValidPortName
const maxPortLen = 15

// Container builds a container for the given collector.
func Container(cfg config.Config, logger logr.Logger, otelcol v1beta1.OpenTelemetryCollector, addConfig bool) corev1.Container {
	image := otelcol.Spec.Image
	if len(image) == 0 {
		image = cfg.CollectorImage()
	}

	// build container ports from service ports
	ports, err := getConfigContainerPorts(logger, otelcol.Spec.Config)
	if err != nil {
		logger.Error(err, "container ports config")
	}

	for _, p := range otelcol.Spec.Ports {
		ports[p.Name] = corev1.ContainerPort{
			Name:          p.Name,
			ContainerPort: p.Port,
			Protocol:      p.Protocol,
			HostPort:      p.HostPort,
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

	if otelcol.Spec.TargetAllocator.Enabled && cfg.CertManagerAvailability() == certmanager.Available && featuregate.EnableTargetAllocatorMTLS.IsEnabled() {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				Name:      naming.TAClientCertificate(otelcol.Name),
				MountPath: constants.TACollectorTLSDirPath,
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

	if len(otelcol.Spec.ConfigMaps) > 0 {
		for keyCfgMap := range otelcol.Spec.ConfigMaps {
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      naming.ConfigMapExtra(otelcol.Spec.ConfigMaps[keyCfgMap].Name),
				MountPath: path.Join("/var/conf", otelcol.Spec.ConfigMaps[keyCfgMap].MountPath, naming.ConfigMapExtra(otelcol.Spec.ConfigMaps[keyCfgMap].Name)),
			})
		}
	}

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

	livenessProbe, livenessProbeErr := otelcol.Spec.Config.GetLivenessProbe(logger)
	if livenessProbeErr != nil {
		logger.Error(livenessProbeErr, "cannot create liveness probe.")
	} else {
		defaultProbeSettings(livenessProbe, otelcol.Spec.LivenessProbe)
	}
	readinessProbe, readinessProbeErr := otelcol.Spec.Config.GetReadinessProbe(logger)
	if readinessProbeErr != nil {
		logger.Error(readinessProbeErr, "cannot create readiness probe.")
	} else {
		defaultProbeSettings(readinessProbe, otelcol.Spec.ReadinessProbe)
	}

	if featuregate.SetGolangFlags.IsEnabled() {
		envVars = append(envVars, corev1.EnvVar{
			Name: "GOMEMLIMIT",
			ValueFrom: &corev1.EnvVarSource{
				ResourceFieldRef: &corev1.ResourceFieldSelector{
					Resource:      "limits.memory",
					ContainerName: naming.Container(),
				},
			},
		},
			corev1.EnvVar{
				Name: "GOMAXPROCS",
				ValueFrom: &corev1.EnvVarSource{
					ResourceFieldRef: &corev1.ResourceFieldSelector{
						Resource:      "limits.cpu",
						ContainerName: naming.Container(),
					},
				},
			},
		)
	}

	if configEnvVars, err := otelcol.Spec.Config.GetEnvironmentVariables(logger); err != nil {
		logger.Error(err, "could not get the environment variables from the config")
	} else {
		envVars = append(envVars, configEnvVars...)
	}

	envVars = append(envVars, proxy.ReadProxyVarsFromEnv()...)
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
		ReadinessProbe:  readinessProbe,
		Lifecycle:       otelcol.Spec.Lifecycle,
	}
}

func getConfigContainerPorts(logger logr.Logger, conf v1beta1.Config) (map[string]corev1.ContainerPort, error) {
	ports := map[string]corev1.ContainerPort{}
	ps, err := conf.GetAllPorts(logger)
	if err != nil {
		return ports, err
	}
	if len(ps) > 0 {
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

	_, metricsPort, err := conf.Service.MetricsEndpoint(logger)
	if err != nil {
		logger.Info("couldn't determine metrics port from configuration, using 8888 default value", "error", err)
		metricsPort = 8888
	}

	ports["metrics"] = corev1.ContainerPort{
		Name:          "metrics",
		ContainerPort: metricsPort,
		Protocol:      corev1.ProtocolTCP,
	}

	return ports, nil
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

func defaultProbeSettings(probe *corev1.Probe, probeConfig *v1beta1.Probe) {
	if probe != nil && probeConfig != nil {
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
}
