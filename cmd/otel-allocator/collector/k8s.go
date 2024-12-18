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
	"errors"
	"time"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type K8sWatcher struct {
	k8sClient kubernetes.Interface
	watcher   watcher
}

type K8sWatcherOption func(*K8sWatcher)

var (
	errNoClient = errors.New("no Kubernetes client given")
)

func NewK8sWatcher(opts ...WatcherOption) (*K8sWatcher, error) {
	c := &K8sWatcher{
		watcher: watcher{
			close:             make(chan struct{}),
			minUpdateInterval: defaultMinUpdateInterval,
		},
	}
	for _, opt := range opts {
		if opt.k8sOption == nil {
			continue
		}
		opt.k8sOption(c)
	}
	if c.k8sClient == nil {
		return &K8sWatcher{}, errNoClient
	}
	return c, nil
}

func (w *K8sWatcher) Watch(options ...WatchOption) error { // labelSelector *metav1.LabelSelector, fn func(collectors map[string]*allocation.Collector)
	config := WatchConfig{}
	for _, option := range options {
		option(&config)
	}

	selector, err := metav1.LabelSelectorAsSelector(config.labelSelector)
	if err != nil {
		return err
	}

	listOptionsFunc := func(listOptions *metav1.ListOptions) {
		listOptions.LabelSelector = selector.String()
	}
	informerFactory := informers.NewSharedInformerFactoryWithOptions(
		w.k8sClient,
		time.Second*30,
		informers.WithNamespace(ns),
		informers.WithTweakListOptions(listOptionsFunc))
	informer := informerFactory.Core().V1().Pods().Informer()

	notify := make(chan struct{}, 1)
	go w.rateLimitedCollectorHandler(notify, informer.GetStore(), config.fn)

	notifyFunc := func(_ interface{}) {
		select {
		case notify <- struct{}{}:
		default:
		}
	}
	_, err = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: notifyFunc,
		UpdateFunc: func(oldObj, newObj interface{}) {
			notifyFunc(newObj)
		},
		DeleteFunc: notifyFunc,
	})
	if err != nil {
		return err
	}

	informer.Run(w.watcher.close)
	return nil
}

// rateLimitedCollectorHandler runs fn on collectors present in the store whenever it gets a notification on the notify channel,
// but not more frequently than once per k.eventPeriod.
func (w *K8sWatcher) rateLimitedCollectorHandler(notify chan struct{}, store cache.Store, fn func(collectors map[string]*allocation.Collector)) {
	ticker := time.NewTicker(w.watcher.minUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.watcher.close:
			return
		case <-ticker.C: // throttle events to avoid excessive updates
			select {
			case <-notify:
				w.runOnCollectors(store, fn)
			default:
			}
		}
	}
}

// runOnCollectors runs the provided function on the set of collectors from the Store.
func (w *K8sWatcher) runOnCollectors(store cache.Store, fn func(collectors map[string]*allocation.Collector)) {
	collectorMap := map[string]*allocation.Collector{}
	objects := store.List()
	for _, obj := range objects {
		pod := obj.(*v1.Pod)
		if pod.Spec.NodeName == "" {
			continue
		}
		collectorMap[pod.Name] = allocation.NewCollector(pod.Name, pod.Spec.NodeName)
	}
	collectorsDiscovered.Set(float64(len(collectorMap)))
	fn(collectorMap)
}

func (w *K8sWatcher) Close() {
	w.watcher.Close()
}
