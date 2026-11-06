// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"time"
)

const webhookShutdownDelay = 5 * time.Second

// DelayWebhookShutdown keeps the webhook serving while terminating endpoints are removed.
func DelayWebhookShutdown(ctx context.Context) (context.Context, context.CancelFunc) {
	return delayWebhookShutdown(ctx, webhookShutdownDelay)
}

func delayWebhookShutdown(ctx context.Context, delay time.Duration) (context.Context, context.CancelFunc) {
	delayed, cancel := context.WithCancel(context.WithoutCancel(ctx))
	go func() {
		select {
		case <-ctx.Done():
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-timer.C:
				cancel()
			case <-delayed.Done():
			}
		case <-delayed.Done():
		}
	}()
	return delayed, cancel
}
