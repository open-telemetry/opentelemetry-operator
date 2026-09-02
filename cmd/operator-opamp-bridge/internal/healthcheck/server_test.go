// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package healthcheck

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerStartsHealthEndpoint(t *testing.T) {
	server := NewServer(logr.Discard(), "127.0.0.1:0")

	require.NoError(t, server.Start(t.Context()))
	t.Cleanup(func() {
		// t.Context() is already canceled when cleanup runs; use Background.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		require.NoError(t, server.Stop(ctx))
	})

	client := http.Client{Timeout: time.Second}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+server.Addr()+Path, http.NoBody)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestServerReturnsBindError(t *testing.T) {
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	server := NewServer(logr.Discard(), listener.Addr().String())

	err = server.Start(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start health listener")
}
