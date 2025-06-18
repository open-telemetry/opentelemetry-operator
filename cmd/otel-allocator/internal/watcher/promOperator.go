// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/blang/semver/v4"
	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	promv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/prometheus-operator/prometheus-operator/pkg/assets"
	monitoringclient "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	"github.com/prometheus-operator/prometheus-operator/pkg/informers"
	"github.com/prometheus-operator/prometheus-operator/pkg/k8sutil"
	"github.com/prometheus-operator/prometheus-operator/pkg/listwatch"
	"github.com/prometheus-operator/prometheus-operator/pkg/operator"
	"github.com/prometheus-operator/prometheus-operator/pkg/prometheus"
	prometheusgoclient "github.com/prometheus/client_golang/prometheus"
	promconfig "github.com/prometheus/prometheus/config"
	kubeDiscovery "github.com/prometheus/prometheus/discovery/kubernetes"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"

	allocatorconfig "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/internal/config"
)

const (
	resyncPeriod     = 5 * time.Minute
	minEventInterval = time.Second * 5
)

func NewPrometheusCRWatcher(ctx context.Context, logger logr.Logger, cfg allocatorconfig.Config) (*PrometheusCRWatcher, error) {
	promLogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	slogger := slog.New(logr.ToSlogHandler(logger))
	var resourceSelector *prometheus.ResourceSelector
	mClient, err := monitoringclient.NewForConfig(cfg.ClusterConfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(cfg.ClusterConfig)
	if err != nil {
		return nil, err
	}

	allowList, denyList := cfg.PrometheusCR.GetAllowDenyLists()

	factory := informers.NewMonitoringInformerFactories(allowList, denyList, mClient, allocatorconfig.DefaultResyncTime, nil)

	monitoringInformers, err := getInformers(factory, cfg.ClusterConfig, promLogger)
	if err != nil {
		return nil, err
	}

	// we want to use endpointslices by default
	serviceDiscoveryRole := monitoringv1.ServiceDiscoveryRole("EndpointSlice")

	// TODO: We should make these durations configurable
	prom := &monitoringv1.Prometheus{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cfg.CollectorNamespace,
		},
		Spec: monitoringv1.PrometheusSpec{
			CommonPrometheusFields: monitoringv1.CommonPrometheusFields{
				ScrapeInterval:                  monitoringv1.Duration(cfg.PrometheusCR.ScrapeInterval.String()),
				PodMonitorSelector:              cfg.PrometheusCR.PodMonitorSelector,
				PodMonitorNamespaceSelector:     cfg.PrometheusCR.PodMonitorNamespaceSelector,
				ServiceMonitorSelector:          cfg.PrometheusCR.ServiceMonitorSelector,
				ServiceMonitorNamespaceSelector: cfg.PrometheusCR.ServiceMonitorNamespaceSelector,
				ScrapeConfigSelector:            cfg.PrometheusCR.ScrapeConfigSelector,
				ScrapeConfigNamespaceSelector:   cfg.PrometheusCR.ScrapeConfigNamespaceSelector,
				ProbeSelector:                   cfg.PrometheusCR.ProbeSelector,
				ProbeNamespaceSelector:          cfg.PrometheusCR.ProbeNamespaceSelector,
				ServiceDiscoveryRole:            &serviceDiscoveryRole,
				Version:                         "2.55.1", // fix Prometheus version 2 to avoid generating incompatible config
				ScrapeProtocols:                 cfg.PrometheusCR.ScrapeProtocols,
			},
			EvaluationInterval: monitoringv1.Duration("30s"),
		},
	}

	generator, err := prometheus.NewConfigGenerator(promLogger, prom, prometheus.WithEndpointSliceSupport())

	if err != nil {
		return nil, err
	}

	store := assets.NewStoreBuilder(clientset.CoreV1(), clientset.CoreV1())
	promRegisterer := prometheusgoclient.NewRegistry()
	operatorMetrics := operator.NewMetrics(promRegisterer)
	eventRecorderFactory := operator.NewEventRecorderFactory(false)
	eventRecorder := eventRecorderFactory(clientset, "target-allocator")

	var nsMonInf cache.SharedIndexInformer
	getNamespaceInformerErr := retry.OnError(retry.DefaultRetry,
		func(err error) bool {
			logger.Error(err, "Retrying namespace informer creation in promOperator CRD watcher")
			return true
		}, func() error {
			nsMonInf, err = getNamespaceInformer(ctx, allowList, denyList, promLogger, clientset, operatorMetrics)
			return err
		})
	if getNamespaceInformerErr != nil {
		logger.Error(getNamespaceInformerErr, "Failed to create namespace informer in promOperator CRD watcher")
		return nil, getNamespaceInformerErr
	}

	resourceSelector, err = prometheus.NewResourceSelector(slogger, prom, store, nsMonInf, operatorMetrics, eventRecorder)
	if err != nil {
		logger.Error(err, "Failed to create resource selector in promOperator CRD watcher")
	}

	return &PrometheusCRWatcher{
		logger:                          slogger,
		kubeMonitoringClient:            mClient,
		k8sClient:                       clientset,
		informers:                       monitoringInformers,
		nsInformer:                      nsMonInf,
		stopChannel:                     make(chan struct{}),
		eventInterval:                   minEventInterval,
		configGenerator:                 generator,
		kubeConfigPath:                  cfg.KubeConfigFilePath,
		podMonitorNamespaceSelector:     cfg.PrometheusCR.PodMonitorNamespaceSelector,
		serviceMonitorNamespaceSelector: cfg.PrometheusCR.ServiceMonitorNamespaceSelector,
		scrapeConfigNamespaceSelector:   cfg.PrometheusCR.ScrapeConfigNamespaceSelector,
		probeNamespaceSelector:          cfg.PrometheusCR.ProbeNamespaceSelector,
		resourceSelector:                resourceSelector,
		store:                           store,
		prometheusCR:                    prom,
	}, nil
}

