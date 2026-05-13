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
	"strings"
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
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/logger"
)

const (
	agentType      = "io.opentelemetry.operator-opamp-bridge"
	operatorMode   = "operator"
	standaloneMode = "standalone"
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
	instanceId         uuid.UUID    `yaml:"-"`

	// ComponentsAllowed is a list of allowed OpenTelemetry components for each pipeline type (receiver, processor, etc.)
	ComponentsAllowed map[string][]string `yaml:"componentsAllowed,omitempty"`
	Endpoint          string              `yaml:"endpoint"`
	Headers           Headers             `yaml:"headers,omitempty"`
	Capabilities      map[Capability]bool `yaml:"capabilities"`
	HeartbeatInterval time.Duration       `yaml:"heartbeatInterval,omitempty"`
	Name              string              `yaml:"name,omitempty"`
	AgentDescription  AgentDescription    `yaml:"description,omitempty"`
	Standalone        StandaloneConfig    `yaml:"standalone,omitempty"`

	// Mode selects the operating mode: "operator" (default) uses OpenTelemetryCollector CRDs,
	// "standalone" manages static Kubernetes config sources from this config.
	Mode string `yaml:"mode,omitempty"`
}

// AgentDescription is copied from the OpAMP Extension in the collector.
// https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/ccc3e6ed6386d404eb4beddd257ff979d2a346f4/extension/opampextension/config.go#L48
type AgentDescription struct {
	// NonIdentifyingAttributes are a map of key-value pairs that may be specified to provide
	// extra information about the agent to the OpAMP server.
	NonIdentifyingAttributes map[string]string `yaml:"non_identifying_attributes"`
}

type StandaloneConfig struct {
	Agents []StandaloneAgentConfig `yaml:"agents,omitempty"`
}

type StandaloneAgentConfig struct {
	Name      string                           `yaml:"name"`
	Namespace string                           `yaml:"namespace"`
	Type      string                           `yaml:"type"`
	Config    map[string]StandaloneConfigEntry `yaml:"config"`
}

type StandaloneConfigEntry struct {
	Kind      string `yaml:"kind"`
	Namespace string `yaml:"namespace"`
	Name      string `yaml:"name"`
	Key       string `yaml:"key"`
}

