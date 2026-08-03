// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	"github.com/open-telemetry/opamp-go/protobufs"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/config"
	bridgemanager "github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/manager"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/operator"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/rollout"
)

// Client implements operator.ConfigApplier for standalone mode.
// ConfigMaps are the primary managed objects.
type Client struct {
	log             logr.Logger
	k8sClient       client.Client
	restCfg         *rest.Config
	cmCache         cache.Cache
	onUpdate        func()
	watchNamespaces []string
	healthMux       sync.RWMutex
	healthUpdaters  map[workloadKey]func() error
}

type workloadKey struct {
	namespace    string
	workloadType string
	workloadName string
}

// NewClient creates a standalone OpAMP bridge Client that works directly on ConfigMaps
// without the need for CRDs or the operator.
func NewClient(log logr.Logger, c client.Client, restCfg *rest.Config, onUpdate func(), agents ...config.StandaloneAgentConfig) *Client {
	return &Client{
		log:             log,
		k8sClient:       c,
		restCfg:         restCfg,
		onUpdate:        onUpdate,
		watchNamespaces: namespacesForAgents(agents),
		healthUpdaters:  map[workloadKey]func() error{},
	}
}

// Start creates an informer cache for ConfigMaps so configured agents can
// refresh their effective config when local data changes.
func (c *Client) Start(ctx context.Context) error {
	cacheOptions := cache.Options{
		ByObject: map[client.Object]cache.ByObject{
			&v1.ConfigMap{}:       {},
			&appsv1.Deployment{}:  {},
			&appsv1.DaemonSet{}:   {},
			&appsv1.StatefulSet{}: {},
		},
	}
	if len(c.watchNamespaces) > 0 {
		cacheOptions.DefaultNamespaces = namespaceCacheConfig(c.watchNamespaces)
	}
	ca, err := cache.New(c.restCfg, cacheOptions)
	if err != nil {
		return fmt.Errorf("failed to create standalone cache: %w", err)
	}

	configMapInformer, err := ca.GetInformer(ctx, &v1.ConfigMap{})
	if err != nil {
		return fmt.Errorf("failed to get ConfigMap informer: %w", err)
	}
	configMapHandler := toolscache.ResourceEventHandlerFuncs{
		AddFunc:    func(_ any) { c.notifyUpdate() },
		UpdateFunc: func(_, _ any) { c.notifyUpdate() },
		DeleteFunc: func(_ any) { c.notifyUpdate() },
	}
	if _, err = configMapInformer.AddEventHandler(configMapHandler); err != nil {
		return fmt.Errorf("failed to add ConfigMap event handler: %w", err)
	}
	if err := c.addWorkloadEventHandler(ctx, ca, &appsv1.Deployment{}, "deployment"); err != nil {
		return err
	}
	if err := c.addWorkloadEventHandler(ctx, ca, &appsv1.DaemonSet{}, "daemonset"); err != nil {
		return err
	}
	if err := c.addWorkloadEventHandler(ctx, ca, &appsv1.StatefulSet{}, "statefulset"); err != nil {
		return err
	}

	go func() {
		if err := ca.Start(ctx); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			c.log.Error(err, "standalone cache stopped with error")
		}
	}()

	if !ca.WaitForCacheSync(ctx) {
		return errors.New("timed out waiting for ConfigMap cache to sync")
	}

	c.cmCache = ca
	c.log.Info("standalone informer cache synced")
	return nil
}

// addWorkloadEventHandler registers an informer handler for one supported workload kind.
// workload is the Kubernetes object type to watch, and workloadType is the lowercase type used in standalone config.
func (c *Client) addWorkloadEventHandler(ctx context.Context, ca cache.Cache, workload client.Object, workloadType string) error {
	informer, err := ca.GetInformer(ctx, workload)
	if err != nil {
		return fmt.Errorf("failed to get %s informer: %w", workloadType, err)
	}

	handler := toolscache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { c.notifyWorkloadHealthUpdate(workloadType, obj) },
		UpdateFunc: func(_, newObj any) { c.notifyWorkloadHealthUpdate(workloadType, newObj) },
		DeleteFunc: func(obj any) { c.notifyWorkloadHealthUpdate(workloadType, obj) },
	}
	if _, err = informer.AddEventHandler(handler); err != nil {
		return fmt.Errorf("failed to add %s event handler: %w", workloadType, err)
	}
	return nil
}

func (c *Client) notifyUpdate() {
	if c.onUpdate != nil {
		c.onUpdate()
	}
}

