// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/pflag"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/controller-runtime/pkg/client"

	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

var scheme *k8sruntime.Scheme

func init() {
	scheme = k8sruntime.NewScheme()
	utilruntime.Must(otelv1alpha1.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))
}

func main() {
	var timeout int
	var kubeconfigPath string

	defaultKubeconfigPath := filepath.Join(homedir.HomeDir(), ".kube", "config")

	pflag.IntVar(&timeout, "timeout", 300, "The timeout for the check.")
	pflag.StringVar(&kubeconfigPath, "kubeconfig-path", defaultKubeconfigPath, "Absolute path to the KubeconfigPath file")
	pflag.Parse()

	pollInterval := 500 * time.Millisecond
	timeoutPoll := time.Duration(timeout) * time.Second

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		fmt.Printf("Error reading the kubeconfig: %s\n", err.Error())
		os.Exit(1)
	}

	clusterClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		fmt.Printf("Creating the Kubernetes client: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("Waiting until the OpenTelemetry Operator deployment is created")
	operatorDeployment := &appsv1.Deployment{}

	ctx := context.Background()
	err = wait.PollUntilContextTimeout(ctx, pollInterval, timeoutPoll, false, func(c context.Context) (done bool, err error) {
		err = clusterClient.Get(
			c,
			client.ObjectKey{
				Name:      "opentelemetry-operator-controller-manager",
				Namespace: "opentelemetry-operator-system",
			},
			operatorDeployment,
		)
		if err != nil {
			fmt.Printf("Failed to get OpenTelemetry operator deployment: %s\n", err)
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("OpenTelemetry Operator deployment is created. Now checking if it if fully operational.")

	// Sometimes, the deployment of the OTEL Operator is ready but, when
	// creating new instances of the OTEL Collector, the webhook is not reachable
	// and kubectl apply fails. This code deployes an OTEL Collector instance
	// until success (or timeout)
	collectorInstance := otelv1alpha1.OpenTelemetryCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "operator-check",
			Namespace: "default",
		},
	}

	// Ensure the collector is not there before the check
	_ = clusterClient.Delete(context.Background(), &collectorInstance)

	fmt.Println("Check if the OpenTelemetry collector CR can be created.")
	collectorCtx := context.Background()
	err = wait.PollUntilContextTimeout(collectorCtx, pollInterval, timeoutPoll, false, func(c context.Context) (done bool, err error) {
		err = clusterClient.Create(
			c,
			&collectorInstance,
		)
		if err != nil {
			fmt.Printf("failed: to create OpenTelemetry collector CR %s\n", err)
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := clusterClient.Delete(context.Background(), &collectorInstance); err != nil {
		fmt.Printf("Failed to delete OpenTelemetry collector CR: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("OpenTelemetry operator is ready.")
}
