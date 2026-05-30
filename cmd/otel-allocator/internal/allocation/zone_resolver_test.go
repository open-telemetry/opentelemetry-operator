// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package allocation

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	standardZoneLabel = "topology.kubernetes.io/zone"
)

func nodeWithZone(name, zoneLabel, zone string) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{zoneLabel: zone},
		},
	}
}

func TestNodeZoneResolver_SyncNodes_PopulatesIndex(t *testing.T) {
	// Happy path: every node carries the configured zone label and the
	// resolver builds a complete node→zone index.
	client := fake.NewSimpleClientset(
		nodeWithZone("node-a-1", standardZoneLabel, "us-east-1a"),
		nodeWithZone("node-a-2", standardZoneLabel, "us-east-1a"),
		nodeWithZone("node-b-1", standardZoneLabel, "us-east-1b"),
	)
	r := NewNodeZoneResolver(logf.Log.WithName("test"), client, standardZoneLabel)
	require.NoError(t, r.SyncNodes(t.Context()))

	assert.Equal(t, "us-east-1a", r.GetZone("node-a-1"))
	assert.Equal(t, "us-east-1a", r.GetZone("node-a-2"))
	assert.Equal(t, "us-east-1b", r.GetZone("node-b-1"))
	// Unknown nodes return the zero value, not a panic. This is what the
	// collector watcher relies on when a pod runs on a node we have no
	// label info for.
	assert.Equal(t, "", r.GetZone("unknown"))
}

func TestNodeZoneResolver_SyncNodes_LegacyLabelFallback(t *testing.T) {
	// Nodes managed by older kubelets (or by certain CAPI providers) only
	// carry the legacy "failure-domain.beta.kubernetes.io/zone" label.
	// The resolver must fall back to that label so existing clusters can
	// opt in to zone-aware allocation without re-labeling every node.
	client := fake.NewSimpleClientset(
		nodeWithZone("node-legacy", legacyZoneLabel, "eu-west-1c"),
		nodeWithZone("node-modern", standardZoneLabel, "eu-west-1a"),
	)
	r := NewNodeZoneResolver(logf.Log.WithName("test"), client, standardZoneLabel)
	require.NoError(t, r.SyncNodes(t.Context()))

	assert.Equal(t, "eu-west-1c", r.GetZone("node-legacy"),
		"the legacy failure-domain label must be honored as a fallback")
	assert.Equal(t, "eu-west-1a", r.GetZone("node-modern"))
}

func TestNodeZoneResolver_SyncNodes_StandardOverridesLegacy(t *testing.T) {
	// Nodes that carry both labels (transitional state during a cluster
	// upgrade) must prefer the standard label so the resolver matches the
	// behavior of upstream Kubernetes schedulers.
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-dual",
			Labels: map[string]string{
				standardZoneLabel: "eu-west-1a",
				legacyZoneLabel:   "eu-west-1c",
			},
		},
	}
	client := fake.NewSimpleClientset(node)
	r := NewNodeZoneResolver(logf.Log.WithName("test"), client, standardZoneLabel)
	require.NoError(t, r.SyncNodes(t.Context()))

	assert.Equal(t, "eu-west-1a", r.GetZone("node-dual"),
		"standard zone label must take precedence over legacy when both are present")
}

func TestNodeZoneResolver_SyncNodes_SkipsNodesWithoutZoneLabel(t *testing.T) {
	// Nodes that carry no zone label at all (e.g. kind clusters, bare-metal
	// installs without topology) must simply be absent from the index.
	// The resolver returns "" for these, which the strategy treats as
	// zone-less.
	noZone := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node-no-zone",
			Labels: map[string]string{"some-other-label": "value"},
		},
	}
	withZone := nodeWithZone("node-with-zone", standardZoneLabel, "us-east-1a")
	client := fake.NewSimpleClientset(noZone, withZone)
	r := NewNodeZoneResolver(logf.Log.WithName("test"), client, standardZoneLabel)
	require.NoError(t, r.SyncNodes(t.Context()))

	assert.Equal(t, "", r.GetZone("node-no-zone"),
		"nodes with no zone label must report empty, not a default zone")
	assert.Equal(t, "us-east-1a", r.GetZone("node-with-zone"))
}

