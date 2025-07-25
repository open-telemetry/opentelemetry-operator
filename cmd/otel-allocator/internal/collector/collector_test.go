// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/embedded"
	"go.uber.org/atomic"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/allocation"
)

var logger = logf.Log.WithName("collector-unit-tests")
var labelMap = map[string]string{
	"app.kubernetes.io/instance":   "default.test",
	"app.kubernetes.io/managed-by": "opentelemetry-operator",
}
var labelSelector = metav1.LabelSelector{
	MatchLabels: labelMap,
}

type reportingGauge struct {
	embedded.Int64Gauge
	value atomic.Int64
}

func (r *reportingGauge) Record(_ context.Context, value int64, _ ...metric.RecordOption) {
	r.value.Store(value)
}

func getTestPodWatcher(collectorNotReadyGracePeriod time.Duration) *Watcher {
	podWatcher := Watcher{
		k8sClient:                    fake.NewClientset(),
		close:                        make(chan struct{}),
		log:                          logger,
		minUpdateInterval:            time.Millisecond,
		collectorNotReadyGracePeriod: collectorNotReadyGracePeriod,
		collectorsDiscovered:         &reportingGauge{},
	}
	return &podWatcher
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
		Status: v1.PodStatus{Phase: v1.PodRunning, Conditions: []v1.PodCondition{{
			Type:   v1.PodReady,
			Status: v1.ConditionTrue,
		}}},
	}
}

func podWithPodPhaseAndStartTime(name string, podPhase v1.PodPhase, startTime time.Time) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test-ns",
			Labels:    labelMap,
		},
		Spec: v1.PodSpec{
			NodeName: "test-node",
		},
		Status: v1.PodStatus{Phase: podPhase, StartTime: &metav1.Time{Time: startTime}},
	}
}

func podWithPodReadyConditionStatusAndLastTransitionTime(name string, podConditionStatus v1.ConditionStatus, lastTransitionTime time.Time) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test-ns",
			Labels:    labelMap,
		},
		Spec: v1.PodSpec{
			NodeName: "test-node",
		},
		Status: v1.PodStatus{Conditions: []v1.PodCondition{{
			Type:               v1.PodReady,
			Status:             podConditionStatus,
			LastTransitionTime: metav1.Time{Time: lastTransitionTime},
		}}},
	}
}

