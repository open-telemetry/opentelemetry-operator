package config

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Config struct {
	CollectionDir    string
	KubernetesClient client.Client
}

func NewConfig(scheme *runtime.Scheme) (Config, error) {
	var kubeconfigPath string
	var collectionDir string

	pflag.StringVar(&kubeconfigPath, "kubeconfig-path", filepath.Join(homedir.HomeDir(), ".kube", "config"), "Absolute path to the KubeconfigPath file")
	pflag.StringVar(&collectionDir, "collection-dir", filepath.Join(homedir.HomeDir(), "must-gather"), "Absolute path to the KubeconfigPath file")
	pflag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return Config{}, fmt.Errorf("Error reading the kubeconfig: %s\n", err.Error())
	}

	clusterClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return Config{}, fmt.Errorf("Creating the Kubernetes client: %s\n", err)
	}

	return Config{
		CollectionDir:    collectionDir,
		KubernetesClient: clusterClient,
	}, nil
}