// notifyHealthUpdate invokes the registered OpAMP health update for a configured workload.
func (c *Client) notifyHealthUpdate(namespace, workloadType, workloadName string) {
	key := newWorkloadKey(namespace, workloadType, workloadName)
	c.healthMux.RLock()
	updateHealth := c.healthUpdaters[key]
	c.healthMux.RUnlock()
	if updateHealth == nil {
		return
	}
	if err := updateHealth(); err != nil {
		c.log.Error(err, "failed to update health after workload change", "workloadType", workloadType, "name", workloadName, "namespace", namespace)
	}
}

// RegisterHealthUpdater wires workload informer events to the matching OpAMP agent.
func (c *Client) RegisterHealthUpdater(agent config.StandaloneAgentConfig, updateHealth func() error) {
	c.healthMux.Lock()
	defer c.healthMux.Unlock()
	c.healthUpdaters[newWorkloadKey(agent.Namespace, agent.WorkloadRef.Kind, agent.WorkloadRef.Name)] = updateHealth
}

func newWorkloadKey(namespace, workloadType, workloadName string) workloadKey {
	return workloadKey{
		namespace:    namespace,
		workloadType: strings.ToLower(workloadType),
		workloadName: workloadName,
	}
}

// notifyWorkloadHealthUpdate extracts a workload from an informer event and updates its matching agent health.
func (c *Client) notifyWorkloadHealthUpdate(workloadType string, obj any) {
	workload := eventObject(obj)
	if workload == nil {
		return
	}
	c.notifyHealthUpdate(workload.GetNamespace(), workloadType, workload.GetName())
}

// eventObject normalizes informer event objects, including delete tombstones, into controller-runtime objects.
func eventObject(obj any) client.Object {
	if object, ok := obj.(client.Object); ok {
		return object
	}
	tombstone, ok := obj.(toolscache.DeletedFinalStateUnknown)
	if !ok {
		return nil
	}
	object, _ := tombstone.Obj.(client.Object)
	return object
}

// CheckPermissions verifies the Kubernetes access standalone mode needs before starting agents.
func (c *Client) CheckPermissions(ctx context.Context, agents []config.StandaloneAgentConfig, remoteConfigEnabled bool) error {
	perms, err := ListRequiredPermissions(agents, remoteConfigEnabled)
	if err != nil {
		return err
	}
	if err := bridgemanager.CheckPermissions(ctx, c.k8sClient, perms); err != nil {
		return fmt.Errorf("standalone permission check failed: %w", err)
	}
	return nil
}

// ListRequiredPermissions builds the Kubernetes permissions needed for standalone watches and configured agents.
// remoteConfigEnabled adds update permissions for managed ConfigMaps and workloads because applying config triggers rollouts.
func ListRequiredPermissions(agents []config.StandaloneAgentConfig, remoteConfigEnabled bool) ([]bridgemanager.Permission, error) {
	perms := []bridgemanager.Permission{}
	namespaces := namespacesForAgents(agents)
	for _, rule := range []struct {
		apiGroup string
		resource string
	}{
		{resource: "configmaps"},
		{apiGroup: "apps", resource: "deployments"},
		{apiGroup: "apps", resource: "daemonsets"},
		{apiGroup: "apps", resource: "statefulsets"},
	} {
		for _, namespace := range namespaces {
			for _, verb := range []string{"list", "watch"} {
				perms = append(perms, bridgemanager.Permission{Verb: verb, APIGroup: rule.apiGroup, Resource: rule.resource, Namespace: namespace})
			}
		}
	}

	for _, agent := range agents {
		workloadResource, err := standaloneWorkloadResource(agent.WorkloadRef.Kind)
		if err != nil {
			return nil, err
		}
		perms = append(perms, bridgemanager.Permission{Verb: "get", APIGroup: "apps", Resource: workloadResource, Namespace: agent.Namespace, Name: agent.WorkloadRef.Name})
		if remoteConfigEnabled {
			perms = append(perms, bridgemanager.Permission{Verb: "patch", APIGroup: "apps", Resource: workloadResource, Namespace: agent.Namespace, Name: agent.WorkloadRef.Name})
		}
		for _, entry := range agent.Config {
			if entry.Kind != config.StandaloneConfigEntryKindConfigMap {
				continue
			}
			perms = append(perms, bridgemanager.Permission{Verb: "get", Resource: "configmaps", Namespace: agent.Namespace, Name: entry.Name})
			if remoteConfigEnabled {
				perms = append(perms, bridgemanager.Permission{Verb: "update", Resource: "configmaps", Namespace: agent.Namespace, Name: entry.Name})
			}
		}
	}

	return perms, nil
}

