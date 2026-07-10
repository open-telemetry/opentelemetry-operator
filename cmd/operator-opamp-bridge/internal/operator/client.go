// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/open-telemetry/opamp-go/protobufs"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

const (
	CollectorResource       = "OpenTelemetryCollector"
	ResourceIdentifierKey   = "created-by"
	ResourceIdentifierValue = "operator-opamp-bridge"
	ReportingLabelKey       = "opentelemetry.io/opamp-reporting"
	ManagedLabelKey         = "opentelemetry.io/opamp-managed"

	restartAnnotation = "kubectl.kubernetes.io/restartedAt"
)

type ConfigApplier interface {
	// Apply receives an OpAMP remote config entry name and applies the corresponding configuration.
	// Implementations define the accepted key format, but keys should match entries returned by
	// CollectorInstance.GetConfigMap.
	Apply(key string, configmap *protobufs.AgentConfigFile) error

	// Delete attempts to delete the resource identified by an OpAMP remote config entry name.
	// Implementations define the accepted key format, but keys should match entries returned by
	// CollectorInstance.GetConfigMap.
	Delete(key string) error

	// Restart triggers a rolling restart of the managed collector workload(s) in response to
	// an OpAMP ServerToAgentCommand with type CommandType_Restart.
	Restart(ctx context.Context) error

	// ListInstances retrieves all collector instances managed by the bridge.
	ListInstances() ([]CollectorInstance, error)

	// GetHealth retrieves the health of resources managed by the bridge.
	GetHealth() (Health, error)
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

func (c Client) Apply(key string, configmap *protobufs.AgentConfigFile) error {
	resource, err := kubeResourceFromKey(key)
	if err != nil {
		return err
	}
	name, namespace := resource.name, resource.namespace
	c.log.Info("Received new config", "name", name, "namespace", namespace)

	if len(configmap.Body) == 0 {
		return errors.NewBadRequest("invalid config to apply: config is empty")
	}

	var collector v1beta1.OpenTelemetryCollector
	err = yaml.Unmarshal(configmap.Body, &collector)
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

	configuredComponents := map[string]map[string]any{
		"receivers": collectorConfig.Receivers.Object,
		"exporters": collectorConfig.Exporters.Object,
	}
	if collectorConfig.Processors != nil {
		configuredComponents["processors"] = collectorConfig.Processors.Object
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

func (c Client) create(ctx context.Context, name, namespace string, collector *v1beta1.OpenTelemetryCollector) error {
	// Set the defaults
	setTypedMeta(collector)
	collector.Name = name
	collector.Namespace = namespace

	if collector.Labels == nil {
		collector.Labels = map[string]string{}
	}
	collector.Labels[ResourceIdentifierKey] = ResourceIdentifierValue

	c.log.Info("Creating collector")
	return c.k8sClient.Create(ctx, collector)
}

func (c Client) update(ctx context.Context, o, n *v1beta1.OpenTelemetryCollector) error {
	n.ObjectMeta = o.ObjectMeta
	n.TypeMeta = o.TypeMeta

	c.log.Info("Updating collector")
	return c.k8sClient.Update(ctx, n)
}

func (c Client) Delete(key string) error {
	resource, err := kubeResourceFromKey(key)
	if err != nil {
		return err
	}
	ctx := context.Background()
	result := v1beta1.OpenTelemetryCollector{}
	err = c.k8sClient.Get(ctx, client.ObjectKey{
		Namespace: resource.namespace,
		Name:      resource.name,
	}, &result)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return c.k8sClient.Delete(ctx, &result)
}

// Restart triggers a rolling restart of every managed collector's underlying workload by
// patching the pod template restart annotation, mirroring `kubectl rollout restart`.
// All collectors with the managed label are restarted; errors are joined and returned.
func (c Client) Restart(ctx context.Context) error {
	collectors, err := c.listOpenTelemetryCollectors()
	if err != nil {
		return fmt.Errorf("failed to list collectors for restart: %w", err)
	}
	var errs []error
	for i := range collectors {
		col := &collectors[i]
		workloadName := naming.Collector(col.GetName())
		if err := c.triggerRollout(ctx, col.GetNamespace(), string(col.Col.Spec.Mode), workloadName); err != nil {
			errs = append(errs, err)
		}
	}
	// errors.Join returns nil when errs is empty
	var joinedErr error
	for _, e := range errs {
		if joinedErr == nil {
			joinedErr = e
		} else {
			joinedErr = fmt.Errorf("%w; %w", joinedErr, e)
		}
	}
	return joinedErr
}

// triggerRollout patches the pod template restart annotation on the collector's managed workload,
// causing Kubernetes to perform a rolling restart identical to `kubectl rollout restart`.
// Sidecar mode collectors have no standalone workload, so they are skipped.
func (c Client) triggerRollout(ctx context.Context, namespace, mode, workloadName string) error {
	restartVal := time.Now().Format(time.RFC3339)
	switch strings.ToLower(mode) {
	case "deployment":
		deploy := &appsv1.Deployment{}
		if err := c.k8sClient.Get(ctx, client.ObjectKey{Name: workloadName, Namespace: namespace}, deploy); err != nil {
			return fmt.Errorf("failed to get Deployment %s/%s for restart: %w", namespace, workloadName, err)
		}
		if deploy.Spec.Template.Annotations == nil {
			deploy.Spec.Template.Annotations = map[string]string{}
		}
		deploy.Spec.Template.Annotations[restartAnnotation] = restartVal
		if err := c.k8sClient.Update(ctx, deploy); err != nil {
			return fmt.Errorf("failed to restart Deployment %s/%s: %w", namespace, workloadName, err)
		}
	case "daemonset":
		ds := &appsv1.DaemonSet{}
		if err := c.k8sClient.Get(ctx, client.ObjectKey{Name: workloadName, Namespace: namespace}, ds); err != nil {
			return fmt.Errorf("failed to get DaemonSet %s/%s for restart: %w", namespace, workloadName, err)
		}
		if ds.Spec.Template.Annotations == nil {
			ds.Spec.Template.Annotations = map[string]string{}
		}
		ds.Spec.Template.Annotations[restartAnnotation] = restartVal
		if err := c.k8sClient.Update(ctx, ds); err != nil {
			return fmt.Errorf("failed to restart DaemonSet %s/%s: %w", namespace, workloadName, err)
		}
	case "statefulset":
		sts := &appsv1.StatefulSet{}
		if err := c.k8sClient.Get(ctx, client.ObjectKey{Name: workloadName, Namespace: namespace}, sts); err != nil {
			return fmt.Errorf("failed to get StatefulSet %s/%s for restart: %w", namespace, workloadName, err)
		}
		if sts.Spec.Template.Annotations == nil {
			sts.Spec.Template.Annotations = map[string]string{}
		}
		sts.Spec.Template.Annotations[restartAnnotation] = restartVal
		if err := c.k8sClient.Update(ctx, sts); err != nil {
			return fmt.Errorf("failed to restart StatefulSet %s/%s: %w", namespace, workloadName, err)
		}
	case "sidecar":
		c.log.Info("Skipping restart for sidecar mode collector — no standalone workload", "name", workloadName, "namespace", namespace)
		return nil
	default:
		return fmt.Errorf("unsupported collector mode %q for restart", mode)
	}
	c.log.Info("Triggered workload rollout restart", "mode", mode, "name", workloadName, "namespace", namespace)
	return nil
}

// ListInstances returns all collectors that are visible to OpAMP as effective config entries.
func (c Client) ListInstances() ([]CollectorInstance, error) {
	collectors, err := c.listOpenTelemetryCollectors()
	if err != nil {
		return nil, err
	}
	result := make([]CollectorInstance, len(collectors))
	for i := range collectors {
		result[i] = collectors[i]
	}
	return result, nil
}

// GetHealth reports bridge root health with managed collector health as children.
func (c Client) GetHealth() (Health, error) {
	collectors, err := c.listOpenTelemetryCollectors()
	if err != nil {
		return Health{}, err
	}
	healthMap := map[string]Health{}
	for _, col := range collectors {
		podMap, err := c.generateCollectorHealth(col.selectorLabels(), col.GetNamespace())
		if err != nil {
			return Health{}, err
		}
		isPoolHealthy := true
		for _, pod := range podMap {
			isPoolHealthy = isPoolHealthy && pod.Healthy
		}
		healthMap[NewKubeResourceKey(col.GetNamespace(), col.GetName()).String()] = Health{
			StartTime: col.Col.GetCreationTimestamp().Time,
			Status:    col.Col.Status.Scale.StatusReplicas,
			Children:  podMap,
			Healthy:   isPoolHealthy,
		}
	}
	return Health{
		Healthy:  true,
		Children: healthMap,
	}, nil
}

// generateCollectorHealth reports pod health for one collector selected by labels within a namespace.
func (c Client) generateCollectorHealth(selectorLabels map[string]string, namespace string) (map[string]Health, error) {
	pods, err := c.getCollectorPods(selectorLabels, namespace)
	if err != nil {
		return nil, err
	}
	healthMap := map[string]Health{}
	for _, item := range pods.Items {
		key := NewKubeResourceKey(item.GetNamespace(), item.GetName())
		healthy := true
		if item.Status.Phase != "Running" {
			healthy = false
		}
		var startTime time.Time
		if item.Status.StartTime != nil {
			startTime = item.Status.StartTime.Time
		} else {
			healthy = false
		}
		healthMap[key.String()] = Health{
			StartTime: startTime,
			Status:    string(item.Status.Phase),
			Healthy:   healthy,
			Children:  map[string]Health{},
		}
	}
	return healthMap, nil
}

// listOpenTelemetryCollectors returns collectors managed by this bridge plus reporting-only collectors.
func (c Client) listOpenTelemetryCollectors() ([]CRDInstance, error) {
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
		setTypedMeta(&instances[i])
	}

	result := make([]CRDInstance, len(instances))
	for i := range instances {
		result[i] = newCRDInstance(instances[i])
	}
	return result, nil
}

func (c Client) GetInstance(name, namespace string) (*v1beta1.OpenTelemetryCollector, error) {
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
	setTypedMeta(&result)
	return &result, nil
}

func (c Client) getCollectorPods(selectorLabels map[string]string, namespace string) (*v1.PodList, error) {
	ctx := context.Background()
	podList := &v1.PodList{}
	err := c.k8sClient.List(ctx, podList, client.MatchingLabels(selectorLabels), client.InNamespace(namespace))
	return podList, err
}

// setTypedMeta sets the TypeMeta of the given collector to the correct values. The controller-runtime
// client will not set the TypeMeta for us, so we need to do it manually.
func setTypedMeta(collector *v1beta1.OpenTelemetryCollector) {
	collector.Kind = CollectorResource
	collector.APIVersion = v1beta1.GroupVersion.String()
}
