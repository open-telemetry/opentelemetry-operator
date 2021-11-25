package adapters

import (
	"errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strings"
)

var (
	errNoService    = errors.New("no service available as part of the configuration")
	errNoExtensions = errors.New("no extensions available as part of the configuration")

	errServiceNotAMap    = errors.New("service property in the configuration doesn't contain valid services")
	errExtensionsNotAMap = errors.New("extensions property in the configuration doesn't contain valid extensions")

	errNoExtensionHealthCheck = errors.New("extensions property in the configuration does not contain the expected health_check extension")

	errNoServiceExtensions = errors.New("service property in the configuration doesn't contain extensions")

	errServiceExtensionsNotSlice     = errors.New("service extensions property in the configuration does not contain valid extensions")
	errNoServiceExtensionHealthCheck = errors.New("no healthcheck extension available in service extension configuration")
)

type probeConfiguration struct {
	path string
	port intstr.IntOrString
}

const (
	defaultHealthCheckPath = "/"
	defaultHealthCheckPort = 13133
)

// ConfigToContainerProbe converts the incoming configuration object into a container probe or returns an error
func ConfigToContainerProbe(config map[interface{}]interface{}) (*corev1.Probe, error) {
	serviceProperty, ok := config["service"]
	if !ok {
		return nil, errNoService
	}
	service, ok := serviceProperty.(map[interface{}]interface{})
	if !ok {
		return nil, errServiceNotAMap
	}

	serviceExtensionsProperty, ok := service["extensions"]
	if !ok {
		return nil, errNoServiceExtensions
	}

	serviceExtensions, ok := serviceExtensionsProperty.([]interface{})
	if !ok {
		return nil, errServiceExtensionsNotSlice
	}
	healthCheckServiceExtensions := make([]string, 0)
	for _, ext := range serviceExtensions {
		parsedExt, ok := ext.(string)
		if ok && strings.HasPrefix(parsedExt, "health_check") {
			healthCheckServiceExtensions = append(healthCheckServiceExtensions, parsedExt)
		}
	}

	if len(healthCheckServiceExtensions) == 0 {
		return nil, errNoServiceExtensionHealthCheck
	}

	extensionsProperty, ok := config["extensions"]
	if !ok {
		return nil, errNoExtensions
	}
	extensions, ok := extensionsProperty.(map[interface{}]interface{})
	if !ok {
		return nil, errExtensionsNotAMap
	}
	// in the event of multiple health_check service extensions defined, we arbitrarily take the first one found
	for _, healthCheckForProbe := range healthCheckServiceExtensions {
		healthCheckExtension, ok := extensions[healthCheckForProbe]
		if ok {
			return createProbeFromExtension(healthCheckExtension)
		}
	}

	return nil, errNoExtensionHealthCheck
}

func createProbeFromExtension(extension interface{}) (*corev1.Probe, error) {
	probeCfg := extractProbeConfigurationFromExtension(extension)
	return &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: probeCfg.path,
				Port: probeCfg.port,
			},
		},
	}, nil
}

func extractProbeConfigurationFromExtension(ext interface{}) probeConfiguration {
	extensionCfg, ok := ext.(map[interface{}]interface{})
	if !ok {
		return defaultProbeConfiguration()
	}
	return probeConfiguration{
		path: extractPathFromExtensionConfig(extensionCfg),
		port: extractPortFromExtensionConfig(extensionCfg),
	}
}

func defaultProbeConfiguration() probeConfiguration {
	return probeConfiguration{
		path: defaultHealthCheckPath,
		port: intstr.FromInt(defaultHealthCheckPort),
	}
}

func extractPathFromExtensionConfig(cfg map[interface{}]interface{}) string {
	if path, ok := cfg["path"]; ok {
		if parsedPath, ok := path.(string); ok {
			return parsedPath
		}
	}
	return defaultHealthCheckPath
}

func extractPortFromExtensionConfig(cfg map[interface{}]interface{}) intstr.IntOrString {
	endpoint, ok := cfg["endpoint"]
	if !ok {
		return defaultHealthCheckEndpoint()
	}
	parsedEndpoint, ok := endpoint.(string)
	if !ok {
		return defaultHealthCheckEndpoint()
	}
	endpointComponents := strings.Split(parsedEndpoint, ":")
	if len(endpointComponents) != 2 {
		return defaultHealthCheckEndpoint()
	}
	return intstr.Parse(endpointComponents[1])
}

func defaultHealthCheckEndpoint() intstr.IntOrString {
	return intstr.FromInt(defaultHealthCheckPort)
}
