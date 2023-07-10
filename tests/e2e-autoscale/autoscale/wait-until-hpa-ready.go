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

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	var hpaName string
	var timeout time.Duration
	var kubeconfigPath string

	defaultKubeconfigPath := filepath.Join(homedir.HomeDir(), ".kube", "config")

	pflag.DurationVar(&timeout, "timeout", 5*time.Minute, "The timeout for the check.")
	pflag.StringVar(&hpaName, "hpa", "", "HPA to check")
	pflag.StringVar(&kubeconfigPath, "kubeconfig-path", defaultKubeconfigPath, "Absolute path to the KubeconfigPath file")
	pflag.Parse()

	if len(hpaName) == 0 {
		fmt.Println("hpa flag is mandatory")
		os.Exit(1)
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		fmt.Printf("Error reading the kubeconfig: %s\n", err)
		os.Exit(1)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	namespace, err := client.CoreV1().Namespaces().Get(context.Background(), os.Getenv("NAMESPACE"), metav1.GetOptions{})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	hpaClientV2 := client.AutoscalingV2().HorizontalPodAutoscalers(namespace.Name)
	hpaClientV1 := client.AutoscalingV1().HorizontalPodAutoscalers(namespace.Name)

	pollInterval := time.Second

	// Search in v2 and v1 for an HPA with the given name

	ctx := context.Background()
	err = wait.PollUntilContextTimeout(ctx, pollInterval, timeout, false, func(c context.Context) (done bool, err error) {
		hpav2, err := hpaClientV2.Get(
			c,
			hpaName,
			metav1.GetOptions{},
		)
		if err != nil {
			hpav1, err := hpaClientV1.Get(
				c,
				hpaName,
				metav1.GetOptions{},
			)
			if err != nil {
				fmt.Printf("HPA %s not found\n", hpaName)
				return false, nil
			}

			if hpav1.Status.CurrentCPUUtilizationPercentage == nil {
				fmt.Printf("Current metrics are not set yet for HPA %s\n", hpaName)
				return false, nil
			}
			return true, nil
		}

		if hpav2.Status.CurrentMetrics == nil {
			fmt.Printf("Current metrics are not set yet for HPA %s\n", hpaName)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("%s is ready!\n", hpaName)
}
