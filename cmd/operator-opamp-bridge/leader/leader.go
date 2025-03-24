// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package leader

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"go.uber.org/atomic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	rl "k8s.io/client-go/tools/leaderelection/resourcelock"
)

const (
	lockName = "bridge.opentelemetry.io"
)

type Elector struct {
	logger logr.Logger

	leaderElector *leaderelection.LeaderElector
	done          chan struct{}
	isLeader      atomic.Bool
	callback      func(bool)
}

func NewElector(logger logr.Logger, cfg *rest.Config, lockNamespace string, callback func(bool)) (*Elector, error) {
	uid := uuid.New()
	l, err := rl.NewFromKubeconfig(
		rl.LeasesResourceLock,
		lockNamespace,
		lockName,
		rl.ResourceLockConfig{
			Identity: uid.String(),
		},
		cfg,
		time.Second*10,
	)
	if err != nil {
		return nil, err
	}
	elector := &Elector{
		logger:   logger,
		done:     make(chan struct{}, 1),
		callback: callback,
	}
	el, err := leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:          l,
		LeaseDuration: time.Second * 15,
		RenewDeadline: time.Second * 10,
		RetryPeriod:   time.Second * 2,
		Name:          lockName,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: elector.onStartedLeading,
			OnStoppedLeading: elector.onStoppedLeading,
			OnNewLeader:      elector.onNewLeader,
		},
	})
	if err != nil {
		return nil, err
	}
	elector.leaderElector = el
	return elector, nil
}

func (e *Elector) Start(ctx context.Context) {
	ctxWithCancel, cancel := context.WithCancel(ctx)

	go func() {
		e.leaderElector.Run(ctxWithCancel)
	}()

	for {
		<-e.done
		e.logger.Info("stopping leader election")
		cancel()
		return
	}
}

func (e *Elector) Stop() {
	close(e.done)
}

func (e *Elector) onStartedLeading(ctx context.Context) {
	e.logger.Info("elected leader")
	e.isLeader.Store(true)
	e.callback(true)
}

func (e *Elector) onStoppedLeading() {
	e.logger.Info("no longer leader")
	e.isLeader.Store(false)
	e.callback(false)
}

func (e *Elector) onNewLeader(identity string) {
	e.logger.Info("new leader elected", "identity", identity)
}

func (e *Elector) IsLeader() bool {
	return e.isLeader.Load()
}
