package adapters

import (
	"errors"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strings"
)

var (
	// ErrNoService indicates that there is no service in the configuration.
	ErrNoService = errors.New("no service available as part of the configuration")
	// ErrNoExtensions indicates that there are no extensions in the configuration
	ErrNoExtensions = errors.New("no extensions available as part of the configuration")

	// ErrServiceNotAMap indicates that the service property isn't a map of values.
	ErrServiceNotAMap = errors.New("service property in the configuration doesn't contain valid services")
	// ErrExtensionsNotAMap indicates that the extensions property isn't a map of values.
	ErrExtensionsNotAMap = errors.New("extensions property in the configuration doesn't contain valid extensions")

	// ErrNoExtensionHealthCheck indicates no health_check extension was found for the
	ErrNoExtensionHealthCheck = errors.New("extensions property in the configuration does not contain the expected health_check extension")

	// ErrNoServiceExtensions indicates the service configuration does not contain any extensions
	ErrNoServiceExtensions = errors.New("service property in the configuration doesn't contain extensions")

	// ErrServiceExtensionsNotSlice indicates the service extensions property isn't a slice as expected
	ErrServiceExtensionsNotSlice = errors.New("service extensions property in the configuration does not contain valid extensions")

	// ErrNoServiceExtensionHealthCheck indicates no health_check
	ErrNoServiceExtensionHealthCheck = errors.New("no healthcheck extension available in service extension configuration")
)

// ConfigToContainerProbe converts the incoming configuration object into a container probe or returns an error
func ConfigToContainerProbe(logger logr.Logger, config map[interface{}]interface{}) (*corev1.Probe, error) {
	serviceProperty, ok := config["service"]
	if !ok {
		return nil, ErrNoService
	}
	service, ok := serviceProperty.(map[interface{}]interface{})
	if !ok {
		return nil, ErrServiceNotAMap
	}

	serviceExtensionsProperty, ok := service["extensions"]
	if !ok {
		return nil, ErrNoServiceExtensions
	}

	healthcheckForProbe := ""

	serviceExtensions, ok := serviceExtensionsProperty.([]interface{})
	if !ok {
		return nil, ErrServiceExtensionsNotSlice
	}
	// in the event of multiple health_check extensions defined, we arbitrarily take the first one found
	for _, ext := range serviceExtensions {
		parsedExt, ok := ext.(string)
		if ok && strings.HasPrefix(parsedExt, "health_check") {
			healthcheckForProbe = parsedExt
			break
		}
	}

	if healthcheckForProbe == "" {
		return nil, ErrNoServiceExtensionHealthCheck
	}

	extensionsProperty, ok := config["extensions"]
	if !ok {
		return nil, ErrNoExtensions
	}
	extensions, ok := extensionsProperty.(map[interface{}]interface{})
	if !ok {
		return nil, ErrExtensionsNotAMap
	}
	healthcheckExtension, ok := extensions[healthcheckForProbe]
	if !ok {
		return nil, ErrNoExtensionHealthCheck
	}

	return createProbeFromExtension(healthcheckExtension)
}

func createProbeFromExtension(extension interface{}) (*corev1.Probe, error) {
	probeCfg := extractProbeConfigurationFromExtension(extension)
	return &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: probeCfg.path,
				Port: probeCfg.port,
				Host: probeCfg.host,
			},
		},
	}, nil
}

type probeConfiguration struct {
	path string
	port intstr.IntOrString
	host string
}

const (
	defaultHealthCheckPath = "/"
	defaultHealthCheckPort = 13133
	defaultHealthCheckHost = "0.0.0.0"
)

func extractProbeConfigurationFromExtension(ext interface{}) probeConfiguration {
	extensionCfg, ok := ext.(map[interface{}]interface{})
	if !ok {
		return defaultProbeConfiguration()
	}
	endpoint := extractEndpointFromExtensionConfig(extensionCfg)
	return probeConfiguration{
		path: extractPathFromExtensionConfig(extensionCfg),
		port: endpoint.port,
		host: endpoint.host,
	}
}

func defaultProbeConfiguration() probeConfiguration {
	return probeConfiguration{
		path: defaultHealthCheckPath,
		port: intstr.FromInt(defaultHealthCheckPort),
		host: defaultHealthCheckHost,
	}
}

type healthCheckEndpoint struct {
	port intstr.IntOrString
	host string
}

func defaultHealthCheckEndpoint() healthCheckEndpoint {
	defaultProbe := defaultProbeConfiguration()
	return healthCheckEndpoint{
		port: defaultProbe.port,
		host: defaultProbe.host,
	}
}

func extractEndpointFromExtensionConfig(cfg map[interface{}]interface{}) healthCheckEndpoint {
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
	return healthCheckEndpoint{
		port: intstr.Parse(endpointComponents[1]),
		host: endpointComponents[0],
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
