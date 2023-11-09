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

package watcher

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/go-logr/logr"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	promv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/prometheus-operator/prometheus-operator/pkg/assets"
	monitoringclient "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	"github.com/prometheus-operator/prometheus-operator/pkg/informers"
	"github.com/prometheus-operator/prometheus-operator/pkg/prometheus"
	"github.com/prometheus/common/model"
	promconfig "github.com/prometheus/prometheus/config"
	kubeDiscovery "github.com/prometheus/prometheus/discovery/kubernetes"
	"github.com/prometheus/prometheus/model/relabel"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	allocatorconfig "github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/config"
)

const minEventInterval = time.Second * 5

func NewPrometheusCRWatcher(logger logr.Logger, cfg allocatorconfig.Config) (*PrometheusCRWatcher, error) {
	mClient, err := monitoringclient.NewForConfig(cfg.ClusterConfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(cfg.ClusterConfig)
	if err != nil {
		return nil, err
	}

	factory := informers.NewMonitoringInformerFactories(map[string]struct{}{v1.NamespaceAll: {}}, map[string]struct{}{}, mClient, allocatorconfig.DefaultResyncTime, nil) //TODO decide what strategy to use regarding namespaces

	monitoringInformers, err := getInformers(factory)
	if err != nil {
		return nil, err
	}

	// TODO: We should make these durations configurable
	prom := &monitoringv1.Prometheus{
		Spec: monitoringv1.PrometheusSpec{
			CommonPrometheusFields: monitoringv1.CommonPrometheusFields{
				ScrapeInterval: monitoringv1.Duration(cfg.PrometheusCR.ScrapeInterval.String()),
			},
		},
	}

	generator, err := prometheus.NewConfigGenerator(log.NewNopLogger(), prom, true) // TODO replace Nop?
	if err != nil {
		return nil, err
	}

	servMonSelector := getSelector(cfg.ServiceMonitorSelector)

	podMonSelector := getSelector(cfg.PodMonitorSelector)

	return &PrometheusCRWatcher{
		logger:                 logger,
		kubeMonitoringClient:   mClient,
		k8sClient:              clientset,
		informers:              monitoringInformers,
		stopChannel:            make(chan struct{}),
		eventInterval:          minEventInterval,
		configGenerator:        generator,
		kubeConfigPath:         cfg.KubeConfigFilePath,
		serviceMonitorSelector: servMonSelector,
		podMonitorSelector:     podMonSelector,
	}, nil
}

type PrometheusCRWatcher struct {
	logger               logr.Logger
	kubeMonitoringClient monitoringclient.Interface
	k8sClient            kubernetes.Interface
	informers            map[string]*informers.ForResource
	eventInterval        time.Duration
	stopChannel          chan struct{}
	configGenerator      *prometheus.ConfigGenerator
	kubeConfigPath       string

	serviceMonitorSelector labels.Selector
	podMonitorSelector     labels.Selector
}

func getSelector(s map[string]string) labels.Selector {
	if s == nil {
		return labels.NewSelector()
	}
	return labels.SelectorFromSet(s)
}

// getInformers returns a map of informers for the given resources.
func getInformers(factory informers.FactoriesForNamespaces) (map[string]*informers.ForResource, error) {
	serviceMonitorInformers, err := informers.NewInformersForResource(factory, monitoringv1.SchemeGroupVersion.WithResource(monitoringv1.ServiceMonitorName))
	if err != nil {
		return nil, err
	}

	podMonitorInformers, err := informers.NewInformersForResource(factory, monitoringv1.SchemeGroupVersion.WithResource(monitoringv1.PodMonitorName))
	if err != nil {
		return nil, err
	}

	return map[string]*informers.ForResource{
		monitoringv1.ServiceMonitorName: serviceMonitorInformers,
		monitoringv1.PodMonitorName:     podMonitorInformers,
	}, nil
}

// Watch wrapped informers and wait for an initial sync.
func (w *PrometheusCRWatcher) Watch(upstreamEvents chan Event, upstreamErrors chan error) error {
	success := true
	// this channel needs to be buffered because notifications are asynchronous and neither producers nor consumers wait
	notifyEvents := make(chan struct{}, 1)

	for name, resource := range w.informers {
		resource.Start(w.stopChannel)

		if ok := cache.WaitForNamedCacheSync(name, w.stopChannel, resource.HasSynced); !ok {
			success = false
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
		return fmt.Errorf("failed to sync cache")
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
	store := assets.NewStore(w.k8sClient.CoreV1(), w.k8sClient.CoreV1())
	serviceMonitorInstances := make(map[string]*monitoringv1.ServiceMonitor)
	smRetrieveErr := w.informers[monitoringv1.ServiceMonitorName].ListAll(w.serviceMonitorSelector, func(sm interface{}) {
		monitor := sm.(*monitoringv1.ServiceMonitor)
		key, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(monitor)
		validateError := w.addStoreAssetsForServiceMonitor(ctx, monitor.Name, monitor.Namespace, monitor.Spec.Endpoints, store)
		if validateError != nil {
			w.logger.Error(validateError, "Failed validating ServiceMonitor, skipping", "ServiceMonitor:", monitor.Name, "in namespace", monitor.Namespace)
		} else {
			serviceMonitorInstances[key] = monitor
		}
	})
	if smRetrieveErr != nil {
		return nil, smRetrieveErr
	}

	podMonitorInstances := make(map[string]*monitoringv1.PodMonitor)
	pmRetrieveErr := w.informers[monitoringv1.PodMonitorName].ListAll(w.podMonitorSelector, func(pm interface{}) {
		monitor := pm.(*monitoringv1.PodMonitor)
		key, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(monitor)
		validateError := w.addStoreAssetsForPodMonitor(ctx, monitor.Name, monitor.Namespace, monitor.Spec.PodMetricsEndpoints, store)
		if validateError != nil {
			w.logger.Error(validateError, "Failed validating PodMonitor, skipping", "PodMonitor:", monitor.Name, "in namespace", monitor.Namespace)
		} else {
			podMonitorInstances[key] = monitor
		}
	})
	if pmRetrieveErr != nil {
		return nil, pmRetrieveErr
	}

	generatedConfig, err := w.configGenerator.GenerateServerConfiguration(
		ctx,
		"30s",
		"",
		nil,
		nil,
		monitoringv1.TSDBSpec{},
		nil,
		nil,
		serviceMonitorInstances,
		podMonitorInstances,
		map[string]*monitoringv1.Probe{},
		map[string]*promv1alpha1.ScrapeConfig{},
		store,
		nil,
		nil,
		nil,
		[]string{})
	if err != nil {
		return nil, err
	}

	promCfg := &promconfig.Config{}
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
}

// addStoreAssetsForServiceMonitor adds authentication / authorization related information to the assets store,
// based on the service monitor and endpoints specs.
// This code borrows from
// https://github.com/prometheus-operator/prometheus-operator/blob/06b5c4189f3f72737766d86103d049115c3aff48/pkg/prometheus/resource_selector.go#L73.
func (w *PrometheusCRWatcher) addStoreAssetsForServiceMonitor(
	ctx context.Context,
	smName, smNamespace string,
	endps []monitoringv1.Endpoint,
	store *assets.Store,
) error {
	var err error
	var validateErr error
	for i, endp := range endps {
		objKey := fmt.Sprintf("serviceMonitor/%s/%s/%d", smNamespace, smName, i)

		if err = store.AddBearerToken(ctx, smNamespace, endp.BearerTokenSecret, objKey); err != nil {
			break
		}

		if err = store.AddBasicAuth(ctx, smNamespace, endp.BasicAuth, objKey); err != nil {
			break
		}

		if endp.TLSConfig != nil {
			if err = store.AddTLSConfig(ctx, smNamespace, endp.TLSConfig); err != nil {
				break
			}
		}

		if err = store.AddOAuth2(ctx, smNamespace, endp.OAuth2, objKey); err != nil {
			break
		}

		smAuthKey := fmt.Sprintf("serviceMonitor/auth/%s/%s/%d", smNamespace, smName, i)
		if err = store.AddSafeAuthorizationCredentials(ctx, smNamespace, endp.Authorization, smAuthKey); err != nil {
			break
		}

		for _, rl := range endp.RelabelConfigs {
			if rl.Action != "" {
				if validateErr = validateRelabelConfig(*rl); validateErr != nil {
					break
				}
			}
		}

		for _, rl := range endp.MetricRelabelConfigs {
			if rl.Action != "" {
				if validateErr = validateRelabelConfig(*rl); validateErr != nil {
					break
				}
			}
		}
	}

	if err != nil {
		w.logger.Error(err, "Failed to obtain credentials for a ServiceMonitor", "serviceMonitor", smName)
	}

	if validateErr != nil {
		return validateErr
	}

	return nil
}

// addStoreAssetsForServiceMonitor adds authentication / authorization related information to the assets store,
// based on the service monitor and pod metrics endpoints specs.
// This code borrows from
// https://github.com/prometheus-operator/prometheus-operator/blob/06b5c4189f3f72737766d86103d049115c3aff48/pkg/prometheus/resource_selector.go#L314.
func (w *PrometheusCRWatcher) addStoreAssetsForPodMonitor(
	ctx context.Context,
	pmName, pmNamespace string,
	podMetricsEndps []monitoringv1.PodMetricsEndpoint,
	store *assets.Store,
) error {
	var err error
	for i, endp := range podMetricsEndps {
		objKey := fmt.Sprintf("podMonitor/%s/%s/%d", pmNamespace, pmName, i)

		if err = store.AddBearerToken(ctx, pmNamespace, &endp.BearerTokenSecret, objKey); err != nil {
			break
		}

		if err = store.AddBasicAuth(ctx, pmNamespace, endp.BasicAuth, objKey); err != nil {
			break
		}

		if endp.TLSConfig != nil {
			if err = store.AddSafeTLSConfig(ctx, pmNamespace, &endp.TLSConfig.SafeTLSConfig); err != nil {
				break
			}
		}

		if err = store.AddOAuth2(ctx, pmNamespace, endp.OAuth2, objKey); err != nil {
			break
		}

		smAuthKey := fmt.Sprintf("podMonitor/auth/%s/%s/%d", pmNamespace, pmName, i)
		if err = store.AddSafeAuthorizationCredentials(ctx, pmNamespace, endp.Authorization, smAuthKey); err != nil {
			break
		}

		for _, rl := range endp.RelabelConfigs {
			if rl.Action != "" {
				if validateErr = validateRelabelConfig(*rl); validateErr != nil {
					break
				}
			}
		}

		for _, rl := range endp.MetricRelabelConfigs {
			if rl.Action != "" {
				if validateErr = validateRelabelConfig(*rl); validateErr != nil {
					break
				}
			}
		}
	}

	if err != nil {
		w.logger.Error(err, "Failed to obtain credentials for a PodMonitor", "podMonitor", pmName)
	}

	if validateErr != nil {
		return validateErr
	}

	return nil
}

// validateRelabelConfig validates relabel config for service and pod monitor,
// based on the service monitor and pod metrics endpoints specs.
// This code borrows from
// https://github.com/prometheus-operator/prometheus-operator/blob/ba536405154d18f3a6f312818283d671182af6f3/pkg/prometheus/resource_selector.go#L237
func validateRelabelConfig(rc monitoringv1.RelabelConfig) error {
	relabelTarget := regexp.MustCompile(`^(?:(?:[a-zA-Z_]|\$(?:\{\w+\}|\w+))+\w*)+$`)

	if _, err := relabel.NewRegexp(rc.Regex); err != nil {
		return fmt.Errorf("invalid regex %s for relabel configuration", rc.Regex)
	}

	if rc.Modulus == 0 && rc.Action == string(relabel.HashMod) {
		return fmt.Errorf("relabel configuration for hashmod requires non-zero modulus")
	}

	if (rc.Action == string(relabel.Replace) || rc.Action == string(relabel.HashMod) || rc.Action == string(relabel.Lowercase) || rc.Action == string(relabel.Uppercase) || rc.Action == string(relabel.KeepEqual) || rc.Action == string(relabel.DropEqual)) && rc.TargetLabel == "" {
		return fmt.Errorf("relabel configuration for %s action needs targetLabel value", rc.Action)
	}

	if (rc.Action == string(relabel.Replace) || rc.Action == string(relabel.Lowercase) || rc.Action == string(relabel.Uppercase) || rc.Action == string(relabel.KeepEqual) || rc.Action == string(relabel.DropEqual)) && !relabelTarget.MatchString(rc.TargetLabel) {
		return fmt.Errorf("%q is invalid 'target_label' for %s action", rc.TargetLabel, rc.Action)
	}

	if (rc.Action == string(relabel.Lowercase) || rc.Action == string(relabel.Uppercase) || rc.Action == string(relabel.KeepEqual) || rc.Action == string(relabel.DropEqual)) && !(rc.Replacement == relabel.DefaultRelabelConfig.Replacement || rc.Replacement == "") {
		return fmt.Errorf("'replacement' can not be set for %s action", rc.Action)
	}

	if rc.Action == string(relabel.LabelMap) {
		if rc.Replacement != "" && !relabelTarget.MatchString(rc.Replacement) {
			return fmt.Errorf("%q is invalid 'replacement' for %s action", rc.Replacement, rc.Action)
		}
	}

	if rc.Action == string(relabel.HashMod) && !model.LabelName(rc.TargetLabel).IsValid() {
		return fmt.Errorf("%q is invalid 'target_label' for %s action", rc.TargetLabel, rc.Action)
	}

	if rc.Action == string(relabel.KeepEqual) || rc.Action == string(relabel.DropEqual) {
		if !(rc.Regex == "" || rc.Regex == relabel.DefaultRelabelConfig.Regex.String()) ||
			!(rc.Modulus == uint64(0) ||
				rc.Modulus == relabel.DefaultRelabelConfig.Modulus) ||
			!(rc.Separator == "" ||
				rc.Separator == relabel.DefaultRelabelConfig.Separator) ||
			!(rc.Replacement == relabel.DefaultRelabelConfig.Replacement ||
				rc.Replacement == "") {
			return fmt.Errorf("%s action requires only 'source_labels' and `target_label`, and no other fields", rc.Action)
		}
	}

	if rc.Action == string(relabel.LabelDrop) || rc.Action == string(relabel.LabelKeep) {
		if len(rc.SourceLabels) != 0 ||
			!(rc.TargetLabel == "" ||
				rc.TargetLabel == relabel.DefaultRelabelConfig.TargetLabel) ||
			!(rc.Modulus == uint64(0) ||
				rc.Modulus == relabel.DefaultRelabelConfig.Modulus) ||
			!(rc.Separator == "" ||
				rc.Separator == relabel.DefaultRelabelConfig.Separator) ||
			!(rc.Replacement == relabel.DefaultRelabelConfig.Replacement ||
				rc.Replacement == "") {
			return fmt.Errorf("%s action requires only 'regex', and no other fields", rc.Action)
		}
	}
	return nil
}
