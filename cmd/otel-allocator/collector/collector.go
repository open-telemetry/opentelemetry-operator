// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package collector

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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

type CollectorWatcherType int

const (
	K8sCollectorWatcher CollectorWatcherType = iota
	AwsCloudMapCollectorWatcher
)

var collectorWatcherTypeStrings = []string{"k8s", "aws-cloud-map"}

func ParseCollectorWatcherType(s string) (CollectorWatcherType, error) {
	for i, name := range collectorWatcherTypeStrings {
		if strings.ToLower(s) == name {
			return CollectorWatcherType(i), nil
		}
	}
	return 0, errors.New("invalid collector watcher type")
}

// Implement the Stringer interface for CollectorWatcherType
func (c CollectorWatcherType) String() string {
	if int(c) < len(collectorWatcherTypeStrings) {
		return collectorWatcherTypeStrings[c]
	}
	return "unknown"
}

// Implement the Set method for pflag
func (c *CollectorWatcherType) Set(value string) error {
	for i, name := range collectorWatcherTypeStrings {
		if strings.ToLower(value) == name {
			*c = CollectorWatcherType(i)
			return nil
		}
	}
	return errors.New("invalid collector watcher type")
}

// Implement the Type method for pflag
func (c *CollectorWatcherType) Type() string {
	return "CollectorWatcherType"
}

var _ Watcher = &watcher{}

// CollectorWatcher interface defines the common methods for watchers
type Watcher interface {
	Watch(...WatchOption) error
	Close()
}

type watcher struct {
	log               logr.Logger
	minUpdateInterval time.Duration
	close             chan struct{}
}

func (w *watcher) Watch(options ...WatchOption) error {
	config := &WatchConfig{}
	for _, opt := range options {
		opt(config)
	}

	if config.fn == nil {
		return fmt.Errorf("fn is required")
	}

	if config.labelSelector == nil {
		config.labelSelector = &metav1.LabelSelector{}
	}

	if w.minUpdateInterval == 0 {
		w.minUpdateInterval = defaultMinUpdateInterval
	}

	if w.close == nil {
		w.close = make(chan struct{})
	}
	return nil
}

func (w *watcher) Close() {
	if w.close != nil {
		close(w.close)
	}
}

// WatchConfig struct defines the common parameters for the watch method of Collector Watchers
type WatchConfig struct {
	labelSelector *metav1.LabelSelector
	fn            func(collectors map[string]*allocation.Collector)
}

type WatchOption func(wc *WatchConfig)

func WithLabelSelector(labelSelector *metav1.LabelSelector) WatchOption {
	return func(wc *WatchConfig) {
		wc.labelSelector = labelSelector
	}
}

func WithFn(fn func(collectors map[string]*allocation.Collector)) WatchOption {
	return func(wc *WatchConfig) {
		wc.fn = fn
	}
}

type WatcherOption struct {
	k8sOption         K8sWatcherOption
	awsCloudMapOption AwsCloudMapWatcherOption
}

func WithMinUpdateInterval(interval time.Duration) WatcherOption {
	return WatcherOption{
		k8sOption: func(w *K8sWatcher) {
			w.watcher.minUpdateInterval = interval
		},
		awsCloudMapOption: func(w *AwsCloudMapWatcher) {
			w.watcher.minUpdateInterval = interval
		},
	}
}

func WithLogger(logger logr.Logger) WatcherOption {
	return WatcherOption{
		k8sOption: func(w *K8sWatcher) {
			w.watcher.log = logger
		},
		awsCloudMapOption: func(w *AwsCloudMapWatcher) {
			w.watcher.log = logger
		},
	}
}

func WithKubeConfig(config *rest.Config) WatcherOption {
	return WatcherOption{
		k8sOption: func(w *K8sWatcher) {
			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				// Handle error, perhaps log it or panic
				w.watcher.log.Error(err, "Failed to create Kubernetes client")
				return
			}
			w.k8sClient = clientset
		},
	}
}

func WithCloudMapConfig(namespaceName, serviceName *string) WatcherOption {
	return WatcherOption{
		awsCloudMapOption: func(w *AwsCloudMapWatcher) {
			w.namespaceName = namespaceName
			w.serviceName = serviceName
		},
	}
}

func NewCollectorWatcher(t CollectorWatcherType, options ...WatcherOption) (Watcher, error) {
	switch t {
	case K8sCollectorWatcher:
		return NewK8sWatcher(options...)
	case AwsCloudMapCollectorWatcher:
		return NewAwsCloudMapWatcher(options...)
	default:
		return nil, fmt.Errorf("invalid collector watcher type: %v", t)
	}
}