func NewConfig(logger logr.Logger) *Config {
	return &Config{
		instanceId:         mustGetInstanceId(),
		Name:               opampBridgeName,
		ListenAddr:         defaultServerListenAddr,
		HeartbeatInterval:  defaultHeartbeatInterval,
		KubeConfigFilePath: defaultKubeConfigPath,
		RootLogger:         logger,
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
	if c.Name != opampBridgeName && c.Mode == standaloneMode {
		return c.Name
	}
	return agentType
}

func (*Config) GetAgentVersion() string {
	return agentVersion
}

func (c *Config) GetInstanceId() uuid.UUID {
	return c.instanceId
}

func (c *Config) GetDescription() *protobufs.AgentDescription {
	return &protobufs.AgentDescription{
		IdentifyingAttributes: []*protobufs.KeyValue{
			keyValuePair("service.name", c.GetAgentType()),
			keyValuePair("service.instance.id", c.GetInstanceId().String()),
			keyValuePair("service.version", c.GetAgentVersion()),
		},
		NonIdentifyingAttributes: append(
			c.AgentDescription.nonIdentifyingAttributes(),
			keyValuePair("os.family", runtime.GOOS),
			keyValuePair("host.name", hostname),
		),
	}
}

func (c *Config) ForStandaloneAgent(agent StandaloneAgentConfig) *Config {
	agentConfig := *c
	agentConfig.Name = agent.Name
	agentConfig.instanceId = uuid.NewSHA1(uuid.NameSpaceURL, []byte(fmt.Sprintf("%s/%s/%s", agent.Namespace, agent.Name, agent.Type)))
	agentConfig.AgentDescription.NonIdentifyingAttributes = cloneStringMap(c.AgentDescription.NonIdentifyingAttributes)
	if agentConfig.AgentDescription.NonIdentifyingAttributes == nil {
		agentConfig.AgentDescription.NonIdentifyingAttributes = map[string]string{}
	}
	agentConfig.AgentDescription.NonIdentifyingAttributes["k8s.namespace.name"] = agent.Namespace
	agentConfig.AgentDescription.NonIdentifyingAttributes["opentelemetry.io/agent.name"] = agent.Name
	agentConfig.AgentDescription.NonIdentifyingAttributes["opentelemetry.io/agent.type"] = agent.Type
	return &agentConfig
}

func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (c *Config) Validate() error {
	if !c.IsStandaloneMode() {
		return nil
	}
	if len(c.Standalone.Agents) == 0 {
		return errors.New("standalone mode requires at least one configured agent")
	}
	agents := map[string]struct{}{}
	for _, agent := range c.Standalone.Agents {
		if strings.TrimSpace(agent.Name) == "" {
			return errors.New("standalone agent name is required")
		}
		if _, ok := agents[agent.Name]; ok {
			return fmt.Errorf("duplicate standalone agent name %q", agent.Name)
		}
		agents[agent.Name] = struct{}{}
		if strings.TrimSpace(agent.Namespace) == "" {
			return fmt.Errorf("standalone agent %q namespace is required", agent.Name)
		}
		if strings.TrimSpace(agent.Type) == "" {
			return fmt.Errorf("standalone agent %q type is required", agent.Name)
		}
		if len(agent.Config) == 0 {
			return fmt.Errorf("standalone agent %q requires at least one config entry", agent.Name)
		}
		for remoteName, entry := range agent.Config {
			if strings.TrimSpace(remoteName) == "" {
				return fmt.Errorf("standalone agent %q config remote name is required", agent.Name)
			}
			if strings.ToLower(entry.Kind) != "configmap" {
				return fmt.Errorf("standalone agent %q config %q has unsupported kind %q", agent.Name, remoteName, entry.Kind)
			}
			if strings.TrimSpace(entry.Namespace) == "" {
				return fmt.Errorf("standalone agent %q config %q namespace is required", agent.Name, remoteName)
			}
			if strings.TrimSpace(entry.Name) == "" {
				return fmt.Errorf("standalone agent %q config %q name is required", agent.Name, remoteName)
			}
			if strings.TrimSpace(entry.Key) == "" {
				return fmt.Errorf("standalone agent %q config %q key is required", agent.Name, remoteName)
			}
		}
	}
	return nil
}

func (ad *AgentDescription) nonIdentifyingAttributes() []*protobufs.KeyValue {
	toReturn := make([]*protobufs.KeyValue, len(ad.NonIdentifyingAttributes))
	i := 0
	for k, v := range ad.NonIdentifyingAttributes {
		toReturn[i] = keyValuePair(k, v)
		i++
	}
	return toReturn
}

func keyValuePair(key, value string) *protobufs.KeyValue {
	return &protobufs.KeyValue{
		Key: key,
		Value: &protobufs.AnyValue{
			Value: &protobufs.AnyValue_StringValue{
				StringValue: value,
			},
		},
	}
}

func mustGetInstanceId() uuid.UUID {
	u, err := uuid.NewV7()
	if err != nil {
		// This really should never happen and if it does, we should fail.
		panic(err)
	}
	return u
}

func (c *Config) GetNewInstanceId() uuid.UUID {
	c.instanceId = mustGetInstanceId()
	return c.instanceId
}

func (c *Config) RemoteConfigEnabled() bool {
	capabilities := c.GetCapabilities()
	return capabilities&protobufs.AgentCapabilities_AgentCapabilities_AcceptsRemoteConfig != 0
}

func (c *Config) GetKubernetesClient() (client.Client, error) {
	if c.Mode != standaloneMode {
		err := schemeBuilder.AddToScheme(scheme.Scheme)
		if err != nil {
			return nil, err
		}
	}
	return client.New(c.ClusterConfig, client.Options{
		Scheme: scheme.Scheme,
	})
}

func (c *Config) IsStandaloneMode() bool {
	return c.Mode == standaloneMode
}

func (c *Config) GetRestConfig() *rest.Config {
	return c.ClusterConfig
}

func Load(logger logr.Logger, args []string) (*Config, error) {
	flagSet := GetFlagSet(pflag.ExitOnError)
	err := flagSet.Parse(args)
	if err != nil {
		return nil, err
	}
	cfg := NewConfig(logger)
	configFilePath := defaultConfigFilePath
	// load the config from the config file
	configFilePathByFlag, changed, err := getConfigFilePath(flagSet)
	if err != nil {
		return nil, err
	}
	if changed {
		configFilePath = configFilePathByFlag
	}
	err = LoadFromFile(cfg, configFilePath)
	if err != nil {
		return nil, err
	}

	err = LoadFromCLI(cfg, flagSet)
	if err != nil {
		return nil, err
	}
	if err = cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func LoadFromCLI(target *Config, flagSet *pflag.FlagSet) error {
	klog.SetLogger(target.RootLogger)
	ctrl.SetLogger(target.RootLogger)

	if kubeConfigFilePath, changed, err := getKubeConfigFilePath(flagSet); err != nil {
		return err
	} else if changed {
		target.KubeConfigFilePath = kubeConfigFilePath
	}
	clusterConfig, errBuildFromConfig := clientcmd.BuildConfigFromFlags("", target.KubeConfigFilePath)
	if errBuildFromConfig != nil {
		pathError := &fs.PathError{}
		if ok := errors.As(errBuildFromConfig, &pathError); !ok {
			return errBuildFromConfig
		}
		clusterConfig, errBuildFromConfig = rest.InClusterConfig()
		if errBuildFromConfig != nil {
			return errBuildFromConfig
		}
	}
	target.ClusterConfig = clusterConfig

	if listenAddr, changed, err := getListenAddr(flagSet); err != nil {
		return err
	} else if changed {
		target.ListenAddr = listenAddr
	}
	if heartbeatInterval, changed, err := getHeartbeatInterval(flagSet); err != nil {
		return err
	} else if changed {
		target.HeartbeatInterval = heartbeatInterval
	}
	if name, changed, err := getName(flagSet); err != nil {
		return err
	} else if changed {
		target.Name = name
	}
	if mode, changed, err := getMode(flagSet); err != nil {
		return err
	} else if changed {
		target.Mode = mode
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
