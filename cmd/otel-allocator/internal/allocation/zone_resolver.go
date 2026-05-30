// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"context"
	"sync"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	legacyZoneLabel = "failure-domain.beta.kubernetes.io/zone"
)

// NodeZoneResolver resolves Kubernetes node names to topology zones
// by reading zone labels from Node objects.
type NodeZoneResolver struct {
	mu        sync.RWMutex
	nodeZones map[string]string
	k8sClient kubernetes.Interface
	zoneLabel string
	log       logr.Logger
}

func NewNodeZoneResolver(log logr.Logger, client kubernetes.Interface, zoneLabel string) *NodeZoneResolver {
	return &NodeZoneResolver{
		nodeZones: make(map[string]string),
		k8sClient: client,
		zoneLabel: zoneLabel,
		log:       log.WithValues("component", "zone-resolver"),
	}
}

// SyncNodes fetches all nodes and caches their zone labels.
func (r *NodeZoneResolver) SyncNodes(ctx context.Context) error {
	nodes, err := r.k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	newMap := make(map[string]string, len(nodes.Items))
	for i := range nodes.Items {
		node := &nodes.Items[i]
		zone := node.Labels[r.zoneLabel]
		if zone == "" {
			zone = node.Labels[legacyZoneLabel]
		}
		if zone != "" {
			newMap[node.Name] = zone
		}
	}
	r.nodeZones = newMap
	r.log.V(1).Info("Synced node zones", "count", len(newMap))
	return nil
}

// GetZone returns the zone for a node, or "" if unknown.
func (r *NodeZoneResolver) GetZone(nodeName string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.nodeZones[nodeName]
}
