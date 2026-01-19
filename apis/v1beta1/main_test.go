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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/yaml"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

var k8sClient client.Client

// loadCRDsWithV1Beta1Storage loads CRDs from the bases directory and patches
// the Instrumentation CRD to use v1beta1 as the storage version for these tests.
func loadCRDsWithV1Beta1Storage() ([]*apiextensionsv1.CustomResourceDefinition, error) {
	crdDir := filepath.Join("..", "..", "config", "crd", "bases")
	entries, err := os.ReadDir(crdDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read CRD directory: %w", err)
	}

	var crds []*apiextensionsv1.CustomResourceDefinition
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(crdDir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read CRD file %s: %w", entry.Name(), err)
		}

		crd := &apiextensionsv1.CustomResourceDefinition{}
		if err := yaml.Unmarshal(data, crd); err != nil {
			return nil, fmt.Errorf("failed to unmarshal CRD %s: %w", entry.Name(), err)
		}

		// Patch Instrumentation CRD to use v1beta1 as storage version for these tests
		if crd.Name == "instrumentations.opentelemetry.io" {
			for i := range crd.Spec.Versions {
				if crd.Spec.Versions[i].Name == "v1beta1" {
					crd.Spec.Versions[i].Storage = true
				} else {
					crd.Spec.Versions[i].Storage = false
				}
			}
		}

		crds = append(crds, crd)
	}

	return crds, nil
}

func TestMain(m *testing.M) {
	sch := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(sch))
	utilruntime.Must(v1beta1.AddToScheme(sch))

	crds, err := loadCRDsWithV1Beta1Storage()
	if err != nil {
		fmt.Printf("failed to load CRDs: %v", err)
		os.Exit(1)
	}

	testEnv := &envtest.Environment{
		CRDs: crds,
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