type PrometheusCRWatcher struct {
	logger                          *slog.Logger
	kubeMonitoringClient            monitoringclient.Interface
	k8sClient                       kubernetes.Interface
	informers                       map[string]*informers.ForResource
	nsInformer                      cache.SharedIndexInformer
	eventInterval                   time.Duration
	stopChannel                     chan struct{}
	configGenerator                 *prometheus.ConfigGenerator
	kubeConfigPath                  string
	podMonitorNamespaceSelector     *metav1.LabelSelector
	serviceMonitorNamespaceSelector *metav1.LabelSelector
	scrapeConfigNamespaceSelector   *metav1.LabelSelector
	probeNamespaceSelector          *metav1.LabelSelector
	resourceSelector                *prometheus.ResourceSelector
	store                           *assets.StoreBuilder
	prometheusCR                    *monitoringv1.Prometheus
}

func getNamespaceInformer(ctx context.Context, allowList, denyList map[string]struct{}, promOperatorLogger *slog.Logger, clientset kubernetes.Interface, operatorMetrics *operator.Metrics) (cache.SharedIndexInformer, error) {
	kubernetesVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}
	kubernetesSemverVersion, err := semver.ParseTolerant(kubernetesVersion.String())
	if err != nil {
		return nil, err
	}
	lw, _, err := listwatch.NewNamespaceListWatchFromClient(
		ctx,
		promOperatorLogger,
		kubernetesSemverVersion,
		clientset.CoreV1(),
		clientset.AuthorizationV1().SelfSubjectAccessReviews(),
		allowList,
		denyList,
	)
	if err != nil {
		return nil, err
	}

	return cache.NewSharedIndexInformer(
		operatorMetrics.NewInstrumentedListerWatcher(lw),
		&v1.Namespace{}, resyncPeriod, cache.Indexers{},
	), nil

}