func TestNodeZoneResolver_SyncNodes_RebuildsOnRepeatedCall(t *testing.T) {
	// SyncNodes is called periodically (currently only at startup, but we
	// guard the rebuild contract here so a future watcher implementation
	// can rely on it). After re-sync, nodes that disappeared from the API
	// must be absent from the index.
	client := fake.NewSimpleClientset(
		nodeWithZone("ephemeral", standardZoneLabel, "zone-a"),
		nodeWithZone("durable", standardZoneLabel, "zone-b"),
	)
	r := NewNodeZoneResolver(logf.Log.WithName("test"), client, standardZoneLabel)
	require.NoError(t, r.SyncNodes(t.Context()))
	assert.Equal(t, "zone-a", r.GetZone("ephemeral"))

	// Remove the ephemeral node and re-sync.
	require.NoError(t, client.CoreV1().Nodes().Delete(t.Context(), "ephemeral", metav1.DeleteOptions{}))
	require.NoError(t, r.SyncNodes(t.Context()))

	assert.Equal(t, "", r.GetZone("ephemeral"),
		"a node that disappeared from the API must drop out of the resolver index on re-sync")
	assert.Equal(t, "zone-b", r.GetZone("durable"),
		"surviving nodes must keep their zone after re-sync")
}

func TestNodeZoneResolver_SyncNodes_PropagatesAPIError(t *testing.T) {
	// API errors must propagate so the caller can decide whether to retry
	// or to start in degraded mode (the existing main.go logs and
	// continues, but a future caller might want to fail-fast).
	client := fake.NewSimpleClientset()
	client.PrependReactor("list", "nodes", func(clienttesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, errors.New("simulated API outage")
	})
	r := NewNodeZoneResolver(logf.Log.WithName("test"), client, standardZoneLabel)
	err := r.SyncNodes(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "simulated API outage")
}

func TestNodeZoneResolver_SyncNodes_CustomZoneLabel(t *testing.T) {
	// Operators can override the zone label to support non-standard
	// taxonomies (e.g. "topology.example.com/datacenter" in non-cloud
	// installs). Verify the custom label flows through and the standard
	// label is *not* consulted in that mode.
	const customLabel = "topology.example.com/datacenter"
	mixed := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-custom",
			Labels: map[string]string{
				customLabel:       "dc-east",
				standardZoneLabel: "should-be-ignored",
			},
		},
	}
	client := fake.NewSimpleClientset(mixed)
	r := NewNodeZoneResolver(logf.Log.WithName("test"), client, customLabel)
	require.NoError(t, r.SyncNodes(t.Context()))

	assert.Equal(t, "dc-east", r.GetZone("node-custom"),
		"custom zone label must be respected; standard label must not leak in when a custom label is configured")
}

func TestNodeZoneResolver_GetZone_ConcurrentReads(t *testing.T) {
	// GetZone is called from the collector watcher's event handler, which
	// runs in its own goroutine; SyncNodes might (in future) be called
	// from a parallel re-sync loop. Exercise the lock to surface any
	// regression that drops the read lock.
	client := fake.NewSimpleClientset(nodeWithZone("n", standardZoneLabel, "z"))
	r := NewNodeZoneResolver(logf.Log.WithName("test"), client, standardZoneLabel)
	require.NoError(t, r.SyncNodes(t.Context()))

	done := make(chan struct{})
	for range 16 {
		go func() {
			for range 100 {
				_ = r.GetZone("n")
			}
			done <- struct{}{}
		}()
	}
	// Also drive SyncNodes concurrently with reads.
	for range 4 {
		go func() {
			for range 25 {
				_ = r.SyncNodes(context.Background())
			}
			done <- struct{}{}
		}()
	}
	for range 20 {
		<-done
	}
	assert.Equal(t, "z", r.GetZone("n"))
}
