package config

import (
	"errors"
	"flag"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	"io/fs"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	agentType             = "io.opentelemetry.remote-configuration"
	defaultConfigFilePath = "/conf/remoteconfiguration.yaml"
)

type Config struct {
	Endpoint     string   `yaml:"endpoint"`
	Protocol     string   `yaml:"protocol"`
	Capabilities []string `yaml:"capabilities"`

	// ComponentsAllowed is a list of allowed OpenTelemetry components for each pipeline type (receiver, processor, etc.)
	ComponentsAllowed map[string][]string `yaml:"components_allowed,omitempty"`
}

func (c *Config) GetCapabilities() protobufs.AgentCapabilities {
	var capabilities int32
	for _, capability := range c.Capabilities {
		// This is a helper so that we don't force consumers to prefix every agent capability
		formatted := fmt.Sprintf("AgentCapabilities_%s", capability)
		if v, ok := protobufs.AgentCapabilities_value[formatted]; ok {
			capabilities = v | capabilities
		}
	}
	return protobufs.AgentCapabilities(capabilities)
}

type CLIConfig struct {
	ListenAddr     *string
	ConfigFilePath *string
	AgentType      *string
	AgentVersion   *string

	ClusterConfig *rest.Config
	// KubeConfigFilePath empty if in cluster configuration is in use
	KubeConfigFilePath string
	RootLogger         logr.Logger
}

func Load(file string) (Config, error) {
	var cfg Config
	if err := unmarshal(&cfg, file); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func unmarshal(cfg *Config, configFile string) error {
	yamlFile, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}
	if err = yaml.UnmarshalStrict(yamlFile, cfg); err != nil {
		return fmt.Errorf("error unmarshaling YAML: %w", err)
	}
	return nil
}

func ParseCLI() (CLIConfig, error) {
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)
	agentVersion := os.Getenv("OPAMP_VERSION")
	cLIConf := CLIConfig{
		ListenAddr:     pflag.String("listen-addr", ":8080", "The address where this service serves."),
		ConfigFilePath: pflag.String("config-file", defaultConfigFilePath, "The path to the config file."),
		AgentType:      pflag.String("agent-type", agentType, "The type agent that is connecting."),
		AgentVersion:   pflag.String("agent-version", agentVersion, "The version of the agent."),
	}
	kubeconfigPath := pflag.String("kubeconfig-path", filepath.Join(homedir.HomeDir(), ".kube", "config"), "absolute path to the KubeconfigPath file")
	pflag.Parse()

	cLIConf.RootLogger = zap.New(zap.UseFlagOptions(&opts))
	klog.SetLogger(cLIConf.RootLogger)

	clusterConfig, err := clientcmd.BuildConfigFromFlags("", *kubeconfigPath)
	cLIConf.KubeConfigFilePath = *kubeconfigPath
	if err != nil {
		pathError := &fs.PathError{}
		if ok := errors.As(err, &pathError); !ok {
			return CLIConfig{}, err
		}
		clusterConfig, err = rest.InClusterConfig()
		if err != nil {
			return CLIConfig{}, err
		}
		cLIConf.KubeConfigFilePath = "" // reset as we use in cluster configuration
	}
	cLIConf.ClusterConfig = clusterConfig
	return cLIConf, nil
}