// checkCRDAvailability checks if a specific CRD is available in the cluster
func checkCRDAvailability(dcl discovery.DiscoveryInterface, groupVersion string, resourceName string) (bool, error) {
	apiList, err := dcl.ServerGroups()
	if err != nil {
		return false, err
	}

	apiGroups := apiList.Groups
	for _, group := range apiGroups {
		if group.Name == groupVersion {
			for _, version := range group.Versions {
				resources, err := dcl.ServerResourcesForGroupVersion(version.GroupVersion)
				if err != nil {
					return false, err
				}

				for _, resource := range resources.APIResources {
					if resource.Name == resourceName {
						return true, nil
					}
				}
			}
		}
	}

	return false, nil
}

// getInformers returns a map of informers for the given resources.
func getInformers(factory informers.FactoriesForNamespaces, clusterConfig *rest.Config, logger *slog.Logger) (map[string]*informers.ForResource, error) {
	informersMap := make(map[string]*informers.ForResource)

	// Get the discovery client
	dcl, err := discovery.NewDiscoveryClientForConfig(clusterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %w", err)
	}

	// Check for ServiceMonitor availability
	serviceMonitorAvailable, err := checkCRDAvailability(dcl, "monitoring.coreos.com", "servicemonitors")
	if err != nil {
		logger.Warn("Failed to check ServiceMonitor availability", "error", err)
	} else if serviceMonitorAvailable {
		serviceMonitorInformers, err := informers.NewInformersForResource(factory, monitoringv1.SchemeGroupVersion.WithResource(monitoringv1.ServiceMonitorName))
		if err != nil {
			return nil, err
		}
		informersMap[monitoringv1.ServiceMonitorName] = serviceMonitorInformers
	} else {
		logger.Warn("ServiceMonitor CRD not available, skipping informer")
	}

	// Check for PodMonitor availability
	podMonitorAvailable, err := checkCRDAvailability(dcl, "monitoring.coreos.com", "podmonitors")
	if err != nil {
		logger.Warn("Failed to check PodMonitor availability", "error", err)
	} else if podMonitorAvailable {
		podMonitorInformers, err := informers.NewInformersForResource(factory, monitoringv1.SchemeGroupVersion.WithResource(monitoringv1.PodMonitorName))
		if err != nil {
			return nil, err
		}
		informersMap[monitoringv1.PodMonitorName] = podMonitorInformers
	} else {
		logger.Warn("PodMonitor CRD not available, skipping informer")
	}

	// Check for Probe availability
	probeAvailable, err := checkCRDAvailability(dcl, "monitoring.coreos.com", "probes")
	if err != nil {
		logger.Warn("Failed to check Probe availability", "error", err)
	} else if probeAvailable {
		probeInformers, err := informers.NewInformersForResource(factory, monitoringv1.SchemeGroupVersion.WithResource(monitoringv1.ProbeName))
		if err != nil {
			return nil, err
		}
		informersMap[monitoringv1.ProbeName] = probeInformers
	} else {
		logger.Warn("Probe CRD not available, skipping informer")
	}

	// Check for ScrapeConfig availability
	scrapeConfigAvailable, err := checkCRDAvailability(dcl, "monitoring.coreos.com", "scrapeconfigs")
	if err != nil {
		logger.Warn("Failed to check ScrapeConfig availability", "error", err)
	} else if scrapeConfigAvailable {
		scrapeConfigInformers, err := informers.NewInformersForResource(factory, promv1alpha1.SchemeGroupVersion.WithResource(promv1alpha1.ScrapeConfigName))
		if err != nil {
			return nil, err
		}
		informersMap[promv1alpha1.ScrapeConfigName] = scrapeConfigInformers
	} else {
		logger.Warn("ScrapeConfig CRD not available, skipping informer")
	}

	return informersMap, nil
}

