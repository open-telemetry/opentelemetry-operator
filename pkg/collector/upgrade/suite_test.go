// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlenvtest "sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/rbac"
	"github.com/open-telemetry/opentelemetry-operator/internal/testenv"
	"github.com/open-telemetry/opentelemetry-operator/internal/version"
	wh "github.com/open-telemetry/opentelemetry-operator/internal/webhook"
)

var (
	k8sClient  client.Client
	testEnv    *ctrlenvtest.Environment
	testScheme *runtime.Scheme = scheme.Scheme
	ctx        context.Context
	cancel     context.CancelFunc
	cfg        *rest.Config
)

func TestMain(m *testing.M) {
	ctx, cancel = context.WithCancel(context.TODO())
	defer cancel()
	utilruntime.Must(v1alpha1.AddToScheme(testScheme))
	utilruntime.Must(v1beta1.AddToScheme(testScheme))

	tenv, err := testenv.Start(&ctrlenvtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
		WebhookInstallOptions: ctrlenvtest.WebhookInstallOptions{
			Paths:                   []string{filepath.Join("..", "..", "..", "config", "webhook")},
			IgnoreSchemeConvertible: true,
		},
	}, testScheme)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	testEnv = tenv.Env
	cfg = tenv.Config
	k8sClient = tenv.Client

	mgr, err := testenv.NewWebhookManager(cfg, testScheme, &testEnv.WebhookInstallOptions)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	clientset, clientErr := kubernetes.NewForConfig(cfg)
	if clientErr != nil {
		fmt.Printf("failed to setup kubernetes clientset %v", clientErr)
	}
	reviewer := rbac.NewReviewer(clientset)

	if err = wh.SetupCollectorWebhook(mgr, config.New(), reviewer, nil, nil, nil); err != nil {
		fmt.Printf("failed to SetupWebhookWithManager: %v", err)
		os.Exit(1)
	}

	//+kubebuilder:scaffold:webhook

	if err := testenv.RunWebhookServer(ctx, mgr, &testEnv.WebhookInstallOptions); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	code := m.Run()

	if err := tenv.Stop(); err != nil {
		fmt.Println(err)
	}
	os.Exit(code)
}

func makeVersion(v string) version.Version {
	return version.Version{
		OpenTelemetryCollector: v,
	}
}