func namespacesForAgents(agents []config.StandaloneAgentConfig) []string {
	namespaces := []string{}
	for _, agent := range agents {
		namespace := strings.TrimSpace(agent.Namespace)
		if namespace == "" || slices.Contains(namespaces, namespace) {
			continue
		}
		namespaces = append(namespaces, namespace)
	}
	slices.Sort(namespaces)
	return namespaces
}

func namespaceCacheConfig(namespaces []string) map[string]cache.Config {
	configs := make(map[string]cache.Config, len(namespaces))
	for _, namespace := range namespaces {
		configs[namespace] = cache.Config{}
	}
	return configs
}

func standaloneWorkloadResource(workloadType string) (string, error) {
	switch strings.ToLower(workloadType) {
	case "deployment":
		return "deployments", nil
	case "daemonset":
		return "daemonsets", nil
	case "statefulset":
		return "statefulsets", nil
	default:
		return "", fmt.Errorf("unsupported workload type %q", workloadType)
	}
}

// getConfigMapFile reads the specified config file from a k8s configmap and returns it as raw bytes.
func (c *Client) getConfigMapFile(namespace string, entry config.StandaloneConfigEntry) ([]byte, error) {
	cm := &v1.ConfigMap{}
	if err := c.k8sClient.Get(context.Background(), client.ObjectKey{Name: entry.Name, Namespace: namespace}, cm); err != nil {
		return nil, fmt.Errorf("failed to get ConfigMap %s/%s: %w", namespace, entry.Name, err)
	}
	body, ok := cm.Data[entry.Key]
	if !ok {
		return nil, fmt.Errorf("ConfigMap %s/%s does not contain key %q", namespace, entry.Name, entry.Key)
	}
	return []byte(body), nil
}

// applyConfigMapFile is called when an opamp server pushes config for an agent. It validates the config, updates the local
// k8s configmap and triggers a rolling restart of the relevant workload.
func (c *Client) applyConfigMapFile(namespace, workloadType, workloadName string, entry config.StandaloneConfigEntry, configFile *protobufs.AgentConfigFile) error {
	if len(configFile.Body) == 0 {
		return errors.New("invalid config to apply: config is empty")
	}
	if err := validateCollectorConfigEntry(string(configFile.Body)); err != nil {
		return fmt.Errorf("invalid collector config: %w", err)
	}

	existing := &v1.ConfigMap{}
	err := c.k8sClient.Get(context.Background(), client.ObjectKey{Name: entry.Name, Namespace: namespace}, existing)
	if apierrors.IsNotFound(err) {
		return fmt.Errorf("standalone mode does not support creating ConfigMap %s/%s", namespace, entry.Name)
	} else if err != nil {
		return fmt.Errorf("failed to get ConfigMap %s/%s: %w", namespace, entry.Name, err)
	}

	if existing.Data == nil {
		existing.Data = map[string]string{}
	}
	existing.Data[entry.Key] = string(configFile.Body)
	if updateErr := c.k8sClient.Update(context.Background(), existing); updateErr != nil {
		return fmt.Errorf("failed to update ConfigMap %s/%s: %w", namespace, entry.Name, updateErr)
	}
	c.log.Info("Updated ConfigMap key", "name", entry.Name, "namespace", namespace, "key", entry.Key)

	if err := rollout.TriggerRollout(context.Background(), c.k8sClient, namespace, workloadType, workloadName); err != nil {
		return fmt.Errorf("failed to trigger rollout for %s/%s: %w", namespace, workloadName, err)
	}
	c.log.Info("Triggered workload rollout", "workloadType", workloadType, "name", workloadName, "namespace", namespace)
	return nil
}

type workloadReplicaStatus struct {
	ready   int32
	desired int32
}

func (s workloadReplicaStatus) Healthy() bool {
	return s.desired > 0 && s.ready == s.desired
}

func (s workloadReplicaStatus) String() string {
	return strconv.Itoa(int(s.ready)) + "/" + strconv.Itoa(int(s.desired))
}

func desiredReplicas(replicas *int32) int32 {
	if replicas == nil {
		return 1
	}
	return *replicas
}

