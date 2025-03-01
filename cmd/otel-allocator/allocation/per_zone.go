// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"context"
	"fmt"
	"time"

	"github.com/buraksezer/consistent"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/open-telemetry/opentelemetry-operator/cmd/otel-allocator/target"
)

const perZoneStrategyName = "per-zone"
const cacheTTL = 120 * time.Minute

type zonedNode struct {
	nodeName string
	zone     string
}

type collectorZonedNode struct {
	zonedNode
	collector string
}

type targetZonedNode struct {
	zonedNode
	target string
}

func collectorZonedNodeKeyFunc(object interface{}) (string, error) {
	return object.(collectorZonedNode).collector, nil
}

func targetZonedNodeKeyFunc(object interface{}) (string, error) {
	return object.(targetZonedNode).target, nil
}

var _ Strategy = &perZoneStrategy{}

type perZoneStrategy struct {
	kubeClient             kubernetes.Interface
	config                 consistent.Config
	collectorToZonedNode   cache.Store
	targetToZonedNode      cache.Store
	consistentHasherByZone map[string]*consistent.Consistent
}

func newPerZoneStrategy() Strategy {
	config := consistent.Config{
		PartitionCount:    1061,
		ReplicationFactor: 5,
		Load:              1.1,
		Hasher:            hasher{},
	}
	perZoneStrategy := &perZoneStrategy{
		config:                 config,
		consistentHasherByZone: make(map[string]*consistent.Consistent),
		collectorToZonedNode:   cache.NewTTLStore(collectorZonedNodeKeyFunc, cacheTTL),
		targetToZonedNode:      cache.NewTTLStore(targetZonedNodeKeyFunc, cacheTTL),
	}
	return perZoneStrategy
}

func (s *perZoneStrategy) GetCollectorForTarget(collectors map[string]*Collector, item *target.Item) (*Collector, error) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	targetUrl := item.TargetURL
	targetNodeName := item.GetNodeName()
	node, exist, err := s.targetToZonedNode.GetByKey(targetUrl)
	if err != nil {
		fmt.Printf("failed to retrieve target zoned node from cache: %s", err)
	}
	if !exist || node.(targetZonedNode).zonedNode.nodeName != item.GetNodeName() {
		k8sNode, err := s.retrieveK8sNode(ctx, targetNodeName)
		if err != nil {
			return nil, fmt.Errorf("err retrieving k8s node %q for target %q: %w\n", item.GetNodeName(), targetUrl, err)
		}
		targetNodeZone, azLabelExist := k8sNode.ObjectMeta.Labels[v1.LabelTopologyZone]
		if !azLabelExist {
			return nil, fmt.Errorf("succeeded to find the target node %s in the cluster but it doesn't support zone awareness", targetNodeName)
		}
		node = targetZonedNode{
			zonedNode: zonedNode{
				nodeName: targetNodeName,
				zone:     targetNodeZone,
			},
			target: targetUrl,
		}
		err = s.targetToZonedNode.Add(node)
		if err != nil {
			fmt.Printf("error adding a cache for the node and zone information that relates to target %q: %s", targetUrl, err)
		}
	}

	zonedConsistentHasher, exist := s.consistentHasherByZone[node.(targetZonedNode).zonedNode.zone]
	if !exist {
		return nil, fmt.Errorf("unknown zone %q", node.(targetZonedNode).zonedNode.zone)
	}
	member := zonedConsistentHasher.LocateKey([]byte(targetUrl))
	collectorName := member.String()
	collector, exist := collectors[collectorName]
	if !exist {
		return nil, fmt.Errorf("unknown collector %q", collectorName)
	}
	return collector, nil
}

func (s *perZoneStrategy) SetCollectors(collectors map[string]*Collector) {
	clear(s.consistentHasherByZone)

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	collectorsByZone := make(map[string][]string)

	for _, collector := range collectors {
		collectorNodeName := collector.NodeName
		collectorName := collector.Name
		node, exist, err := s.collectorToZonedNode.GetByKey(collectorName)
		if err != nil {
			fmt.Printf("failed to retrieve collector zoned node from cache: %s", err)
		}
		if !exist || node.(collectorZonedNode).zonedNode.nodeName != collector.NodeName {
			k8sNode, err := s.retrieveK8sNode(ctx, collectorNodeName)
			if err != nil {
				fmt.Printf("error retrieving k8s node %q for collector %q: %s\n", collectorNodeName, collectorName, err)
				continue
			}
			collectorNodeZone, exist := k8sNode.ObjectMeta.Labels[v1.LabelTopologyZone]
			if !exist {
				fmt.Printf("succeeded to find the collector node %q for collector %q in the cluster but it doesn't support zone awareness\n", collectorNodeName, collectorName)
				continue
			}
			node = collectorZonedNode{
				zonedNode: zonedNode{
					nodeName: collectorNodeName,
					zone:     collectorNodeZone,
				},
				collector: collectorName,
			}
			err = s.collectorToZonedNode.Add(node)
			if err != nil {
				fmt.Printf("error adding a cache for the node and zone information that relates to collector %q: %s \n", collectorName, err)
			}
		}
		collectorsByZone[node.(collectorZonedNode).zonedNode.zone] = append(collectorsByZone[node.(collectorZonedNode).zonedNode.zone], collectorName)
	}

	var members []consistent.Member
	for zone, collectorNames := range collectorsByZone {
		members = make([]consistent.Member, 0, len(collectors))
		for _, collectorName := range collectorNames {
			members = append(members, collectors[collectorName])
		}
		s.consistentHasherByZone[zone] = consistent.New(members, s.config)
	}
}

func (s *perZoneStrategy) GetName() string {
	return perZoneStrategyName
}

func (s *perZoneStrategy) SetFallbackStrategy(fallbackStrategy Strategy) {}

func (s *perZoneStrategy) SetKubeClient(kubeClient kubernetes.Interface) {
	s.kubeClient = kubeClient
}

func (s *perZoneStrategy) retrieveK8sNode(ctx context.Context, nodeName string) (*v1.Node, error) {
	var node *v1.Node
	var err error
	if node, err = s.kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("could not find the node %q in the cluster\n", nodeName)
		}
		return nil, fmt.Errorf("error when finding the node %q in the cluster, see error %w\n", nodeName, err)
	}
	return node, nil
}
