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

package config

import (
	"errors"
	"flag"
	"io/fs"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

var (
	schemeBuilder = runtime.NewSchemeBuilder(registerKnownTypes)
)

func registerKnownTypes(s *runtime.Scheme) error {
	s.AddKnownTypes(v1alpha1.GroupVersion, &v1alpha1.OpenTelemetryCollector{}, &v1alpha1.OpenTelemetryCollectorList{})
	metav1.AddToGroupVersion(s, v1alpha1.GroupVersion)
	return nil
}

type CLIConfig struct {
	ListenAddr     *string
	ConfigFilePath *string

	ClusterConfig *rest.Config
	// KubeConfigFilePath empty if in cluster configuration is in use
	KubeConfigFilePath string
	RootLogger         logr.Logger
}

func GetLogger() logr.Logger {
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)

	return zap.New(zap.UseFlagOptions(&opts))
}

func ParseCLI(logger logr.Logger) (CLIConfig, error) {
	cLIConf := CLIConfig{
		RootLogger:     logger,
		ListenAddr:     pflag.String("listen-addr", ":8080", "The address where this service serves."),
		ConfigFilePath: pflag.String("config-file", defaultConfigFilePath, "The path to the config file."),
	}
	kubeconfigPath := pflag.String("kubeconfig-path", filepath.Join(homedir.HomeDir(), ".kube", "config"), "absolute path to the KubeconfigPath file")
	pflag.Parse()

	klog.SetLogger(logger)

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

func (cli CLIConfig) GetKubernetesClient() (client.Client, error) {
	err := schemeBuilder.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}
	return client.New(cli.ClusterConfig, client.Options{
		Scheme: scheme.Scheme,
	})
}
