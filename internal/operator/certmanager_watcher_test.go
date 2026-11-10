// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/autodetectutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/autodetect/certmanager"
)

type mockCertManagerAvailabilityChecker struct {
	availability certmanager.Availability
	err          error
}

func (m mockCertManagerAvailabilityChecker) CertManagerAvailability(context.Context) (certmanager.Availability, error) {
	return m.availability, m.err
}

type mockCertManagerRollout struct {
	requests int
	err      error
}

func (m *mockCertManagerRollout) requestRollout(context.Context) error {
	m.requests++
	return m.err
}

func TestCertManagerAvailabilityWatcherCheck(t *testing.T) {
	tests := []struct {
		name                string
		initialAvailability certmanager.Availability
		currentAvailability certmanager.Availability
		checkErr            error
		requests            int
	}{
		{
			name:                "unavailable is unchanged",
			initialAvailability: certmanager.NotAvailable,
			currentAvailability: certmanager.NotAvailable,
		},
		{
			name:                "available is unchanged",
			initialAvailability: certmanager.Available,
			currentAvailability: certmanager.Available,
		},
		{
			name:                "cert-manager becomes available",
			initialAvailability: certmanager.NotAvailable,
			currentAvailability: certmanager.Available,
			requests:            1,
		},
		{
			name:                "cert-manager becomes unavailable",
			initialAvailability: certmanager.Available,
			currentAvailability: certmanager.NotAvailable,
			requests:            1,
		},
		{
			name:                "confirmed missing permissions",
			initialAvailability: certmanager.Available,
			currentAvailability: certmanager.NotAvailable,
			checkErr:            fmt.Errorf("check failed: %w", certmanager.ErrMissingPermissions),
			requests:            1,
		},
		{
			name:                "missing permissions remain unchanged",
			initialAvailability: certmanager.NotAvailable,
			currentAvailability: certmanager.NotAvailable,
			checkErr:            certmanager.ErrMissingPermissions,
		},
		{
			name:                "transient failure does not trigger rollout",
			initialAvailability: certmanager.Available,
			currentAvailability: certmanager.NotAvailable,
			checkErr:            errors.New("discovery failure"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rollout := &mockCertManagerRollout{}
			watcher := certManagerAvailabilityWatcher{
				checker: mockCertManagerAvailabilityChecker{
					availability: test.currentAvailability,
					err:          test.checkErr,
				},
				initialAvailability: test.initialAvailability,
				rollout:             rollout,
			}
			watcher.check(context.Background())
			require.Equal(t, test.requests, rollout.requests)
		})
	}
}

func TestCertManagerAvailabilityWatcherRetriesRollout(t *testing.T) {
	rollout := &mockCertManagerRollout{err: errors.New("temporary error")}
	watcher := certManagerAvailabilityWatcher{
		checker:             mockCertManagerAvailabilityChecker{availability: certmanager.Available},
		initialAvailability: certmanager.NotAvailable,
		rollout:             rollout,
	}
	watcher.check(context.Background())
	rollout.err = nil
	watcher.check(context.Background())
	require.Equal(t, 2, rollout.requests)
}

type blockingCertManagerAvailabilityChecker struct {
	started chan struct{}
}

func (m blockingCertManagerAvailabilityChecker) CertManagerAvailability(ctx context.Context) (certmanager.Availability, error) {
	close(m.started)
	<-ctx.Done()
	return certmanager.NotAvailable, ctx.Err()
}

func TestCertManagerAvailabilityWatcherStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	checker := blockingCertManagerAvailabilityChecker{started: make(chan struct{})}
	watcher := certManagerAvailabilityWatcher{checker: checker, rollout: &mockCertManagerRollout{}}
	done := make(chan struct{})
	go func() {
		defer close(done)
		watcher.run(ctx)
	}()
	<-checker.started
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("watcher did not stop")
	}
}

type countingCertManagerChecker struct {
	checks atomic.Int32
}

func (m *countingCertManagerChecker) CertManagerAvailability(context.Context) (certmanager.Availability, error) {
	m.checks.Add(1)
	return certmanager.NotAvailable, nil
}

func TestCertManagerAvailabilityWatcherPolls(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	checker := &countingCertManagerChecker{}
	watcher := certManagerAvailabilityWatcher{
		checker:  checker,
		rollout:  &mockCertManagerRollout{},
		interval: time.Millisecond,
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		watcher.run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	require.Eventually(t, func() bool { return checker.checks.Load() >= 2 }, time.Second, time.Millisecond)
}

func TestStartCertManagerAvailabilityWatcher(t *testing.T) {
	pod, rs, deployment := certManagerRolloutObjects()
	var err error
	pod.Name, err = os.Hostname()
	require.NoError(t, err)
	t.Setenv(autodetectutils.NAMESPACE_ENV_VAR, pod.Namespace)
	clientset := fake.NewClientset(pod, rs, deployment)
	ctx, cancel := context.WithCancel(context.Background())
	done := StartCertManagerAvailabilityWatcher(ctx, mockCertManagerAvailabilityChecker{availability: certmanager.Available}, certmanager.NotAvailable, clientset)
	t.Cleanup(func() {
		cancel()
		<-done
	})
	require.Eventually(t, func() bool {
		updated, getErr := clientset.AppsV1().Deployments(pod.Namespace).Get(ctx, deployment.Name, metav1.GetOptions{})
		return getErr == nil && updated.Spec.Template.Annotations[certManagerRolloutAnnotation] != ""
	}, time.Second, time.Millisecond)
}
