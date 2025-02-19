// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation"
)

var logger = logf.Log.WithName("collector-unit-tests")
var labelMap = map[string]string{
	"app.kubernetes.io/instance":   "default.test",
	"app.kubernetes.io/managed-by": "opentelemetry-operator",
}
var labelSelector = metav1.LabelSelector{
	MatchLabels: labelMap,
}

func getTestPodWatcher() Watcher {
	podWatcher := Watcher{
		k8sClient:         fake.NewSimpleClientset(),
		close:             make(chan struct{}),
		log:               logger,
		minUpdateInterval: time.Millisecond,
	}
	return podWatcher
}

func pod(name string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test-ns",
			Labels:    labelMap,
		},
		Spec: v1.PodSpec{
			NodeName: "test-node",
		},
	}
}

func Test_runWatch(t *testing.T) {
	type args struct {
		kubeFn       func(t *testing.T, podWatcher Watcher)
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
				kubeFn: func(t *testing.T, podWatcher Watcher) {
					for _, k := range []string{"test-pod1", "test-pod2", "test-pod3"} {
						p := pod(k)
						_, err := podWatcher.k8sClient.CoreV1().Pods("test-ns").Create(context.Background(), p, metav1.CreateOptions{})
						assert.NoError(t, err)
					}
				},
				collectorMap: map[string]*allocation.Collector{},
			},
			want: map[string]*allocation.Collector{
				"test-pod1": {
					Name:     "test-pod1",
					NodeName: "test-node",
				},
				"test-pod2": {
					Name:     "test-pod2",
					NodeName: "test-node",
				},
				"test-pod3": {
					Name:     "test-pod3",
					NodeName: "test-node",
				},
			},
		},
		{
			name: "pod delete",
			args: args{
				kubeFn: func(t *testing.T, podWatcher Watcher) {
					for _, k := range []string{"test-pod2", "test-pod3"} {
						err := podWatcher.k8sClient.CoreV1().Pods("test-ns").Delete(context.Background(), k, metav1.DeleteOptions{})
						assert.NoError(t, err)
					}
				},
				collectorMap: map[string]*allocation.Collector{
					"test-pod1": {
						Name:     "test-pod1",
						NodeName: "test-node",
					},
					"test-pod2": {
						Name:     "test-pod2",
						NodeName: "test-node",
					},
					"test-pod3": {
						Name:     "test-pod3",
						NodeName: "test-node",
					},
				},
			},
			want: map[string]*allocation.Collector{
				"test-pod1": {
					Name:     "test-pod1",
					NodeName: "test-node",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			podWatcher := getTestPodWatcher()
			defer func() {
				close(podWatcher.close)
			}()
			var actual map[string]*allocation.Collector
			mapMutex := sync.Mutex{}
			for _, k := range tt.args.collectorMap {
				p := pod(k.Name)
				_, err := podWatcher.k8sClient.CoreV1().Pods("test-ns").Create(context.Background(), p, metav1.CreateOptions{})
				assert.NoError(t, err)
			}
			go func(podWatcher Watcher) {
				err := podWatcher.Watch(&labelSelector, func(colMap map[string]*allocation.Collector) {
					mapMutex.Lock()
					defer mapMutex.Unlock()
					actual = colMap
				})
				require.NoError(t, err)
			}(podWatcher)

			tt.args.kubeFn(t, podWatcher)

			assert.EventuallyWithT(t, func(collect *assert.CollectT) {
				mapMutex.Lock()
				defer mapMutex.Unlock()
				assert.Len(collect, actual, len(tt.want))
				assert.Equal(collect, actual, tt.want)
				assert.Equal(collect, testutil.ToFloat64(collectorsDiscovered), float64(len(actual)))
			}, time.Second*3, time.Millisecond)
		})
	}
}

// this tests runWatch in the case of watcher channel closing.
func Test_closeChannel(t *testing.T) {
	podWatcher := getTestPodWatcher()

	var wg sync.WaitGroup
	wg.Add(1)

	go func(podWatcher Watcher) {
		defer wg.Done()
		err := podWatcher.Watch(&labelSelector, func(colMap map[string]*allocation.Collector) {})
		require.NoError(t, err)
	}(podWatcher)

	podWatcher.Close()
	wg.Wait()
}
