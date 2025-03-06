// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"runtime"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	opampclient "github.com/open-telemetry/opamp-go/client"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/logger"
)

const (
	agentType = "io.opentelemetry.operator-opamp-bridge"
)

var (
	agentVersion  = os.Getenv("OPAMP_VERSION")
	hostname, _   = os.Hostname()
	schemeBuilder = k8sruntime.NewSchemeBuilder(registerKnownTypes)
)

func registerKnownTypes(s *k8sruntime.Scheme) error {
	s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.OpenTelemetryCollector{}, &v1alpha1.OpenTelemetryCollectorList{})
	s.AddKnownTypes(v1beta1.GroupVersion, &v1beta1.OpenTelemetryCollector{}, &v1beta1.OpenTelemetryCollectorList{})
	metav1.AddToGroupVersion(s, v1alpha1.GroupVersion)
	metav1.AddToGroupVersion(s, v1beta1.GroupVersion)
	return nil
}

func GetLogger() logr.Logger {
	return zap.New(zap.UseFlagOptions(&zapCmdLineOpts))
}

type Capability string

const (
	Unspecified                    Capability = "Unspecified"
	ReportsStatus                  Capability = "ReportsStatus"
	AcceptsRemoteConfig            Capability = "AcceptsRemoteConfig"
	ReportsEffectiveConfig         Capability = "ReportsEffectiveConfig"
	AcceptsPackages                Capability = "AcceptsPackages"
	ReportsPackageStatuses         Capability = "ReportsPackageStatuses"
	ReportsOwnTraces               Capability = "ReportsOwnTraces"
	ReportsOwnMetrics              Capability = "ReportsOwnMetrics"
	ReportsOwnLogs                 Capability = "ReportsOwnLogs"
	AcceptsOpAMPConnectionSettings Capability = "AcceptsOpAMPConnectionSettings"
	AcceptsOtherConnectionSettings Capability = "AcceptsOtherConnectionSettings"
	AcceptsRestartCommand          Capability = "AcceptsRestartCommand"
	ReportsHealth                  Capability = "ReportsHealth"
	ReportsRemoteConfig            Capability = "ReportsRemoteConfig"
)

type Config struct {
	// KubeConfigFilePath is empty if InClusterConfig() should be used, otherwise it's a path to where a valid
	// kubernetes configuration file.
	KubeConfigFilePath string       `yaml:"kubeConfigFilePath,omitempty"`
	ListenAddr         string       `yaml:"listenAddr,omitempty"`
	ClusterConfig      *rest.Config `yaml:"-"`
	RootLogger         logr.Logger  `yaml:"-"`

	// ComponentsAllowed is a list of allowed OpenTelemetry components for each pipeline type (receiver, processor, etc.)
	ComponentsAllowed map[string][]string `yaml:"componentsAllowed,omitempty"`
	Endpoint          string              `yaml:"endpoint"`
	Headers           Headers             `yaml:"headers,omitempty"`
	Capabilities      map[Capability]bool `yaml:"capabilities"`
	HeartbeatInterval time.Duration       `yaml:"heartbeatInterval,omitempty"`
	Name              string              `yaml:"name,omitempty"`
}

func NewConfig(logger logr.Logger) *Config {
	return &Config{
		RootLogger: logger,
	}
}

func (c *Config) CreateClient() opampclient.OpAMPClient {
	opampLogger := logger.NewLogger(c.RootLogger.WithName("client"))
	agentScheme := c.GetAgentScheme()
	if agentScheme == "http" || agentScheme == "https" {
		return opampclient.NewHTTP(opampLogger)
	}
	return opampclient.NewWebSocket(opampLogger)
}

func (c *Config) GetComponentsAllowed() map[string]map[string]bool {
	m := make(map[string]map[string]bool)
	for component, componentSet := range c.ComponentsAllowed {
		if _, ok := m[component]; !ok {
			m[component] = make(map[string]bool)
		}
		for _, s := range componentSet {
			m[component][s] = true
		}
	}
	return m
}

