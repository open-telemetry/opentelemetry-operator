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
	"sync"
	"testing"

	"k8s.io/apimachinery/pkg/watch"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation"
)

var logger = logf.Log.WithName("collector-unit-tests")

func getTestClient() (Client, watch.Interface) {
	kubeClient := Client{
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

	watcher, err := kubeClient.k8sClient.CoreV1().Pods("test-ns").Watch(context.Background(), opts)
	if err != nil {
		fmt.Printf("failed to setup a Collector Pod watcher: %v", err)
		os.Exit(1)
	}
	return kubeClient, watcher
}

func pod(name string) *v1.Pod {
	labelSet := make(map[string]string)
	labelSet["app.kubernetes.io/instance"] = "default.test"
	labelSet["app.kubernetes.io/managed-by"] = "opentelemetry-operator"

	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test-ns",
			Labels:    labelSet,
		},
	}
}

func Test_runWatch(t *testing.T) {
	type args struct {
		kubeFn       func(t *testing.T, client Client, group *sync.WaitGroup)
		collectorMap map[string]*allocation.Collector
	}
	tests := []struct {
		name string
		args args
		want map[string]*allocation.Collector
	}{
		{
			name: "pod add",
			args: args{
				kubeFn: func(t *testing.T, client Client, group *sync.WaitGroup) {
					for _, k := range []string{"test-pod1", "test-pod2", "test-pod3"} {
						p := pod(k)
						group.Add(1)
						_, err := client.k8sClient.CoreV1().Pods("test-ns").Create(context.Background(), p, metav1.CreateOptions{})
						assert.NoError(t, err)
					}
				},
				collectorMap: map[string]*allocation.Collector{},
			},
			want: map[string]*allocation.Collector{
				"test-pod1": {
					Name: "test-pod1",
				},
				"test-pod2": {
					Name: "test-pod2",
				},
				"test-pod3": {
					Name: "test-pod3",
				},
			},
		},
		{
			name: "pod delete",
			args: args{
				kubeFn: func(t *testing.T, client Client, group *sync.WaitGroup) {
					for _, k := range []string{"test-pod2", "test-pod3"} {
						group.Add(1)
						err := client.k8sClient.CoreV1().Pods("test-ns").Delete(context.Background(), k, metav1.DeleteOptions{})
						assert.NoError(t, err)
					}
				},
				collectorMap: map[string]*allocation.Collector{
					"test-pod1": {
						Name: "test-pod1",
					},
					"test-pod2": {
						Name: "test-pod2",
					},
					"test-pod3": {
						Name: "test-pod3",
					},
				},
			},
			want: map[string]*allocation.Collector{
				"test-pod1": {
					Name: "test-pod1",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kubeClient, watcher := getTestClient()
			defer func() {
				close(kubeClient.close)
				watcher.Stop()
			}()
			var wg sync.WaitGroup
			actual := make(map[string]*allocation.Collector)
			for _, k := range tt.args.collectorMap {
				p := pod(k.Name)
				_, err := kubeClient.k8sClient.CoreV1().Pods("test-ns").Create(context.Background(), p, metav1.CreateOptions{})
				wg.Add(1)
				assert.NoError(t, err)
			}
			go runWatch(context.Background(), &kubeClient, watcher.ResultChan(), map[string]*allocation.Collector{}, func(colMap map[string]*allocation.Collector) {
				actual = colMap
				wg.Done()
			})

			tt.args.kubeFn(t, kubeClient, &wg)
			wg.Wait()

			assert.Len(t, actual, len(tt.want))
			assert.Equal(t, actual, tt.want)
		})
	}
}
