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

	ports := getContainerPorts(logger, otelcol)

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

	if len(otelcol.Spec.ConfigMaps) > 0 {
		for keyCfgMap := range otelcol.Spec.ConfigMaps {
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      naming.ConfigMapExtra(otelcol.Spec.ConfigMaps[keyCfgMap].Name),
				MountPath: path.Join("/var/conf", otelcol.Spec.ConfigMaps[keyCfgMap].MountPath, naming.ConfigMapExtra(otelcol.Spec.ConfigMaps[keyCfgMap].Name)),
			})
		}
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

	return corev1.Container{
		Name:            naming.Container(),
		Image:           image,
		ImagePullPolicy: otelcol.Spec.ImagePullPolicy,
		Ports:           ports,
		VolumeMounts:    volumeMounts,
		Args:            args,
		Env:             getContainerEnvVars(otelcol, logger),
		EnvFrom:         otelcol.Spec.EnvFrom,
		Resources:       otelcol.Spec.Resources,
		SecurityContext: otelcol.Spec.SecurityContext,
		LivenessProbe:   livenessProbe,
		ReadinessProbe:  readinessProbe,
		Lifecycle:       otelcol.Spec.Lifecycle,
	}
}

func getConfigContainerPorts(logger logr.Logger, conf v1beta1.Config) ([]corev1.ContainerPort, error) {
	ports := []corev1.ContainerPort{}
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
			ports = append(ports, corev1.ContainerPort{
				Name:          truncName,
				ContainerPort: p.Port,
				Protocol:      p.Protocol,
			})
		}
	}

	_, metricsPort, err := conf.Service.MetricsEndpoint(logger)
	if err != nil {
		logger.Info("couldn't determine metrics port from configuration, using 8888 default value", "error", err)
		metricsPort = 8888
	}

	ports = append(ports, corev1.ContainerPort{
		Name:          "metrics",
		ContainerPort: metricsPort,
		Protocol:      corev1.ProtocolTCP,
	})

	return ports, nil
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

func getContainerPorts(logger logr.Logger, otelcol v1beta1.OpenTelemetryCollector) []corev1.ContainerPort {
	// build container ports from service ports
	ports, err := getConfigContainerPorts(logger, otelcol.Spec.Config)
	if err != nil {
		logger.Error(err, "container ports config")
	}

	if len(otelcol.Spec.Ports) > 0 {
		// we should add all the ports from the CR
		// there are two cases where problems might occur:
		// 1) when the port number is already being used by a receiver
		// 2) same, but for the port name
		//
		// in the first case, we remove the port we inferred from the list
		// in the second case, we rename our inferred port to something like "port-%d"
		portNumbers, portNames := extractPortNumbersAndNames(otelcol.Spec.Ports)
		var resultingInferredPorts []corev1.ContainerPort
		for _, inferred := range ports {
			if filtered := filterContainerPort(logger, inferred, portNumbers, portNames); filtered != nil {
				resultingInferredPorts = append(resultingInferredPorts, *filtered)
			}
		}
		specPorts := getSpecPorts(otelcol.Spec.Ports, logger)
		ports = append(specPorts, resultingInferredPorts...)
	}

	sort.Slice(ports, func(i, j int) bool {
		return ports[i].Name < ports[j].Name
	})
	return ports
}

// filterContainerPort filters container ports to avoid conflicts with user-specified ports.
// If the candidate port number is already in use, returns nil.
// If the candidate port name conflicts with an existing name, attempts to use a fallback name of format "port-{number}".
// If both the original name and fallback name are taken, returns nil with a warning log.
// Otherwise returns the (potentially renamed) candidate port.
func filterContainerPort(logger logr.Logger, candidate corev1.ContainerPort, portNumbers map[PortNumberKey]bool, portNames map[string]bool) *corev1.ContainerPort {
	if portNumbers[newPortNumberKey(candidate.ContainerPort, candidate.Protocol)] {
		return nil
	}

	// do we have the port name there already?
	if portNames[candidate.Name] {
		// there's already a port with the same name! do we have a 'port-%d' already?
		fallbackName := fmt.Sprintf("port-%d", candidate.ContainerPort)
		if portNames[fallbackName] {
			// that wasn't expected, better skip this port
			logger.V(2).Info("a port name specified in the CR clashes with an inferred port name, and the fallback port name clashes with another port name! Skipping this port.",
				"inferred-port-name", candidate.Name,
				"fallback-port-name", fallbackName,
			)
			return nil
		}

		candidate.Name = fallbackName
		return &candidate
	}

	// this port is unique, return as is
	return &candidate
}

func toContainerPorts(ports []v1beta1.PortsSpec) []corev1.ContainerPort {
	var containerPorts []corev1.ContainerPort
	for _, p := range ports {
		containerPorts = append(containerPorts, corev1.ContainerPort{
			Name:          p.Name,
			ContainerPort: p.Port,
			Protocol:      p.Protocol,
			HostPort:      p.HostPort,
		})
	}
	return containerPorts
}

// getSpecPorts takes a slice of PortsSpec and returns container ports, handling duplicate port names by generating unique fallback names.
// If a port name is duplicated and its fallback name is also taken, that port is skipped with a log message.
func getSpecPorts(ports []v1beta1.PortsSpec, logger logr.Logger) []corev1.ContainerPort {
	// Handle duplicate port names in otelcol.Spec.Ports
	seenNames := make(map[string]bool)
	var specPorts []corev1.ContainerPort
	for _, port := range toContainerPorts(ports) {
		if seenNames[port.Name] {
			// If we've seen this name before, generate a unique name
			fallbackName := fmt.Sprintf("port-%d", port.ContainerPort)
			if seenNames[fallbackName] {
				// If the fallback name is also taken, skip this port
				logger.V(2).Info("skipping port due to duplicate name and fallback name conflict",
					"port-name", port.Name,
					"fallback-name", fallbackName,
				)
				continue
			}
			port.Name = fallbackName
		}
		seenNames[port.Name] = true
		specPorts = append(specPorts, port)
	}
	return specPorts
}

// getContainerEnvVars returns the environment variables for the collector container.
// It combines user-defined environment variables from the OpenTelemetryCollector spec
// with automatically inferred environment variables, giving precedence to user-defined ones.
func getContainerEnvVars(otelcol v1beta1.OpenTelemetryCollector, logger logr.Logger) []corev1.EnvVar {
	inferredEnvVars := getInferredContainerEnvVars(otelcol, logger)

	envVars := []corev1.EnvVar{}
	envVars = append(envVars, otelcol.Spec.Env...)

	userDefinedEnvVars := make(map[string]bool, len(otelcol.Spec.Env))
	for _, env := range otelcol.Spec.Env {
		userDefinedEnvVars[env.Name] = true
	}

	// We only append the inferred env vars that are not defined by the user.
	for _, env := range inferredEnvVars {
		if _, ok := userDefinedEnvVars[env.Name]; !ok {
			envVars = append(envVars, env)
		}
	}

	return envVars
}

// getInferredContainerEnvVars returns environment variables that are automatically added to the collector container.
// Those include parsing the collector config and adding the env vars derived from it.
func getInferredContainerEnvVars(otelcol v1beta1.OpenTelemetryCollector, logger logr.Logger) []corev1.EnvVar {
	envVars := []corev1.EnvVar{}

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

	if featuregate.SetGolangFlags.IsEnabled() {
		envVars = append(envVars,
			corev1.EnvVar{
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

	return append(envVars, proxy.ReadProxyVarsFromEnv()...)
}