// Watch wrapped informers and wait for an initial sync.
func (w *PrometheusCRWatcher) Watch(upstreamEvents chan Event, upstreamErrors chan error) error {
	success := true
	// this channel needs to be buffered because notifications are asynchronous and neither producers nor consumers wait
	notifyEvents := make(chan struct{}, 1)

	if w.nsInformer != nil {
		go w.nsInformer.Run(w.stopChannel)
		if ok := w.WaitForNamedCacheSync("namespace", w.nsInformer.HasSynced); !ok {
			success = false
		}

		_, _ = w.nsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(oldObj, newObj interface{}) {
				old := oldObj.(*v1.Namespace)
				cur := newObj.(*v1.Namespace)

				// Periodic resync may resend the Namespace without changes
				// in-between.
				if old.ResourceVersion == cur.ResourceVersion {
					return
				}

				for name, selector := range map[string]*metav1.LabelSelector{
					"PodMonitorNamespaceSelector":     w.podMonitorNamespaceSelector,
					"ServiceMonitorNamespaceSelector": w.serviceMonitorNamespaceSelector,
					"ProbeNamespaceSelector":          w.probeNamespaceSelector,
					"ScrapeConfigNamespaceSelector":   w.scrapeConfigNamespaceSelector,
				} {
					sync, err := k8sutil.LabelSelectionHasChanged(old.Labels, cur.Labels, selector)
					if err != nil {
						w.logger.Error("Failed to check label selection between namespaces while handling namespace updates", "selector", name, "error", err)
						return
					}

					if sync {
						select {
						case notifyEvents <- struct{}{}:
						default:
						}
						return
					}
				}
			},
		})
	} else {
		w.logger.Info("Unable to watch namespaces since namespace informer is nil")
	}

	// Only attempt to sync informers that were actually created
	for name, resource := range w.informers {
		if resource == nil {
			w.logger.Info("Skipping nil informer", "informer", name)
			continue
		}

		resource.Start(w.stopChannel)

		if ok := w.WaitForNamedCacheSync(name, resource.HasSynced); !ok {
			w.logger.Info("skipping informer", "informer", name)
			continue
		}

		// only send an event notification if there isn't one already
		resource.AddEventHandler(cache.ResourceEventHandlerFuncs{
			// these functions only write to the notification channel if it's empty to avoid blocking
			// if scrape config updates are being rate-limited
			AddFunc: func(obj interface{}) {
				select {
				case notifyEvents <- struct{}{}:
				default:
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				select {
				case notifyEvents <- struct{}{}:
				default:
				}
			},
			DeleteFunc: func(obj interface{}) {
				select {
				case notifyEvents <- struct{}{}:
				default:
				}
			},
		})
	}
	if !success {
		return fmt.Errorf("failed to sync one of the caches")
	}

	// limit the rate of outgoing events
	w.rateLimitedEventSender(upstreamEvents, notifyEvents)

	<-w.stopChannel
	return nil
}

// rateLimitedEventSender sends events to the upstreamEvents channel whenever it gets a notification on the notifyEvents channel,
// but not more frequently than once per w.eventPeriod.
func (w *PrometheusCRWatcher) rateLimitedEventSender(upstreamEvents chan Event, notifyEvents chan struct{}) {
	ticker := time.NewTicker(w.eventInterval)
	defer ticker.Stop()

	event := Event{
		Source:  EventSourcePrometheusCR,
		Watcher: Watcher(w),
	}

	for {
		select {
		case <-w.stopChannel:
			return
		case <-ticker.C: // throttle events to avoid excessive updates
			select {
			case <-notifyEvents:
				select {
				case upstreamEvents <- event:
				default: // put the notification back in the queue if we can't send it upstream
					select {
					case notifyEvents <- struct{}{}:
					default:
					}
				}
			default:
			}
		}
	}
}

func (w *PrometheusCRWatcher) Close() error {
	close(w.stopChannel)
	return nil
}

