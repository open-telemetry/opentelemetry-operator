// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/open-telemetry/opamp-go/protobufs"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/cmd/operator-opamp-bridge/internal/operator"
)

const (
	standaloneConfigMapKind = "configmap"
)

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

// Start creates an informer cache for ConfigMaps so configured agents can
// refresh their effective config when local data changes.
func (c *Client) Start(ctx context.Context) error {
	ca, err := cache.New(c.restCfg, cache.Options{
		ByObject: map[client.Object]cache.ByObject{
			&v1.ConfigMap{}: {},
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
		AddFunc:    func(_ any) { c.notifyUpdate() },
		UpdateFunc: func(_, _ any) { c.notifyUpdate() },
		DeleteFunc: func(_ any) { c.notifyUpdate() },
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

func (c *Client) notifyUpdate() {
	if c.onUpdate != nil {
		c.onUpdate()
	}
}

func (c *Client) getConfigMapFile(entry config.StandaloneConfigEntry) ([]byte, error) {
	cm := &v1.ConfigMap{}
	if err := c.k8sClient.Get(context.Background(), client.ObjectKey{Name: entry.Name, Namespace: entry.Namespace}, cm); err != nil {
		return nil, fmt.Errorf("failed to get ConfigMap %s/%s: %w", entry.Namespace, entry.Name, err)
	}
	body, ok := cm.Data[entry.Key]
	if !ok {
		return nil, fmt.Errorf("ConfigMap %s/%s does not contain key %q", entry.Namespace, entry.Name, entry.Key)
	}
	return []byte(body), nil
}

func (c *Client) getConfigMapCreationTimestamp(entry config.StandaloneConfigEntry) (v1.ConfigMap, error) {
	cm := &v1.ConfigMap{}
	if err := c.k8sClient.Get(context.Background(), client.ObjectKey{Name: entry.Name, Namespace: entry.Namespace}, cm); err != nil {
		return v1.ConfigMap{}, fmt.Errorf("failed to get ConfigMap %s/%s: %w", entry.Namespace, entry.Name, err)
	}
	return *cm, nil
}

func (c *Client) applyConfigMapFile(entry config.StandaloneConfigEntry, configFile *protobufs.AgentConfigFile) error {
	if len(configFile.Body) == 0 {
		return errors.New("invalid config to apply: config is empty")
	}
	if err := validateCollectorConfigEntry(string(configFile.Body)); err != nil {
		return fmt.Errorf("invalid collector config: %w", err)
	}

	existing := &v1.ConfigMap{}
	err := c.k8sClient.Get(context.Background(), client.ObjectKey{Name: entry.Name, Namespace: entry.Namespace}, existing)
	if apierrors.IsNotFound(err) {
		return fmt.Errorf("standalone mode does not support creating ConfigMap %s/%s", entry.Namespace, entry.Name)
	} else if err != nil {
		return fmt.Errorf("failed to get ConfigMap %s/%s: %w", entry.Namespace, entry.Name, err)
	}

	if existing.Data == nil {
		existing.Data = map[string]string{}
	}
	existing.Data[entry.Key] = string(configFile.Body)
	if updateErr := c.k8sClient.Update(context.Background(), existing); updateErr != nil {
		return fmt.Errorf("failed to update ConfigMap %s/%s: %w", entry.Namespace, entry.Name, updateErr)
	}
	c.log.Info("Updated ConfigMap key", "name", entry.Name, "namespace", entry.Namespace, "key", entry.Key)
	return nil
}

func (c *Client) scopedApplier(agent config.StandaloneAgentConfig) operator.ConfigApplier {
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

func (s *scopedApplier) Apply(name, _ string, configFile *protobufs.AgentConfigFile) error {
	entry, ok := s.agent.Config[name]
	if !ok {
		return fmt.Errorf("standalone agent %q does not manage config %q", s.agent.Name, name)
	}
	if strings.ToLower(entry.Kind) != standaloneConfigMapKind {
		return fmt.Errorf("unsupported standalone config kind %q", entry.Kind)
	}
	return s.client.applyConfigMapFile(entry, configFile)
}

func (*scopedApplier) Delete(name, namespace string) error {
	return fmt.Errorf("standalone mode does not support deleting config %s/%s", namespace, name)
}

func (s *scopedApplier) ListInstances() ([]operator.CollectorInstance, error) {
	result := make([]operator.CollectorInstance, 0, len(s.agent.Config))
	for remoteName, entry := range s.agent.Config {
		if strings.ToLower(entry.Kind) != standaloneConfigMapKind {
			continue
		}
		body, err := s.client.getConfigMapFile(entry)
		if err != nil {
			return nil, err
		}
		cm, err := s.client.getConfigMapCreationTimestamp(entry)
		if err != nil {
			return nil, err
		}
		result = append(result, newStandaloneCollectorInstance(remoteName, "", cm.GetCreationTimestamp().Time, body))
	}
	return result, nil
}

func (*scopedApplier) GetCollectorPods(_ map[string]string, _ string) (*v1.PodList, error) {
	return &v1.PodList{}, nil
}
