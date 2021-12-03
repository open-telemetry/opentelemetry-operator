package watcher

import (
	"context"
	"fmt"
	"github.com/go-kit/log"
	allocatorconfig "github.com/otel-allocator/config"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/prometheus-operator/prometheus-operator/pkg/assets"
	monitoringclient "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	"github.com/prometheus-operator/prometheus-operator/pkg/informers"
	"github.com/prometheus-operator/prometheus-operator/pkg/prometheus"
	promconfig "github.com/prometheus/prometheus/config"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

func newCRDMonitorWatcher() {

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
	factory := informers.NewMonitoringInformerFactories(map[string]struct{}{}, map[string]struct{}{}, mclient, allocatorconfig.DefaultResyncTime, nil) //TODO decide what strategy to use regarding namespaces

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
