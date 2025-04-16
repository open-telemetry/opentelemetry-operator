// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/open-telemetry/opamp-go/protobufs"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
)

const (
	CollectorResource       = "OpenTelemetryCollector"
	ResourceIdentifierKey   = "created-by"
	ResourceIdentifierValue = "operator-opamp-bridge"
	ReportingLabelKey       = "opentelemetry.io/opamp-reporting"
	ManagedLabelKey         = "opentelemetry.io/opamp-managed"
)

type ConfigApplier interface {
	// Apply receives a name and namespace to apply an OpenTelemetryCollector CRD that is contained in the configmap.
	Apply(name string, namespace string, configmap *protobufs.AgentConfigFile) error

	// Delete attempts to delete an OpenTelemetryCollector object given a name and namespace.
	Delete(name string, namespace string) error

	// ListInstances retrieves all OpenTelemetryCollector CRDs created by the operator-opamp-bridge agent.
	ListInstances() ([]v1beta1.OpenTelemetryCollector, error)

	// GetInstance retrieves an OpenTelemetryCollector CRD given a name and namespace.
	GetInstance(name string, namespace string) (*v1beta1.OpenTelemetryCollector, error)

	// GetCollectorPods retrieves all pods that match the given collector's selector labels and namespace.
	GetCollectorPods(selectorLabels map[string]string, namespace string) (*v1.PodList, error)
}

type Client struct {
	log               logr.Logger
	componentsAllowed map[string]map[string]bool
	k8sClient         client.Client
	close             chan bool
	name              string
}

var _ ConfigApplier = &Client{}

func NewClient(name string, log logr.Logger, c client.Client, componentsAllowed map[string]map[string]bool) *Client {
	return &Client{
		log:               log,
		componentsAllowed: componentsAllowed,
		k8sClient:         c,
		close:             make(chan bool, 1),
		name:              name,
	}
}

func (c Client) Apply(name string, namespace string, configmap *protobufs.AgentConfigFile) error {
	c.log.Info("Received new config", "name", name, "namespace", namespace)

	if len(configmap.Body) == 0 {
		return errors.NewBadRequest("invalid config to apply: config is empty")
	}

	var collector v1beta1.OpenTelemetryCollector
	err := yaml.Unmarshal(configmap.Body, &collector)
	if err != nil {
		return errors.NewBadRequest(fmt.Sprintf("failed to unmarshal config into v1beta1 API Version: %v", err))
	}

	err = c.validateComponents(&collector.Spec.Config)
	if err != nil {
		return err
	}

	ctx := context.Background()
	updatedCollector := collector.DeepCopy()
	instance, err := c.GetInstance(name, namespace)
	if err != nil {
		return err
	}

	err = c.validateLabels(instance)
	if err != nil {
		return err
	}
	err = c.validateLabels(updatedCollector)
	if err != nil {
		return err
	}

	if instance == nil {
		return c.create(ctx, name, namespace, updatedCollector)
	}
	return c.update(ctx, instance, updatedCollector)
}

func (c Client) validateComponents(collectorConfig *v1beta1.Config) error {
	if len(c.componentsAllowed) == 0 {
		return nil
	}

	configuredComponents := map[string]map[string]interface{}{
		"receivers":  collectorConfig.Receivers.Object,
		"processors": collectorConfig.Processors.Object,
		"exporters":  collectorConfig.Exporters.Object,
	}

	var invalidComponents []string
	for component, componentMap := range configuredComponents {
		if _, ok := c.componentsAllowed[component]; !ok {
			invalidComponents = append(invalidComponents, component)
			continue
		}
		for componentName := range componentMap {
			if _, ok := c.componentsAllowed[component][componentName]; !ok {
				invalidComponents = append(invalidComponents, fmt.Sprintf("%s.%s", component, componentName))
			}
		}
	}

	if len(invalidComponents) > 0 {
		return errors.NewBadRequest(fmt.Sprintf("Items in config are not allowed: %v", invalidComponents))
	}

	return nil
}

