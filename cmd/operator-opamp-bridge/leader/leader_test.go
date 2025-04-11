// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package leader

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
)

var (
	l = logr.Discard()
)

func TestElector(t *testing.T) {
	cfg := &rest.Config{}
	lockNamespace := "default"
	curState := false
	c := func(s bool) {
		curState = s
	}
	elector, err := NewElector(l, cfg, lockNamespace, c)
	assert.NoError(t, err)
	ctx := context.Background()

	go elector.Start(ctx)
	time.Sleep(1 * time.Second)

	assert.False(t, elector.IsLeader())
	assert.False(t, curState)
	elector.onStartedLeading(ctx)
	assert.True(t, elector.IsLeader())
	assert.True(t, curState)
	elector.onStoppedLeading()
	assert.False(t, elector.IsLeader())
	assert.False(t, curState)
	elector.Stop()
	time.Sleep(1 * time.Second)
}
