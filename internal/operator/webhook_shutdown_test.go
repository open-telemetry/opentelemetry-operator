// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDelayWebhookShutdown(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	delayed, stop := delayWebhookShutdown(parent, 100*time.Millisecond)
	t.Cleanup(stop)
	cancel()
	select {
	case <-delayed.Done():
		t.Fatal("webhook shutdown was not delayed")
	case <-time.After(20 * time.Millisecond):
	}
	select {
	case <-delayed.Done():
	case <-time.After(time.Second):
		t.Fatal("webhook shutdown was not completed")
	}
}

func TestDelayWebhookShutdownCanBeCanceled(t *testing.T) {
	delayed, cancel := delayWebhookShutdown(context.Background(), time.Second)
	cancel()
	require.ErrorIs(t, delayed.Err(), context.Canceled)
}
