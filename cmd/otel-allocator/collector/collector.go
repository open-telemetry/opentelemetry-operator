// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation"
)

const (
	defaultMinUpdateInterval = time.Second * 5
)

var (
	ns                   = os.Getenv("OTELCOL_NAMESPACE")
	collectorsDiscovered = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "opentelemetry_allocator_collectors_discovered",
		Help: "Number of collectors discovered.",
	})
)

type Watcher struct {
	log               logr.Logger
	k8sClient         kubernetes.Interface
	close             chan struct{}
	minUpdateInterval time.Duration
}

func NewCollectorWatcher(logger logr.Logger, kubeConfig *rest.Config) (*Watcher, error) {
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return &Watcher{}, err
	}

	return &Watcher{
		log:               logger.WithValues("component", "opentelemetry-targetallocator"),
		k8sClient:         clientset,
		close:             make(chan struct{}),
		minUpdateInterval: defaultMinUpdateInterval,
	}, nil
}

func (k *Watcher) Watch(labelSelector *metav1.LabelSelector, fn func(collectors map[string]*allocation.Collector)) error {
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return err
	}

	listOptionsFunc := func(listOptions *metav1.ListOptions) {
		listOptions.LabelSelector = selector.String()
	}
	informerFactory := informers.NewSharedInformerFactoryWithOptions(
		k.k8sClient,
		time.Second*30,
		informers.WithNamespace(ns),
		informers.WithTweakListOptions(listOptionsFunc))
	informer := informerFactory.Core().V1().Pods().Informer()

	notify := make(chan struct{}, 1)
	go k.rateLimitedCollectorHandler(notify, informer.GetStore(), fn)

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

	informer.Run(k.close)
	return nil
}

// rateLimitedCollectorHandler runs fn on collectors present in the store whenever it gets a notification on the notify channel,
// but not more frequently than once per k.eventPeriod.
func (k *Watcher) rateLimitedCollectorHandler(notify chan struct{}, store cache.Store, fn func(collectors map[string]*allocation.Collector)) {
	ticker := time.NewTicker(k.minUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-k.close:
			return
		case <-ticker.C: // throttle events to avoid excessive updates
			select {
			case <-notify:
				k.runOnCollectors(store, fn)
			default:
			}
		}
	}
}

// runOnCollectors runs the provided function on the set of collectors from the Store.
func (k *Watcher) runOnCollectors(store cache.Store, fn func(collectors map[string]*allocation.Collector)) {
	objects := store.List()
	collectorMap := make(map[string]*allocation.Collector, len(objects))
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

func (k *Watcher) Close() {
	close(k.close)
}
