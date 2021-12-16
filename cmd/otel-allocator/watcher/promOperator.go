package watcher

import (
	"context"
	"fmt"
	"github.com/go-kit/log"
	"github.com/go-logr/logr"
	allocatorconfig "github.com/otel-allocator/config"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/prometheus-operator/prometheus-operator/pkg/assets"
	monitoringclient "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	"github.com/prometheus-operator/prometheus-operator/pkg/informers"
	"github.com/prometheus-operator/prometheus-operator/pkg/prometheus"
	promconfig "github.com/prometheus/prometheus/config"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

func newCRDMonitorWatcher(logger logr.Logger, config allocatorconfig.CLIConfig) (*PrometheusCRWatcher, error) {
	mClient, err := monitoringclient.NewForConfig(config.ClusterConfig)
	if err != nil {
		return nil, err
	}

	factory := informers.NewMonitoringInformerFactories(map[string]struct{}{v1.NamespaceAll: {}}, map[string]struct{}{}, mClient, allocatorconfig.DefaultResyncTime, nil) //TODO decide what strategy to use regarding namespaces

	serviceMonitorInformers, err := informers.NewInformersForResource(factory, monitoringv1.SchemeGroupVersion.WithResource(monitoringv1.ServiceMonitorName))
	if err != nil {
		return nil, err
	}

	podMonitorInformers, err := informers.NewInformersForResource(factory, monitoringv1.SchemeGroupVersion.WithResource(monitoringv1.PodMonitorName))
	if err != nil {
		return nil, err
	}

	monitoringInformers := map[string]*informers.ForResource{
		monitoringv1.ServiceMonitorName: serviceMonitorInformers,
		monitoringv1.PodMonitorName:     podMonitorInformers,
	}

	generator := prometheus.NewConfigGenerator(log.NewNopLogger()) // TODO replace Nop?

	return &PrometheusCRWatcher{
		kubeMonitoringClient: mClient,
		informers:            monitoringInformers,
		stopChannel:          make(chan struct{}),
		Errors:               make(chan error),
		Events:               make(chan Event),
		configGenerator:      generator,
	}, nil
}

type PrometheusCRWatcher struct {
	kubeMonitoringClient *monitoringclient.Clientset
	informers            map[string]*informers.ForResource
	stopChannel          chan struct{}
	Errors               chan error
	Events               chan Event
	configGenerator      *prometheus.ConfigGenerator
}

// Start wrapped informers and wait for an initial sync
func (w *PrometheusCRWatcher) Start() error {
	event := Event{Source: EventSourcePrometheusCR}
	success := true

	for name, resource := range w.informers {
		go resource.Start(w.stopChannel)

		if ok := cache.WaitForNamedCacheSync(name, w.stopChannel, resource.HasSynced); !ok {
			success = false
		}

		resource.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				w.Events <- event
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				w.Events <- event
			},
			DeleteFunc: func(obj interface{}) {
				w.Events <- event
			},
		})
	}
	if !success {
		return fmt.Errorf("failed to sync cache")
	}

	return nil
}

func (w *PrometheusCRWatcher) Close() error {
	w.stopChannel <- struct{}{}
	return nil
}

func (w *PrometheusCRWatcher) CreatePromConfig() (*promconfig.Config, error) {
	serviceMonitorInstances := make(map[string]*monitoringv1.ServiceMonitor)
	smRetrieveErr := w.informers[monitoringv1.ServiceMonitorName].ListAll(labels.NewSelector(), func(sm interface{}) {
		monitor := sm.(*monitoringv1.ServiceMonitor)
		key, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(monitor)
		serviceMonitorInstances[key] = monitor
	})
	if smRetrieveErr != nil {
		return nil, smRetrieveErr
	}

	podMonitorInstances := make(map[string]*monitoringv1.PodMonitor)
	pmRetrieveErr := w.informers[monitoringv1.PodMonitorName].ListAll(labels.NewSelector(), func(pm interface{}) {
		monitor := pm.(*monitoringv1.PodMonitor)
		key, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(monitor)
		podMonitorInstances[key] = monitor
	})
	if pmRetrieveErr != nil {
		return nil, pmRetrieveErr
	}

	store := assets.Store{
		TLSAssets:       nil,
		TokenAssets:     nil,
		BasicAuthAssets: nil,
		OAuth2Assets:    nil,
		SigV4Assets:     nil,
	}
	generatedConfig, err := w.configGenerator.GenerateConfig(&monitoringv1.Prometheus{}, serviceMonitorInstances, podMonitorInstances, map[string]*monitoringv1.Probe{}, &store, nil, nil, nil, []string{})
	if err != nil {
		return nil, err
	}

	promCfg := &promconfig.Config{}
	unmarshalErr := yaml.Unmarshal(generatedConfig, promCfg)
	if unmarshalErr != nil {
		return nil, unmarshalErr
	}

	return promCfg, nil
}

