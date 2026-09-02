// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package healthcheck

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-logr/logr"
)

const Path = "/healthz"

type Server struct {
	logger logr.Logger
	server *http.Server
	addr   string
}

func NewServer(logger logr.Logger, listenAddr string) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc(Path, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return &Server{
		logger: logger,
		server: &http.Server{
			Addr:              listenAddr,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (s *Server) Start(ctx context.Context) error {
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", s.server.Addr)
	if err != nil {
		return fmt.Errorf("failed to start health listener on %q: %w", s.server.Addr, err)
	}

	s.addr = listener.Addr().String()
	go func() {
		s.logger.Info("starting health listener", "address", s.addr)
		if err := s.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error(err, "health listener stopped with error")
		}
	}()
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s == nil || s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

func (s *Server) Addr() string {
	return s.addr
}
