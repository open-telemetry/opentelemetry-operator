// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/open-telemetry/opamp-go/protobufs"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/operator"
)

const (
	managedByLabel       = "opentelemetry.io/managed-by"
	managedByValue       = "opamp-bridge-standalone"
	restartAnnotation    = "kubectl.kubernetes.io/restartedAt"
	rolloutAnnotationKey = "opentelemetry.io/opamp-rollout-target"
)

var _ operator.ConfigApplier = &Client{}

// Client implements operator.ConfigApplier for standalone mode.
// ConfigMaps are the primary managed objects.
type Client struct {
	log       logr.Logger
	k8sClient client.Client
	restCfg   *rest.Config
	name      string
	cmCache   cache.Cache
	onUpdate  func()
}

// NewClient creates a standalone OpAMP bridge Client that works directly on ConfigMaps
// without the need for CRDs or the operator.
func NewClient(name string, log logr.Logger, c client.Client, restCfg *rest.Config, onUpdate func()) *Client {
	return &Client{
		log:       log,
		k8sClient: c,
		restCfg:   restCfg,
		name:      name,
		onUpdate:  onUpdate,
	}
}

// Start creates a label-filtered informer cache for managed ConfigMaps.
func (c *Client) Start(ctx context.Context) error {
	managedSelector := labels.SelectorFromSet(labels.Set{managedByLabel: managedByValue})

	ca, err := cache.New(c.restCfg, cache.Options{
		ByObject: map[client.Object]cache.ByObject{
			&v1.ConfigMap{}: {Label: managedSelector},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create ConfigMap cache: %w", err)
	}

	informer, err := ca.GetInformer(ctx, &v1.ConfigMap{})
	if err != nil {
		return fmt.Errorf("failed to get ConfigMap informer: %w", err)
	}

	handler := toolscache.ResourceEventHandlerFuncs{
		AddFunc:    func(_ any) { c.onUpdate() },
		UpdateFunc: func(_, _ any) { c.onUpdate() },
		DeleteFunc: func(_ any) { c.onUpdate() },
	}
	if _, err = informer.AddEventHandler(handler); err != nil {
		return fmt.Errorf("failed to add ConfigMap event handler: %w", err)
	}

	go func() {
		if err := ca.Start(ctx); err != nil {
			c.log.Error(err, "ConfigMap cache stopped with error")
		}
	}()

	if !ca.WaitForCacheSync(ctx) {
		return errors.New("timed out waiting for ConfigMap cache to sync")
	}

	c.cmCache = ca
	c.log.Info("ConfigMap informer cache synced")
	return nil
}

// Apply writes configFile data into the ConfigMap identified by name/namespace,
// creating it if it does not exist. The body must be a YAML-encoded standaloneConfig.
func (c *Client) Apply(name, namespace string, configFile *protobufs.AgentConfigFile) error {
	if len(configFile.Body) == 0 {
		return errors.New("invalid config to apply: config is empty")
	}

	var received standaloneConfig
	if err := yaml.Unmarshal(configFile.Body, &received); err != nil {
		return fmt.Errorf("failed to unmarshal config body into standalone config: %w", err)
	}
	if err := received.validate(name, namespace); err != nil {
		return fmt.Errorf("invalid standalone config: %w", err)
	}

	ctx := context.Background()

	existing := &v1.ConfigMap{}
	err := c.k8sClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, existing)
	if apierrors.IsNotFound(err) {
		cm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels:    map[string]string{managedByLabel: managedByValue},
			},
			Data: received.Config,
		}
		if createErr := c.k8sClient.Create(ctx, cm); createErr != nil {
			return fmt.Errorf("failed to create ConfigMap %s/%s: %w", namespace, name, createErr)
		}
		c.log.Info("Created ConfigMap", "name", name, "namespace", namespace)
		return c.triggerRollout(ctx, namespace, parseWorkloadAnnotation(cm.Annotations[rolloutAnnotationKey]))
	} else if err != nil {
		return fmt.Errorf("failed to get ConfigMap %s/%s: %w", namespace, name, err)
	}
	if existing.Labels[managedByLabel] != managedByValue {
		return fmt.Errorf("cannot modify unmanaged ConfigMap %s/%s", namespace, name)
	}

	workloads := parseWorkloadAnnotation(existing.Annotations[rolloutAnnotationKey])

	existing.Data = received.Config
	if updateErr := c.k8sClient.Update(ctx, existing); updateErr != nil {
		return fmt.Errorf("failed to update ConfigMap %s/%s: %w", namespace, name, updateErr)
	}
	c.log.Info("Updated ConfigMap", "name", name, "namespace", namespace)

	return c.triggerRollout(ctx, namespace, workloads)
}