func (c *Config) GetCapabilities() protobufs.AgentCapabilities {
	var capabilities int32
	for capability, enabled := range c.Capabilities {
		if !enabled {
			continue
		}
		// This is a helper so that we don't force consumers to prefix every agent capability
		formatted := fmt.Sprintf("AgentCapabilities_%s", capability)
		if v, ok := protobufs.AgentCapabilities_value[formatted]; ok {
			capabilities = v | capabilities
		}
	}
	return protobufs.AgentCapabilities(capabilities)
}

func (c *Config) GetAgentScheme() string {
	uri, err := url.ParseRequestURI(c.Endpoint)
	if err != nil {
		return ""
	}
	return uri.Scheme
}

func (c *Config) GetAgentType() string {
	return agentType
}

func (c *Config) GetAgentVersion() string {
	return agentVersion
}

func (c *Config) GetDescription() *protobufs.AgentDescription {
	return &protobufs.AgentDescription{
		IdentifyingAttributes: []*protobufs.KeyValue{
			keyValuePair("service.name", c.GetAgentType()),
			keyValuePair("service.version", c.GetAgentVersion()),
		},
		NonIdentifyingAttributes: []*protobufs.KeyValue{
			keyValuePair("os.family", runtime.GOOS),
			keyValuePair("host.name", hostname),
		},
	}
}

func keyValuePair(key string, value string) *protobufs.KeyValue {
	return &protobufs.KeyValue{
		Key: key,
		Value: &protobufs.AnyValue{
			Value: &protobufs.AnyValue_StringValue{
				StringValue: value,
			},
		},
	}
}

func (c *Config) GetNewInstanceId() uuid.UUID {
	u, err := uuid.NewV7()
	if err != nil {
		// This really should never happen and if it does we should fail.
		panic(err)
	}
	return u
}

func (c *Config) RemoteConfigEnabled() bool {
	capabilities := c.GetCapabilities()
	return capabilities&protobufs.AgentCapabilities_AgentCapabilities_AcceptsRemoteConfig != 0
}

func (c *Config) GetKubernetesClient() (client.Client, error) {
	err := schemeBuilder.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}
	return client.New(c.ClusterConfig, client.Options{
		Scheme: scheme.Scheme,
	})
}

func Load(logger logr.Logger, flagSet *pflag.FlagSet) (*Config, error) {
	cfg := NewConfig(logger)

	// load the config from the config file
	configFilePath, err := getConfigFilePath(flagSet)
	if err != nil {
		return nil, err
	}
	err = LoadFromFile(cfg, configFilePath)
	if err != nil {
		return nil, err
	}

	err = LoadFromCLI(cfg, flagSet)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func LoadFromCLI(target *Config, flagSet *pflag.FlagSet) error {
	var err error
	klog.SetLogger(target.RootLogger)
	ctrl.SetLogger(target.RootLogger)

	target.KubeConfigFilePath, err = getKubeConfigFilePath(flagSet)
	if err != nil {
		return err
	}
	clusterConfig, err := clientcmd.BuildConfigFromFlags("", target.KubeConfigFilePath)
	if err != nil {
		pathError := &fs.PathError{}
		if ok := errors.As(err, &pathError); !ok {
			return err
		}
		clusterConfig, err = rest.InClusterConfig()
		if err != nil {
			return err
		}
	}
	target.ClusterConfig = clusterConfig

	target.ListenAddr, err = getListenAddr(flagSet)
	if err != nil {
		return err
	}

	target.HeartbeatInterval, err = getHeartbeatInterval(flagSet)
	if err != nil {
		return err
	}

	target.Name, err = getName(flagSet)
	if err != nil {
		return err
	}

	return nil
}

func LoadFromFile(cfg *Config, configFile string) error {
	yamlFile, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}
	envExpandedYaml := []byte(os.ExpandEnv(string(yamlFile)))
	if err = yaml.Unmarshal(envExpandedYaml, cfg); err != nil {
		return fmt.Errorf("error unmarshaling YAML: %w", err)
	}
	return nil
}
