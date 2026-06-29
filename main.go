// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	configv1 "github.com/openshift/api/config/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	operatorcmd "github.com/open-telemetry/opentelemetry-operator/cmd/operator"
	webhookcmd "github.com/open-telemetry/opentelemetry-operator/cmd/webhook"
)

var scheme = k8sruntime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(otelv1alpha1.AddToScheme(scheme))
	utilruntime.Must(otelv1beta1.AddToScheme(scheme))
	utilruntime.Must(networkingv1.AddToScheme(scheme))
	utilruntime.Must(configv1.AddToScheme(scheme))
	utilruntime.Must(gatewayv1.Install(scheme))

	// +kubebuilder:scaffold:scheme
}

func main() {
	rootCmd := operatorcmd.NewOperatorCmd(scheme)
	rootCmd.AddCommand(webhookcmd.NewWebhookCmd(scheme))
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
