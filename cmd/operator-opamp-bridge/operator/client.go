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

package operator

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/open-telemetry/opamp-go/protobufs"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
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

	// GetInstance retrieves an OpenTelemetryCollector CRD given a name and namespace.
	GetInstance(name string, namespace string) (*v1alpha1.OpenTelemetryCollector, error)

	// ListInstances retrieves all OpenTelemetryCollector CRDs created by the operator-opamp-bridge agent.
	ListInstances() ([]v1alpha1.OpenTelemetryCollector, error)

	// Delete attempts to delete an OpenTelemetryCollector object given a name and namespace.
	Delete(name string, namespace string) error
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

func (c Client) labelSetContainsLabel(instance *v1alpha1.OpenTelemetryCollector, label, value string) bool {
	if instance == nil || instance.GetLabels() == nil {
		return false
	}
	if labels := instance.GetLabels(); labels != nil && strings.EqualFold(labels[label], value) {
		return true
	}
	return false
}

func (c Client) create(ctx context.Context, name string, namespace string, collector *v1alpha1.OpenTelemetryCollector) error {
	// Set the defaults
	collector.Default()
	collector.TypeMeta.Kind = CollectorResource
	collector.TypeMeta.APIVersion = v1alpha1.GroupVersion.String()
	collector.ObjectMeta.Name = name
	collector.ObjectMeta.Namespace = namespace

	if collector.ObjectMeta.Labels == nil {
		collector.ObjectMeta.Labels = map[string]string{}
	}
	collector.ObjectMeta.Labels[ResourceIdentifierKey] = ResourceIdentifierValue
	warnings, err := collector.ValidateCreate()
	if err != nil {
		return err
	}
	if warnings != nil {
		c.log.Info("Some warnings present on collector", "warnings", warnings)
	}
	c.log.Info("Creating collector")
	return c.k8sClient.Create(ctx, collector)
}

func (c Client) update(ctx context.Context, old *v1alpha1.OpenTelemetryCollector, new *v1alpha1.OpenTelemetryCollector) error {
	new.ObjectMeta = old.ObjectMeta
	new.TypeMeta = old.TypeMeta
	warnings, err := new.ValidateUpdate(old)
	if err != nil {
		return err
	}
	if warnings != nil {
		c.log.Info("Some warnings present on collector", "warnings", warnings)
	}
	c.log.Info("Updating collector")
	return c.k8sClient.Update(ctx, new)
}

func (c Client) Apply(name string, namespace string, configmap *protobufs.AgentConfigFile) error {
	c.log.Info("Received new config", "name", name, "namespace", namespace)
	var collector v1alpha1.OpenTelemetryCollector
	err := yaml.Unmarshal(configmap.Body, &collector)
	if err != nil {
		return err
	}
	if len(collector.Spec.Config) == 0 {
		return errors.NewBadRequest("Must supply valid configuration")
	}
	reasons, validateErr := c.validate(collector.Spec)
	if validateErr != nil {
		return validateErr
	}
	if len(reasons) > 0 {
		return errors.NewBadRequest(fmt.Sprintf("Items in config are not allowed: %v", reasons))
	}
	updatedCollector := collector.DeepCopy()
	ctx := context.Background()
	instance, err := c.GetInstance(name, namespace)
	if err != nil {
		return err
	}
	// If either the received collector or the collector being created has reporting set to true, it should be denied
	if c.labelSetContainsLabel(instance, ReportingLabelKey, "true") ||
		c.labelSetContainsLabel(updatedCollector, ReportingLabelKey, "true") {
		return errors.NewBadRequest("cannot modify a collector with `opentelemetry.io/opamp-reporting: true`")
	}
	// If either the received collector or the collector doesn't have the managed label set to true, it should be denied
	if !c.labelSetContainsLabel(instance, ManagedLabelKey, "true") &&
		!c.labelSetContainsLabel(instance, ManagedLabelKey, c.name) &&
		!c.labelSetContainsLabel(updatedCollector, ManagedLabelKey, "true") &&
		!c.labelSetContainsLabel(updatedCollector, ManagedLabelKey, c.name) {
		return errors.NewBadRequest("cannot modify a collector that doesn't have `opentelemetry.io/opamp-managed: true | <bridge-name>` set")
	}
	if instance == nil {
		return c.create(ctx, name, namespace, updatedCollector)
	}
	return c.update(ctx, instance, updatedCollector)
}

func (c Client) Delete(name string, namespace string) error {
	ctx := context.Background()
	result := v1alpha1.OpenTelemetryCollector{}
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

func (c Client) ListInstances() ([]v1alpha1.OpenTelemetryCollector, error) {
	ctx := context.Background()
	result := v1alpha1.OpenTelemetryCollectorList{}
	labelSelector := labels.NewSelector()
	requirement, err := labels.NewRequirement(ManagedLabelKey, selection.In, []string{c.name, "true"})
	if err != nil {
		return nil, err
	}
	err = c.k8sClient.List(ctx, &result, client.MatchingLabelsSelector{Selector: labelSelector.Add(*requirement)})
	if err != nil {
		return nil, err
	}
	reportingCollectors := v1alpha1.OpenTelemetryCollectorList{}
	err = c.k8sClient.List(ctx, &reportingCollectors, client.MatchingLabels{
		ReportingLabelKey: "true",
	})
	if err != nil {
		return nil, err
	}
	items := append(result.Items, reportingCollectors.Items...)
	for i := range items {
		items[i].SetManagedFields(nil)
	}

	return items, nil
}

func (c Client) GetInstance(name string, namespace string) (*v1alpha1.OpenTelemetryCollector, error) {
	ctx := context.Background()
	result := v1alpha1.OpenTelemetryCollector{}
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

func (c Client) validate(spec v1alpha1.OpenTelemetryCollectorSpec) ([]string, error) {
	// Do not use this feature if it's not specified
	if c.componentsAllowed == nil || len(c.componentsAllowed) == 0 {
		return nil, nil
	}
	collectorConfig := make(map[string]map[string]interface{})
	err := yaml.Unmarshal([]byte(spec.Config), &collectorConfig)
	if err != nil {
		return nil, err
	}
	var invalidComponents []string
	for component, componentMap := range collectorConfig {
		if component == "service" {
			// We don't care about what's in the service pipelines.
			// Only components declared in the configuration can be used in the service pipeline.
			continue
		}
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
	return invalidComponents, nil
}
