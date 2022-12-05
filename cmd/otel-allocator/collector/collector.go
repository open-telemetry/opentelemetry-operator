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
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation"
)

const (
	watcherTimeout = 15 * time.Minute
)

var (
	ns                   = os.Getenv("OTELCOL_NAMESPACE")
	collectorsDiscovered = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "opentelemetry_allocator_collectors_discovered",
		Help: "Number of collectors discovered.",
	})
)

type Client struct {
	log       logr.Logger
	k8sClient kubernetes.Interface
	close     chan struct{}
	timeoutSeconds int64
}

func NewClient(logger logr.Logger, kubeConfig *rest.Config) (*Client, error) {
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return &Client{}, err
	}

	return &Client{
		log:       logger,
		k8sClient: clientset,
		close:     make(chan struct{}),
		timeoutSeconds: int64(watcherTimeout / time.Second),
	}, nil
}

func (k *Client) Watch(ctx context.Context, labelMap map[string]string, fn func(collectors map[string]*allocation.Collector)) error {
	collectorMap := map[string]*allocation.Collector{}
	log := k.log.WithValues("component", "opentelemetry-targetallocator")

	// convert watcherTimeout to an integer in seconds
	interval := int64(watcherTimeout / time.Second)
	opts := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelMap).String(),
		TimeoutSeconds: &interval,
	}
	pods, err := k.k8sClient.CoreV1().Pods(ns).List(ctx, opts)
	if err != nil {
		log.Error(err, "Pod failure")
		os.Exit(1)
	}
	for i := range pods.Items {
		pod := pods.Items[i]
		if pod.GetObjectMeta().GetDeletionTimestamp() == nil {
			collectorMap[pod.Name] = allocation.NewCollector(pod.Name)
		}
	}

	fn(collectorMap)

	for {
		watcher, err := k.k8sClient.CoreV1().Pods(ns).Watch(ctx, opts)
		if err != nil {
			log.Error(err, "unable to create collector pod watcher")
			return err
		}
		log.Info("Successfully started a collector pod watcher")
		if msg := runWatch(ctx, k, watcher.ResultChan(), collectorMap, fn); msg != "" && msg != "no event" {
			log.Info("Collector pod watch event stopped " + msg)
			return nil
		}
	}
}

func runWatch(ctx context.Context, k *Client, c <-chan watch.Event, collectorMap map[string]*allocation.Collector, fn func(collectors map[string]*allocation.Collector)) string {
	log := k.log.WithValues("component", "opentelemetry-targetallocator")
	for {
		collectorsDiscovered.Set(float64(len(collectorMap)))
		select {
		case <-k.close:
			return "kubernetes client closed"
		case <-ctx.Done():
			return "context done"
		case event, ok := <-c:
			if !ok {
				log.Info("No event found. Restarting watch routine")
				return ""
			}

			pod, ok := event.Object.(*v1.Pod)
			if !ok {
				log.Info("No pod found in event Object. Restarting watch routine")
				return ""
			}

			switch event.Type { //nolint:exhaustive
			case watch.Added:
				collectorMap[pod.Name] = allocation.NewCollector(pod.Name)
			case watch.Deleted:
				delete(collectorMap, pod.Name)
			}
			fn(collectorMap)
		case <-time.After(time.Duration(k.timeoutSeconds) * time.Second):
			log.Info("Restarting watch routine")
			return ""
		}
	}
}

func (k *Client) Close() {
	close(k.close)
}
