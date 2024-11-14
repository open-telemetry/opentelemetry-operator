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
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery"
	"github.com/aws/aws-sdk-go-v2/service/servicediscovery/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/allocation"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type AwsCloudMapWatcher struct {
	svc           *servicediscovery.Client
	namespaceName *string
	serviceName   *string
	watcher       watcher
}

type AwsCloudMapWatcherOption func(*AwsCloudMapWatcher)

var (
	errNoNamespace    = errors.New("no Cloud Map namespace specified to resolve the backends")
	errNoServiceName  = errors.New("no Cloud Map service_name specified to resolve the backends")
	discoveryDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "cloudmap_discovery_duration_seconds",
		Help:    "Time taken to discover instances in Cloud Map",
		Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
	})

	discoveryErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cloudmap_discovery_errors_total",
		Help: "Total number of collector discovery errors",
	})
	healthyInstances = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cloudmap_healthy_instances",
		Help: "Number of healthy instances in Cloud Map",
	})
	unhealthyInstances = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cloudmap_unhealthy_instances",
		Help: "Number of unhealthy instances in Cloud Map",
	})
	totalInstances = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cloudmap_instances_total",
		Help: "Total number of instances in Cloud Map",
	})
)

func NewAwsCloudMapWatcher(opts ...WatcherOption) (*AwsCloudMapWatcher, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithDefaultRegion(""))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
		return nil, err
	}

	// Using the Config value, create the DynamoDB client
	svc := servicediscovery.NewFromConfig(cfg)

	w := &AwsCloudMapWatcher{
		svc: svc,

		watcher: watcher{
			close:             make(chan struct{}),
			minUpdateInterval: defaultMinUpdateInterval,
		},
	}

	for _, opt := range opts {
		if opt.awsCloudMapOption == nil {
			continue
		}
		opt.awsCloudMapOption(w)
	}

	if w.namespaceName == nil || len(*w.namespaceName) == 0 {
		return nil, errNoNamespace
	}

	if w.serviceName == nil || len(*w.serviceName) == 0 {
		return nil, errNoServiceName
	}

	return w, nil
}

func (w *AwsCloudMapWatcher) Watch(options ...WatchOption) error {
	config := WatchConfig{}

	for _, option := range options {
		option(&config)
	}

	if w.svc == nil {
		return fmt.Errorf("AWS Cloud Map service client not initialized")
	}

	startTime := time.Now()

	discoverOutput, err := w.svc.DiscoverInstances(context.TODO(), &servicediscovery.DiscoverInstancesInput{
		NamespaceName: w.namespaceName,
		ServiceName:   w.serviceName,
		MaxResults:    aws.Int32(100), // Limit results for better performance
	})
	if err != nil {
		return fmt.Errorf("failed to discover instances: %w", err)
	}

	// Track discovery metrics
	discoveryDuration.Observe(time.Since(startTime).Seconds())
	if err != nil {
		discoveryErrors.Inc()
		return fmt.Errorf("failed to discover instances: %w", err)
	}

	discoveredInstances, healthStats := w.processBatch(discoverOutput.Instances)

	// Update metrics
	w.updateMetrics(healthStats.healthy, healthStats.unhealthy)

	w.watcher.log.Info("discovered instances",
		"total", len(discoverOutput.Instances),
		"healthy", healthStats.healthy,
		"unhealthy", healthStats.unhealthy,
		"namespace", w.namespaceName,
		"service", w.serviceName,
	)

	go w.rateLimitedCollectorHandler(discoveredInstances, config.fn)

	return nil
}

func (w *AwsCloudMapWatcher) processBatch(instances []types.HttpInstanceSummary) ([]types.HttpInstanceSummary, struct{ healthy, unhealthy int }) {
	const batchSize = 50
	stats := struct{ healthy, unhealthy int }{}
	result := make([]types.HttpInstanceSummary, 0, len(instances))

	for i := 0; i < len(instances); i += batchSize {
		end := i + batchSize
		if end > len(instances) {
			end = len(instances)
		}

		for _, instance := range instances[i:end] {
			if instance.HealthStatus != types.HealthStatusUnhealthy {
				result = append(result, instance)
				stats.healthy++
			} else {
				stats.unhealthy++
			}
		}
	}

	return result, stats
}

func (w *AwsCloudMapWatcher) updateMetrics(healthy, unhealthy int) {
	healthyInstances.Set(float64(healthy))
	unhealthyInstances.Set(float64(unhealthy))
	totalInstances.Set(float64(healthy + unhealthy))
}

func (w *AwsCloudMapWatcher) rateLimitedCollectorHandler(store []types.HttpInstanceSummary, fn func(collectors map[string]*allocation.Collector)) {
	ticker := time.NewTicker(w.watcher.minUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.watcher.close:
			return
		case <-ticker.C:
			w.runOnCollectors(store, fn)
		}
	}
}

// runOnCollectors runs the provided function on the set of collectors from the Store.
func (w *AwsCloudMapWatcher) runOnCollectors(store []types.HttpInstanceSummary, fn func(collectors map[string]*allocation.Collector)) {
	collectorMap := make(map[string]*allocation.Collector, len(store))
	var node string
	for _, obj := range store {
		for attr, value := range obj.Attributes {
			if attr == "EC2_INSTANCE_ID" {
				node = value
			}
		}
		collectorMap[*obj.InstanceId] = allocation.NewCollector(*obj.InstanceId, node)
	}
	collectorsDiscovered.Set(float64(len(collectorMap)))
	fn(collectorMap)
}

func (w *AwsCloudMapWatcher) Close() {
	w.watcher.Close()
}
