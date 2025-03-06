// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Config struct {
	CollectionDir       string
	OperatorName        string
	OperatorNamespace   string
	KubernetesClient    client.Client
	KubernetesClientSet *kubernetes.Clientset
}

func NewConfig(scheme *runtime.Scheme) (Config, error) {
	var operatorName, operatorNamespace, collectionDir, kubeconfigPath string

	pflag.StringVar(&operatorName, "operator-name", "opentelemetry-operator", "Operator name")
	pflag.StringVar(&operatorNamespace, "operator-namespace", "", "Namespace where the operator was deployed")
	pflag.StringVar(&collectionDir, "collection-dir", filepath.Join(homedir.HomeDir(), "/must-gather"), "Absolute path to the KubeconfigPath file")
	pflag.StringVar(&kubeconfigPath, "kubeconfig", "", "Path to the kubeconfig file")
	pflag.Parse()

	config, err := rest.InClusterConfig()
	if err != nil {
		if kubeconfigPath == "" {
			kubeconfigPath = filepath.Join(homedir.HomeDir(), ".kube", "config")
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return Config{}, fmt.Errorf("failed to create Kubernetes client config: %w", err)
		}
	}

	clusterClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return Config{}, fmt.Errorf("creating the Kubernetes client: %w\n", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return Config{}, fmt.Errorf("creating the Kubernetes clienset: %w\n", err)
	}

	return Config{
		CollectionDir:       collectionDir,
		KubernetesClient:    clusterClient,
		KubernetesClientSet: clientset,
		OperatorName:        operatorName,
		OperatorNamespace:   operatorNamespace,
	}, nil
}