func Test_runWatch(t *testing.T) {
	namespace := "test-ns"
	type args struct {
		collectorNotReadyGracePeriod time.Duration
		kubeFn                       func(t *testing.T, podWatcher *Watcher)
		collectorMap                 map[string]*allocation.Collector
	}
	tests := []struct {
		name string
		args args
		want map[string]*allocation.Collector
	}{
		{
			name: "pod add",
			args: args{
				collectorNotReadyGracePeriod: 0 * time.Second,
				kubeFn: func(t *testing.T, podWatcher *Watcher) {
					for _, k := range []string{"test-pod1", "test-pod2", "test-pod3"} {
						p := pod(k)
						_, err := podWatcher.k8sClient.CoreV1().Pods(namespace).Create(context.Background(), p, metav1.CreateOptions{})
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
				collectorNotReadyGracePeriod: 0 * time.Second,
				kubeFn: func(t *testing.T, podWatcher *Watcher) {
					for _, k := range []string{"test-pod2", "test-pod3"} {
						err := podWatcher.k8sClient.CoreV1().Pods(namespace).Delete(context.Background(), k, metav1.DeleteOptions{})
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
			podWatcher := getTestPodWatcher(tt.args.collectorNotReadyGracePeriod)
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
			go func() {
				err := podWatcher.Watch(namespace, &labelSelector, func(colMap map[string]*allocation.Collector) {
					mapMutex.Lock()
					defer mapMutex.Unlock()
					actual = colMap
				})
				require.NoError(t, err)
			}()
			assert.Eventually(t, podWatcher.isSynced, time.Second*30, time.Millisecond*100)

			tt.args.kubeFn(t, podWatcher)

			assert.EventuallyWithT(t, func(collect *assert.CollectT) {
				mapMutex.Lock()
				defer mapMutex.Unlock()
				assert.Len(collect, actual, len(tt.want))
				assert.Equal(collect, tt.want, actual)
				assert.Equal(collect, podWatcher.collectorsDiscovered.(*reportingGauge).value.Load(), int64(len(actual)))
			}, time.Second*30, time.Millisecond*100)
		})
	}
}

func Test_gracePeriodWithNonRunningPodPhase(t *testing.T) {
	namespace := "test-ns"
	type args struct {
		collectorNotReadyGracePeriod time.Duration
		collectorMap                 map[string]*allocation.Collector
	}
	tests := []struct {
		name string
		args args
		want map[string]*allocation.Collector
	}{
		{
			name: "collector healthiness check disabled",
			args: args{
				collectorNotReadyGracePeriod: 0 * time.Second,
				collectorMap: map[string]*allocation.Collector{
					"test-pod-running": {
						Name:     "test-pod-running",
						NodeName: "test-node",
					},
					"test-pod-unknown-within-grace-period": {
						Name:     "test-pod-unknown-within-grace-period",
						NodeName: "test-node",
					},
					"test-pod-pending-over-grace-period": {
						Name:     "test-pod-pending-over-grace-period",
						NodeName: "test-node",
					},
				},
			},
			want: map[string]*allocation.Collector{
				"test-pod-running": {
					Name:     "test-pod-running",
					NodeName: "test-node",
				},
				"test-pod-unknown-within-grace-period": {
					Name:     "test-pod-unknown-within-grace-period",
					NodeName: "test-node",
				},
				"test-pod-pending-over-grace-period": {
					Name:     "test-pod-pending-over-grace-period",
					NodeName: "test-node",
				},
			},
		},
		{
			name: "collector healthiness check enabled",
			args: args{
				collectorNotReadyGracePeriod: 30 * time.Second,
				collectorMap: map[string]*allocation.Collector{
					"test-pod-running": {
						Name:     "test-pod-running",
						NodeName: "test-node",
					},
					"test-pod-unknown-within-grace-period": {
						Name:     "test-pod-unknown-within-grace-period",
						NodeName: "test-node",
					},
					"test-pod-pending-over-grace-period": {
						Name:     "test-pod-pending-over-grace-period",
						NodeName: "test-node",
					},
				},
			},
			want: map[string]*allocation.Collector{
				"test-pod-running": {
					Name:     "test-pod-running",
					NodeName: "test-node",
				},
				"test-pod-unknown-within-grace-period": {
					Name:     "test-pod-unknown-within-grace-period",
					NodeName: "test-node",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeNow := time.Now()
			podWatcher := getTestPodWatcher(tt.args.collectorNotReadyGracePeriod)
			defer func() {
				close(podWatcher.close)
			}()
			var actual map[string]*allocation.Collector
			mapMutex := sync.Mutex{}
			for _, k := range tt.args.collectorMap {
				var p *v1.Pod
				switch k.Name {
				case "test-pod-running":
					p = podWithPodPhaseAndStartTime(k.Name, v1.PodRunning, timeNow)
				case "test-pod-unknown-within-grace-period":
					p = podWithPodPhaseAndStartTime(k.Name, v1.PodUnknown,
						timeNow.Add(-1*podWatcher.collectorNotReadyGracePeriod).Add(podWatcher.collectorNotReadyGracePeriod/2))
				case "test-pod-pending-over-grace-period":
					p = podWithPodPhaseAndStartTime(k.Name, v1.PodPending,
						timeNow.Add(-1*podWatcher.collectorNotReadyGracePeriod).Add(-podWatcher.collectorNotReadyGracePeriod/2))
				}
				_, err := podWatcher.k8sClient.CoreV1().Pods("test-ns").Create(context.Background(), p, metav1.CreateOptions{})
				assert.NoError(t, err)
			}

			go func() {
				err := podWatcher.Watch(namespace, &labelSelector, func(colMap map[string]*allocation.Collector) {
					mapMutex.Lock()
					defer mapMutex.Unlock()
					actual = colMap
				})
				require.NoError(t, err)
			}()

			assert.EventuallyWithT(t, func(collect *assert.CollectT) {
				mapMutex.Lock()
				defer mapMutex.Unlock()
				assert.Len(collect, actual, len(tt.want))
				assert.Equal(collect, actual, tt.want)
				assert.Equal(collect, podWatcher.collectorsDiscovered.(*reportingGauge).value.Load(), int64(len(actual)))
			}, time.Second*3, time.Millisecond)
		})
	}
}

func Test_gracePeriodWithNonReadyPodCondition(t *testing.T) {
	namespace := "test-ns"
	type args struct {
		collectorNotReadyGracePeriod time.Duration
		collectorMap                 map[string]*allocation.Collector
	}

	tests := []struct {
		name string
		args args
		want map[string]*allocation.Collector
	}{
		{
			name: "collector healthiness check disabled",
			args: args{
				collectorNotReadyGracePeriod: 0 * time.Second,
				collectorMap: map[string]*allocation.Collector{
					"test-pod-ready": {
						Name:     "test-pod-ready",
						NodeName: "test-node",
					},
					"test-pod-non-ready-within-grace-period": {
						Name:     "test-pod-non-ready-within-grace-period",
						NodeName: "test-node",
					},
					"test-pod-non-ready-over-grace-period": {
						Name:     "test-pod-non-ready-over-grace-period",
						NodeName: "test-node",
					},
				},
			},
			want: map[string]*allocation.Collector{
				"test-pod-ready": {
					Name:     "test-pod-ready",
					NodeName: "test-node",
				},
				"test-pod-non-ready-within-grace-period": {
					Name:     "test-pod-non-ready-within-grace-period",
					NodeName: "test-node",
				},
				"test-pod-non-ready-over-grace-period": {
					Name:     "test-pod-non-ready-over-grace-period",
					NodeName: "test-node",
				},
			},
		},
		{
			name: "collector healthiness check enabled",
			args: args{
				collectorNotReadyGracePeriod: 30 * time.Second,
				collectorMap: map[string]*allocation.Collector{
					"test-pod-ready": {
						Name:     "test-pod-ready",
						NodeName: "test-node",
					},
					"test-pod-non-ready-within-grace-period": {
						Name:     "test-pod-non-ready-within-grace-period",
						NodeName: "test-node",
					},
					"test-pod-non-ready-over-grace-period": {
						Name:     "test-pod-non-ready-over-grace-period",
						NodeName: "test-node",
					},
				},
			},
			want: map[string]*allocation.Collector{
				"test-pod-ready": {
					Name:     "test-pod-ready",
					NodeName: "test-node",
				},
				"test-pod-non-ready-within-grace-period": {
					Name:     "test-pod-non-ready-within-grace-period",
					NodeName: "test-node",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeNow := time.Now()
			podWatcher := getTestPodWatcher(tt.args.collectorNotReadyGracePeriod)
			defer func() {
				close(podWatcher.close)
			}()
			var actual map[string]*allocation.Collector
			mapMutex := sync.Mutex{}
			for _, k := range tt.args.collectorMap {
				var p *v1.Pod
				switch k.Name {
				case "test-pod-ready":
					p = podWithPodReadyConditionStatusAndLastTransitionTime(k.Name, v1.ConditionTrue, timeNow)
				case "test-pod-non-ready-within-grace-period":
					p = podWithPodReadyConditionStatusAndLastTransitionTime(k.Name, v1.ConditionFalse,
						timeNow.Add(-1*podWatcher.collectorNotReadyGracePeriod).Add(podWatcher.collectorNotReadyGracePeriod/2))
				case "test-pod-non-ready-over-grace-period":
					p = podWithPodReadyConditionStatusAndLastTransitionTime(k.Name, v1.ConditionFalse,
						timeNow.Add(-1*podWatcher.collectorNotReadyGracePeriod).Add(-podWatcher.collectorNotReadyGracePeriod/2))
				}
				_, err := podWatcher.k8sClient.CoreV1().Pods("test-ns").Create(context.Background(), p, metav1.CreateOptions{})
				assert.NoError(t, err)
			}

			go func() {
				err := podWatcher.Watch(namespace, &labelSelector, func(colMap map[string]*allocation.Collector) {
					mapMutex.Lock()
					defer mapMutex.Unlock()
					actual = colMap
				})
				require.NoError(t, err)
			}()

			assert.EventuallyWithT(t, func(collect *assert.CollectT) {
				mapMutex.Lock()
				defer mapMutex.Unlock()
				assert.Len(collect, actual, len(tt.want))
				assert.Equal(collect, actual, tt.want)
				assert.Equal(collect, podWatcher.collectorsDiscovered.(*reportingGauge).value.Load(), int64(len(actual)))
			}, time.Second*3, time.Millisecond)
		})
	}
}

// this tests runWatch in the case of watcher channel closing.
func Test_closeChannel(t *testing.T) {
	podWatcher := getTestPodWatcher(0 * time.Second)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		err := podWatcher.Watch("default", &labelSelector, func(colMap map[string]*allocation.Collector) {})
		require.NoError(t, err)
	}()

	podWatcher.Close()
	wg.Wait()
}
