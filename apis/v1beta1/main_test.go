// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package v1beta1_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

var k8sClient client.Client

func TestMain(m *testing.M) {
	sch := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(sch))
	utilruntime.Must(v1beta1.AddToScheme(sch))

	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
	}

	cfg, err := testEnv.Start()
	if err != nil {
		fmt.Printf("failed to start test environment: %v", err)
		os.Exit(1)
	}
	k8sClient, err = client.New(cfg, client.Options{Scheme: sch})
	if err != nil {
		fmt.Printf("failed to setup a Kubernetes client: %v", err)
		os.Exit(1)
	}

	code := m.Run()

	if err := testEnv.Stop(); err != nil {
		fmt.Printf("failed to stop test environment: %v", err)
		os.Exit(1)
	}
	os.Exit(code)
}

func prepareNamespace(t *testing.T, ctx context.Context) string {
	t.Helper()

	name := "test-namespace-" + strconv.Itoa(rand.Int()) // nolint:gosec
	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	err := k8sClient.Create(ctx, namespace)
	if err != nil {
		t.Fatalf("failed to create namespace: %v", err)
	}

	t.Cleanup(func() {
		if err := k8sClient.Delete(ctx, namespace); err != nil {
			t.Fatalf("failed to delete namespace: %v", err)
		}
	})
	return name
}