func (w *PrometheusCRWatcher) LoadConfig(ctx context.Context) (*promconfig.Config, error) {
	promCfg := &promconfig.Config{}

	if w.resourceSelector != nil {
		var serviceMonitorInstances map[string]*monitoringv1.ServiceMonitor
		var podMonitorInstances map[string]*monitoringv1.PodMonitor
		var probeInstances map[string]*monitoringv1.Probe
		var scrapeConfigInstances map[string]*promv1alpha1.ScrapeConfig
		var err error

		// Only try to get ServiceMonitors if the informer exists
		if informer, ok := w.informers[monitoringv1.ServiceMonitorName]; ok {
			serviceMonitorInstances, err = w.resourceSelector.SelectServiceMonitors(ctx, informer.ListAllByNamespace)
			if err != nil {
				return nil, err
			}
		}

		// Only try to get PodMonitors if the informer exists
		if informer, ok := w.informers[monitoringv1.PodMonitorName]; ok {
			podMonitorInstances, err = w.resourceSelector.SelectPodMonitors(ctx, informer.ListAllByNamespace)
			if err != nil {
				return nil, err
			}
		}

		// Only try to get Probes if the informer exists
		if informer, ok := w.informers[monitoringv1.ProbeName]; ok {
			probeInstances, err = w.resourceSelector.SelectProbes(ctx, informer.ListAllByNamespace)
			if err != nil {
				return nil, err
			}
		}

		// Only try to get ScrapeConfigs if the informer exists
		if informer, ok := w.informers[promv1alpha1.ScrapeConfigName]; ok {
			scrapeConfigInstances, err = w.resourceSelector.SelectScrapeConfigs(ctx, informer.ListAllByNamespace)
			if err != nil {
				return nil, err
			}
		}

		generatedConfig, err := w.configGenerator.GenerateServerConfiguration(
			w.prometheusCR,
			serviceMonitorInstances,
			podMonitorInstances,
			probeInstances,
			scrapeConfigInstances,
			w.store,
			nil,
			nil,
			nil,
			[]string{})
		if err != nil {
			return nil, err
		}

		unmarshalErr := yaml.Unmarshal(generatedConfig, promCfg)
		if unmarshalErr != nil {
			return nil, unmarshalErr
		}

		// set kubeconfig path to service discovery configs, else kubernetes_sd will always attempt in-cluster
		// authentication even if running with a detected kubeconfig
		for _, scrapeConfig := range promCfg.ScrapeConfigs {
			for _, serviceDiscoveryConfig := range scrapeConfig.ServiceDiscoveryConfigs {
				if serviceDiscoveryConfig.Name() == "kubernetes" {
					sdConfig := interface{}(serviceDiscoveryConfig).(*kubeDiscovery.SDConfig)
					sdConfig.KubeConfig = w.kubeConfigPath
				}
			}
		}
		return promCfg, nil
	} else {
		w.logger.Info("Unable to load config since resource selector is nil, returning empty prometheus config")
		return promCfg, nil
	}
}

// WaitForNamedCacheSync adds a timeout to the informer's wait for the cache to be ready.
// If the PrometheusCRWatcher is unable to load an informer within 15 seconds, the method is
// cancelled and returns false. A successful informer load will return true. This method also
// will be cancelled if the target allocator's stopChannel is called before it returns.
//
// This method is inspired by the upstream prometheus-operator implementation, with a shorter timeout
// and support for the PrometheusCRWatcher's stopChannel.
// https://github.com/prometheus-operator/prometheus-operator/blob/293c16c854ce69d1da9fdc8f0705de2d67bfdbfa/pkg/operator/operator.go#L433
func (w *PrometheusCRWatcher) WaitForNamedCacheSync(controllerName string, inf cache.InformerSynced) bool {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	t := time.NewTicker(time.Second * 5)
	defer t.Stop()

	go func() {
		for {
			select {
			case <-t.C:
				w.logger.Debug("cache sync not yet completed")
			case <-ctx.Done():
				return
			case <-w.stopChannel:
				w.logger.Warn("stop received, shutting down cache syncing")
				cancel()
				return
			}
		}
	}()

	ok := cache.WaitForNamedCacheSync(controllerName, ctx.Done(), inf)
	if !ok {
		w.logger.Error("failed to sync cache")
	} else {
		w.logger.Debug("successfully synced cache")
	}

	return ok
}
