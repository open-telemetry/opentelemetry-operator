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

	"github.com/go-logr/logr"
	"github.com/open-telemetry/opamp-go/protobufs"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
)

const (
	CollectorResource       = "OpenTelemetryCollector"
	ResourceIdentifierKey   = "created-by"
	ResourceIdentifierValue = "operator-opamp-bridge"
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
}

var _ ConfigApplier = &Client{}

func NewClient(log logr.Logger, c client.Client, componentsAllowed map[string]map[string]bool) *Client {
	return &Client{
		log:               log,
		componentsAllowed: componentsAllowed,
		k8sClient:         c,
		close:             make(chan bool, 1),
	}
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
	err := collector.ValidateCreate()
	if err != nil {
		return err
	}
	c.log.Info("Creating collector")
	return c.k8sClient.Create(ctx, collector)
}

func (c Client) update(ctx context.Context, old *v1alpha1.OpenTelemetryCollector, new *v1alpha1.OpenTelemetryCollector) error {
	new.ObjectMeta = old.ObjectMeta
	new.TypeMeta = old.TypeMeta
	err := new.ValidateUpdate(old)
	if err != nil {
		return err
	}
	c.log.Info("Updating collector")
	return c.k8sClient.Update(ctx, new)
}

func (c Client) Apply(name string, namespace string, configmap *protobufs.AgentConfigFile) error {
	c.log.Info("Received new config", "name", name, "namespace", namespace)
	var collectorSpec v1alpha1.OpenTelemetryCollectorSpec
	err := yaml.Unmarshal(configmap.Body, &collectorSpec)
	if err != nil {
		return err
	}
	if len(collectorSpec.Config) == 0 {
		return errors.NewBadRequest("Must supply valid configuration")
	}
	reasons, validateErr := c.validate(collectorSpec)
	if validateErr != nil {
		return validateErr
	}
	if len(reasons) > 0 {
		return errors.NewBadRequest(fmt.Sprintf("Items in config are not allowed: %v", reasons))
	}
	collector := &v1alpha1.OpenTelemetryCollector{Spec: collectorSpec}
	ctx := context.Background()
	instance, err := c.GetInstance(name, namespace)
	if err != nil {
		return err
	}
	if instance != nil {
		return c.update(ctx, instance, collector)
	}
	return c.create(ctx, name, namespace, collector)
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
	err := c.k8sClient.List(ctx, &result, client.MatchingLabels{
		ResourceIdentifierKey: ResourceIdentifierValue,
	})
	if err != nil {
		return nil, err
	}
	return result.Items, nil
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