// Delete removes the ConfigMap identified by name/namespace.
func (c *Client) Delete(name, namespace string) error {
	ctx := context.Background()

	cm := &v1.ConfigMap{}
	err := c.k8sClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, cm)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get ConfigMap %s/%s: %w", namespace, name, err)
	}
	if cm.Labels[managedByLabel] != managedByValue {
		return fmt.Errorf("cannot delete unmanaged ConfigMap %s/%s", namespace, name)
	}
	return c.k8sClient.Delete(ctx, cm)
}

// ListInstances returns one CollectorInstance per managed ConfigMap by reading
// from the local informer cache (no API server call).
func (c *Client) ListInstances() ([]operator.CollectorInstance, error) {
	cmList := &v1.ConfigMapList{}
	if c.cmCache != nil {
		if err := c.cmCache.List(context.Background(), cmList); err != nil {
			return nil, fmt.Errorf("failed to list managed ConfigMaps from cache: %w", err)
		}
	} else if err := c.k8sClient.List(context.Background(), cmList, client.MatchingLabels{managedByLabel: managedByValue}); err != nil {
		return nil, fmt.Errorf("failed to list managed ConfigMaps from cache: %w", err)
	}

	result := make([]operator.CollectorInstance, 0, len(cmList.Items))
	for i := range cmList.Items {
		cm := &cmList.Items[i]
		configBody := marshalConfigMap(cm)
		if len(cm.Data) > 0 && configBody == nil {
			c.log.Error(nil, "Failed to marshal ConfigMap data; instance will have no effective config",
				"name", cm.Name, "namespace", cm.Namespace)
		}
		result = append(result, &standaloneCollectorInstance{
			name:       cm.Name,
			namespace:  cm.Namespace,
			createdAt:  cm.GetCreationTimestamp().Time,
			configBody: configBody,
		})
	}
	return result, nil
}

// GetCollectorPods is not used in standalone mode (pod health is not reported).
func (*Client) GetCollectorPods(_ map[string]string, _ string) (*v1.PodList, error) {
	return &v1.PodList{}, nil
}

// triggerRollout patches the pod template of each workload in the list with a
// restart annotation, causing a rolling restart.
func (c *Client) triggerRollout(ctx context.Context, namespace string, workloads []string) error {
	restartVal := time.Now().Format(time.RFC3339)

	for _, ref := range workloads {
		parts := strings.SplitN(ref, "/", 2)
		if len(parts) != 2 {
			c.log.Error(fmt.Errorf("invalid workload reference %q", ref), "skipping rollout entry")
			continue
		}
		kind, name := parts[0], parts[1]

		switch strings.ToLower(kind) {
		case "deployment":
			deploy := &appsv1.Deployment{}
			if err := c.k8sClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, deploy); err != nil {
				c.log.Error(err, "failed to get Deployment for rollout", "name", name, "namespace", namespace)
				continue
			}
			if deploy.Spec.Template.Annotations == nil {
				deploy.Spec.Template.Annotations = map[string]string{}
			}
			deploy.Spec.Template.Annotations[restartAnnotation] = restartVal
			if err := c.k8sClient.Update(ctx, deploy); err != nil {
				c.log.Error(err, "failed to trigger rollout for Deployment", "name", name, "namespace", namespace)
			}
		case "daemonset":
			ds := &appsv1.DaemonSet{}
			if err := c.k8sClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, ds); err != nil {
				c.log.Error(err, "failed to get DaemonSet for rollout", "name", name, "namespace", namespace)
				continue
			}
			if ds.Spec.Template.Annotations == nil {
				ds.Spec.Template.Annotations = map[string]string{}
			}
			ds.Spec.Template.Annotations[restartAnnotation] = restartVal
			if err := c.k8sClient.Update(ctx, ds); err != nil {
				c.log.Error(err, "failed to trigger rollout for DaemonSet", "name", name, "namespace", namespace)
			}
		default:
			c.log.Error(fmt.Errorf("unsupported workload kind %q", kind), "skipping rollout entry")
		}
	}
	return nil
}

func parseWorkloadAnnotation(annotation string) []string {
	if annotation == "" {
		return nil
	}
	parts := strings.Split(annotation, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func marshalConfigMap(cm *v1.ConfigMap) []byte {
	if len(cm.Data) == 0 {
		return nil
	}
	wire := standaloneConfig{
		Version:   standaloneConfigVersion,
		Name:      cm.Name,
		Namespace: cm.Namespace,
		Config:    cm.Data,
	}
	b, err := yaml.Marshal(wire)
	if err != nil {
		return nil
	}
	return b
}
