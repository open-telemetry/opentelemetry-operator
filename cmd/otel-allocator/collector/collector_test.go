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

package collector

import (
	"context"
	"fmt"
	"os"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation"
)

var client Client
var collectors = map[string]*allocation.Collector{}
var logger = logf.Log.WithName("collector-unit-tests")

func TestMain(m *testing.M) {
	client = Client{
		k8sClient: fake.NewSimpleClientset(),
		close:     make(chan struct{}),
		log:       logger,
	}

	labelMap := map[string]string{
		"app.kubernetes.io/instance":   "default.test",
		"app.kubernetes.io/managed-by": "opentelemetry-operator",
	}

	opts := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
	}

	watcher, err := client.k8sClient.CoreV1().Pods("test-ns").Watch(context.Background(), opts)
	if err != nil {
		fmt.Printf("failed to setup a Collector Pod watcher: %v", err)
		os.Exit(1)
	}

	go runWatch(context.Background(), &client, watcher.ResultChan(), map[string]*allocation.Collector{}, func(colMap map[string]*allocation.Collector) { getCollectors(colMap) })

	code := m.Run()

	close(client.close)

	os.Exit(code)
}

func TestWatchPodAddition(t *testing.T) {
	expected := map[string]*allocation.Collector{
		"test-pod1": {
			Name: "test-pod1",
		},
		"test-pod2": {
			Name: "test-pod2",
		},
		"test-pod3": {
			Name: "test-pod3",
		},
	}

	for _, k := range []string{"test-pod1", "test-pod2", "test-pod3"} {
		expected := pod(k)
		_, err := client.k8sClient.CoreV1().Pods("test-ns").Create(context.Background(), expected, metav1.CreateOptions{})
		assert.NoError(t, err)
	}

	assert.Len(t, collectors, 3)
	assert.Equal(t, expected, collectors)
}

func TestWatchPodDeletion(t *testing.T) {
	expected := []string{"test-pod1"}

	for _, k := range []string{"test-pod2", "test-pod3"} {
		err := client.k8sClient.CoreV1().Pods("test-ns").Delete(context.Background(), k, metav1.DeleteOptions{})
		assert.NoError(t, err)
	}

	assert.Len(t, collectors, 1)

	assert.Equal(t, expected, collectors)
}

func getCollectors(c map[string]*allocation.Collector) {
	collectors = c
}

func pod(name string) *v1.Pod {
	labels := make(map[string]string)
	labels["app.kubernetes.io/instance"] = "default.test"
	labels["app.kubernetes.io/managed-by"] = "opentelemetry-operator"

	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test-ns",
			Labels:    labels,
		},
	}
}
