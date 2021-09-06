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

package reconcile

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/signalfx/splunk-otel-operator/api/v1alpha1"
	"github.com/signalfx/splunk-otel-operator/internal/config"
)

var k8sClient client.Client
var testEnv *envtest.Environment
var testScheme *runtime.Scheme = scheme.Scheme
var logger = logf.Log.WithName("unit-tests")

var instanceUID = uuid.NewUUID()

func TestMain(m *testing.M) {
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
	}

	cfg, err := testEnv.Start()
	if err != nil {
		fmt.Printf("failed to start testEnv: %v", err)
		os.Exit(1)
	}

	if err := v1alpha1.AddToScheme(testScheme); err != nil {
		fmt.Printf("failed to register scheme: %v", err)
		os.Exit(1)
	}
	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: testScheme})
	if err != nil {
		fmt.Printf("failed to setup a Kubernetes client: %v", err)
		os.Exit(1)
	}

	code := m.Run()

	err = testEnv.Stop()
	if err != nil {
		fmt.Printf("failed to stop testEnv: %v", err)
		os.Exit(1)
	}

	os.Exit(code)
}

func params() Params {
	replicas := int32(2)
	configYAML, err := ioutil.ReadFile("test.yaml")
	if err != nil {
		fmt.Printf("Error getting yaml file: %v", err)
	}
	return Params{
		Config: config.New(),
		Client: k8sClient,
		Instance: v1alpha1.SplunkOtelAgent{
			TypeMeta: metav1.TypeMeta{
				Kind:       "splunk.com",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
				UID:       instanceUID,
			},
			Spec: v1alpha1.SplunkOtelAgentSpec{
				Ports: []v1.ServicePort{{
					Name: "web",
					Port: 80,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 80,
					},
					NodePort: 0,
				}},
				Replicas: &replicas,
				Config:   string(configYAML),
			},
		},
		Scheme:   testScheme,
		Log:      logger,
		Recorder: record.NewFakeRecorder(10),
	}
}

func newParams(containerImage string) (Params, error) {
	replicas := int32(1)
	configYAML, err := ioutil.ReadFile("test.yaml")
	if err != nil {
		return Params{}, fmt.Errorf("Error getting yaml file: %w", err)
	}

	cfg := config.New()

	return Params{
		Config: cfg,
		Client: k8sClient,
		Instance: v1alpha1.SplunkOtelAgent{
			TypeMeta: metav1.TypeMeta{
				Kind:       "splunk.com",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "default",
				UID:       instanceUID,
			},
			Spec: v1alpha1.SplunkOtelAgentSpec{
				Mode: v1alpha1.ModeStatefulSet,
				Ports: []v1.ServicePort{{
					Name: "web",
					Port: 80,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 80,
					},
					NodePort: 0,
				}},
				TargetAllocator: v1alpha1.OpenTelemetryTargetAllocatorSpec{
					Enabled: true,
					Image:   containerImage,
				},
				Replicas: &replicas,
				Config:   string(configYAML),
			},
		},
		Scheme: testScheme,
		Log:    logger,
	}, nil
}

func createObjectIfNotExists(tb testing.TB, name string, object client.Object) {
	tb.Helper()
	err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: "default", Name: name}, object)
	if errors.IsNotFound(err) {
		err := k8sClient.Create(context.Background(),
			object)
		assert.NoError(tb, err)
	}
}

func populateObjectIfExists(t testing.TB, object client.Object, namespacedName types.NamespacedName) (bool, error) {
	t.Helper()
	err := k8sClient.Get(context.Background(), namespacedName, object)
	if errors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil

}