func (c Client) validateLabels(collector *v1beta1.OpenTelemetryCollector) error {
	if collector == nil {
		return nil
	}

	resourceLabels := collector.GetLabels()

	// If either the received collector resource has labels indicating it should only report and is not managed,
	// disallow applying the new collector config
	if labelSetContainsLabel(resourceLabels, ReportingLabelKey, "true") {
		return errors.NewBadRequest(fmt.Sprintf("cannot modify a collector with `%s: true`", ReportingLabelKey))
	}

	// If either the collector doesn't have the managed label set to true, it should disallow applying the new collector
	// config
	if !labelSetContainsLabel(resourceLabels, ManagedLabelKey, "true") &&
		!labelSetContainsLabel(resourceLabels, ManagedLabelKey, c.name) {
		return errors.NewBadRequest(fmt.Sprintf("cannot modify a collector that doesn't have `%s: true | <bridge-name>` set", ManagedLabelKey))
	}

	return nil
}

func labelSetContainsLabel(resourceLabelSet map[string]string, label, value string) bool {
	if len(resourceLabelSet) == 0 {
		return false
	}
	return strings.EqualFold(resourceLabelSet[label], value)
}

func (c Client) create(ctx context.Context, name string, namespace string, collector *v1beta1.OpenTelemetryCollector) error {
	// Set the defaults
	collector.TypeMeta.Kind = CollectorResource
	collector.TypeMeta.APIVersion = v1beta1.GroupVersion.String()
	collector.ObjectMeta.Name = name
	collector.ObjectMeta.Namespace = namespace

	if collector.ObjectMeta.Labels == nil {
		collector.ObjectMeta.Labels = map[string]string{}
	}
	collector.ObjectMeta.Labels[ResourceIdentifierKey] = ResourceIdentifierValue

	c.log.Info("Creating collector")
	return c.k8sClient.Create(ctx, collector)
}

func (c Client) update(ctx context.Context, old *v1beta1.OpenTelemetryCollector, new *v1beta1.OpenTelemetryCollector) error {
	new.ObjectMeta = old.ObjectMeta
	new.TypeMeta = old.TypeMeta

	c.log.Info("Updating collector")
	return c.k8sClient.Update(ctx, new)
}

func (c Client) Delete(name string, namespace string) error {
	ctx := context.Background()
	result := v1beta1.OpenTelemetryCollector{}
	err := c.k8sClient.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, &result)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return c.k8sClient.Delete(ctx, &result)
}

func (c Client) ListInstances() ([]v1beta1.OpenTelemetryCollector, error) {
	ctx := context.Background()

	var instances []v1beta1.OpenTelemetryCollector

	labelSelector := labels.NewSelector()
	requirement, err := labels.NewRequirement(ManagedLabelKey, selection.In, []string{c.name, "true"})
	if err != nil {
		return nil, err
	}
	managedCollectorLabelSelector := client.MatchingLabelsSelector{Selector: labelSelector.Add(*requirement)}

	managedCollectors := v1beta1.OpenTelemetryCollectorList{}
	err = c.k8sClient.List(ctx, &managedCollectors, managedCollectorLabelSelector)
	if err != nil {
		return nil, err
	}
	instances = append(instances, managedCollectors.Items...)

	reportingCollectorLabelMatcher := client.MatchingLabels{ReportingLabelKey: "true"}
	reportingCollectors := v1beta1.OpenTelemetryCollectorList{}
	err = c.k8sClient.List(ctx, &reportingCollectors, reportingCollectorLabelMatcher)
	if err != nil {
		return nil, err
	}
	instances = append(instances, reportingCollectors.Items...)

	for i := range instances {
		instances[i].SetManagedFields(nil)
	}

	return instances, nil
}

func (c Client) GetInstance(name string, namespace string) (*v1beta1.OpenTelemetryCollector, error) {
	ctx := context.Background()
	result := v1beta1.OpenTelemetryCollector{}

	err := c.k8sClient.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, &result)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

func (c Client) GetCollectorPods(selectorLabels map[string]string, namespace string) (*v1.PodList, error) {
	ctx := context.Background()
	podList := &v1.PodList{}
	err := c.k8sClient.List(ctx, podList, client.MatchingLabels(selectorLabels), client.InNamespace(namespace))
	return podList, err
}