func test(kubecfg *rest.Config, client kubernetes.Interface, config2 allocatorconfig.Config, ctx context.Context, logger log.Logger) error {
	client, err := kubernetes.NewForConfig(kubecfg)
	if err != nil {
		return fmt.Errorf("instantiating kubernetes client failed", err)
	}

	mclient, err := monitoringclient.NewForConfig(kubecfg)
	if err != nil {
		return fmt.Errorf("instantiating kubernetes client failed", err)
	}
	factory := informers.NewMonitoringInformerFactories(map[string]struct{}{v1.NamespaceAll: {}}, map[string]struct{}{}, mclient, allocatorconfig.DefaultResyncTime, nil) //TODO decide what strategy to use regarding namespaces

	serviceMonitorInformers, _ := informers.NewInformersForResource(factory, monitoringv1.SchemeGroupVersion.WithResource(monitoringv1.ServiceMonitorName))
	podMonitorInformers, _ := informers.NewInformersForResource(factory, monitoringv1.SchemeGroupVersion.WithResource(monitoringv1.PodMonitorName))

	go serviceMonitorInformers.Start(ctx.Done())
	go podMonitorInformers.Start(ctx.Done())

	serviceMonitorSuccess := cache.WaitForNamedCacheSync(monitoringv1.ServiceMonitorName, ctx.Done(), serviceMonitorInformers.HasSynced)
	podMonitorSuccess := cache.WaitForNamedCacheSync(monitoringv1.PodMonitorName, ctx.Done(), podMonitorInformers.HasSynced)
	if !serviceMonitorSuccess || !podMonitorSuccess {
		//failure //TODO add error handling
	}

	serviceMonitorInformers.AddEventHandler(cache.ResourceEventHandlerFuncs{ //TODO add update functions
		AddFunc:    nil,
		UpdateFunc: nil,
		DeleteFunc: nil,
	})

	podMonitorInformers.AddEventHandler(cache.ResourceEventHandlerFuncs{ //TODO add update functions
		AddFunc:    nil,
		UpdateFunc: nil,
		DeleteFunc: nil,
	})

	var serviceMonitorInstances map[string]*monitoringv1.ServiceMonitor

	_ = serviceMonitorInformers.ListAll(nil, func(sm interface{}) {
		monitor := sm.(*monitoringv1.ServiceMonitor)
		key, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(monitor)
		serviceMonitorInstances[key] = monitor
	})

	var podMonitorInstances map[string]*monitoringv1.PodMonitor
	_ = podMonitorInformers.ListAll(nil, func(pm interface{}) {
		monitor := pm.(*monitoringv1.PodMonitor)
		key, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(monitor)
		podMonitorInstances[key] = monitor
	})

	configGenerator := prometheus.NewConfigGenerator(logger)
	var probeInstances map[string]*monitoringv1.Probe
	store := assets.Store{
		TLSAssets:       nil,
		TokenAssets:     nil,
		BasicAuthAssets: nil,
		OAuth2Assets:    nil,
		SigV4Assets:     nil,
	}
	generatetConfig, err := configGenerator.GenerateConfig(&monitoringv1.Prometheus{}, serviceMonitorInstances, podMonitorInstances, probeInstances, &store, nil, nil, nil, []string{})
	if err != nil {
		return err
	}

	promcfg := promconfig.Config{}
	_ = yaml.Unmarshal(generatetConfig, promcfg)

	return nil
}
