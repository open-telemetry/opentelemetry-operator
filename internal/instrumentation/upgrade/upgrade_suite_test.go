// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upgrade

import (
	"os"
	"path/filepath"
	"testing"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlenvtest "sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/testenv"
)

var (
	k8sClient  client.Client
	testEnv    *ctrlenvtest.Environment
	testScheme = scheme.Scheme
	cfg        *rest.Config
)

func TestMain(m *testing.M) {
	utilruntime.Must(v1alpha1.AddToScheme(testScheme))

	tenv := testenv.Start(&ctrlenvtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "config", "crd", "bases"),
		},
	}, testScheme)
	testEnv = tenv.Env
	cfg = tenv.Config
	k8sClient = tenv.Client

	code := m.Run()
	tenv.Stop()
	os.Exit(code)
}