// getWorkloadStatusReplicas reads ready and desired replica counts for the configured standalone workload.
func (c *Client) getWorkloadStatusReplicas(ctx context.Context, namespace, workloadType, workloadName string) (workloadReplicaStatus, error) {
	switch strings.ToLower(workloadType) {
	case "deployment":
		deploy := &appsv1.Deployment{}
		if err := c.k8sClient.Get(ctx, client.ObjectKey{Name: workloadName, Namespace: namespace}, deploy); err != nil {
			return workloadReplicaStatus{}, fmt.Errorf("failed to get Deployment %s/%s status.replicas: %w", namespace, workloadName, err)
		}
		return workloadReplicaStatus{ready: deploy.Status.ReadyReplicas, desired: desiredReplicas(deploy.Spec.Replicas)}, nil
	case "daemonset":
		ds := &appsv1.DaemonSet{}
		if err := c.k8sClient.Get(ctx, client.ObjectKey{Name: workloadName, Namespace: namespace}, ds); err != nil {
			return workloadReplicaStatus{}, fmt.Errorf("failed to get DaemonSet %s/%s status.replicas: %w", namespace, workloadName, err)
		}
		return workloadReplicaStatus{ready: ds.Status.NumberReady, desired: ds.Status.DesiredNumberScheduled}, nil
	case "statefulset":
		sts := &appsv1.StatefulSet{}
		if err := c.k8sClient.Get(ctx, client.ObjectKey{Name: workloadName, Namespace: namespace}, sts); err != nil {
			return workloadReplicaStatus{}, fmt.Errorf("failed to get StatefulSet %s/%s status.replicas: %w", namespace, workloadName, err)
		}
		return workloadReplicaStatus{ready: sts.Status.ReadyReplicas, desired: desiredReplicas(sts.Spec.Replicas)}, nil
	default:
		return workloadReplicaStatus{}, fmt.Errorf("unsupported workload type %q", workloadType)
	}
}

// ScopedApplier returns a ConfigApplier limited to one configured standalone agent.
func (c *Client) ScopedApplier(agent config.StandaloneAgentConfig) operator.ConfigApplier {
	return &scopedApplier{
		client: c,
		agent:  agent,
	}
}

type scopedApplier struct {
	client *Client
	agent  config.StandaloneAgentConfig
}

var _ operator.ConfigApplier = &scopedApplier{}

// Apply writes the named remote config entry for this standalone agent.
// name must match one entry from the agent's standalone config map.
func (s *scopedApplier) Apply(name string, configFile *protobufs.AgentConfigFile) error {
	entry, ok := s.agent.Config[name]
	if !ok {
		return fmt.Errorf("standalone agent %q does not manage config %q", s.agent.WorkloadRef.Name, name)
	}
	if entry.Kind != config.StandaloneConfigEntryKindConfigMap {
		return fmt.Errorf("unsupported standalone config kind %q", entry.Kind)
	}
	return s.client.applyConfigMapFile(s.agent.Namespace, s.agent.WorkloadRef.Kind, s.agent.WorkloadRef.Name, entry, configFile)
}

func (*scopedApplier) Delete(name string) error {
	return fmt.Errorf("standalone mode does not support deleting ConfigMap %s", name)
}

// Restart triggers a rolling restart of the managed workload by patching the pod template
// restart annotation, identical to `kubectl rollout restart`.
func (s *scopedApplier) Restart(ctx context.Context) error {
	if err := rollout.TriggerRollout(ctx, s.client.k8sClient, s.agent.Namespace, s.agent.WorkloadRef.Kind, s.agent.WorkloadRef.Name); err != nil {
		return err
	}
	s.client.log.Info("Triggered workload rollout restart",
		"workloadType", s.agent.WorkloadRef.Kind,
		"name", s.agent.WorkloadRef.Name,
		"namespace", s.agent.Namespace)
	return nil
}

// ListInstances reports this standalone workload's current ConfigMap data as effective OpAMP config.
func (s *scopedApplier) ListInstances() ([]operator.CollectorInstance, error) {
	configMap := make(map[string]operator.ConfigFile, len(s.agent.Config))
	for remoteName, entry := range s.agent.Config {
		if entry.Kind != config.StandaloneConfigEntryKindConfigMap {
			continue
		}
		body, err := s.client.getConfigMapFile(s.agent.Namespace, entry)
		if err != nil {
			return nil, err
		}
		configMap[remoteName] = operator.ConfigFile{
			Body:        body,
			ContentType: "yaml",
		}
	}
	if len(configMap) == 0 {
		return []operator.CollectorInstance{}, nil
	}
	return []operator.CollectorInstance{
		newStandaloneCollectorInstance(s.agent.WorkloadRef.Name, s.agent.Namespace, configMap),
	}, nil
}

// GetHealth reports the standalone workload's replica readiness as OpAMP component health.
func (s *scopedApplier) GetHealth() (operator.Health, error) {
	statusReplicas, err := s.client.getWorkloadStatusReplicas(context.Background(), s.agent.Namespace, s.agent.WorkloadRef.Kind, s.agent.WorkloadRef.Name)
	if err != nil {
		return operator.Health{}, err
	}
	return operator.Health{
		Healthy:  statusReplicas.Healthy(),
		Status:   statusReplicas.String(),
		Children: map[string]operator.Health{},
	}, nil
}
