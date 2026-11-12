// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/e2e-framework/klient"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func proxyTestConfig(t *testing.T, handler http.HandlerFunc) *envconf.Config {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := klient.New(&rest.Config{
		Host:          server.URL,
		ContentConfig: rest.ContentConfig{ContentType: "application/json", AcceptContentTypes: "application/json"},
	})
	require.NoError(t, err)
	return envconf.New().WithClient(client)
}

func TestPodHTTPGet(t *testing.T) {
	for _, tt := range []struct {
		name       string
		pods       []corev1.Pod
		listStatus int
		getStatus  int
		wantError  string
	}{
		{
			name: "selects the first running pod that is not terminating",
			pods: []corev1.Pod{
				{ObjectMeta: metav1.ObjectMeta{Name: "pending"}, Status: corev1.PodStatus{Phase: corev1.PodPending}},
				{ObjectMeta: metav1.ObjectMeta{Name: "terminating", DeletionTimestamp: new(metav1.Now())}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},
				{ObjectMeta: metav1.ObjectMeta{Name: "completed"}, Status: corev1.PodStatus{Phase: corev1.PodSucceeded}},
				{ObjectMeta: metav1.ObjectMeta{Name: "selected"}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},
				{ObjectMeta: metav1.ObjectMeta{Name: "other"}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},
			},
		},
		{name: "no pods", wantError: `no running pod matches "app=sample" in test-ns (0 pods)`},
		{
			name:      "no running pods",
			pods:      []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "pending"}, Status: corev1.PodStatus{Phase: corev1.PodPending}}},
			wantError: `no running pod matches "app=sample" in test-ns (1 pods)`,
		},
		{name: "list fails", listStatus: http.StatusForbidden, wantError: `list pods "app=sample" in test-ns`},
		{
			name:      "proxy fails",
			pods:      []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "selected"}, Status: corev1.PodStatus{Phase: corev1.PodRunning}}},
			getStatus: http.StatusBadGateway,
			wantError: `GET pod selected port 8080 path "/rolldice"`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := proxyTestConfig(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				switch r.URL.Path {
				case "/api/v1/namespaces/test-ns/pods":
					assert.Equal(t, "app=sample", r.URL.Query().Get("labelSelector"))
					if tt.listStatus != 0 {
						http.Error(w, "cannot list pods", tt.listStatus)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					assert.NoError(t, json.NewEncoder(w).Encode(corev1.PodList{
						TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "PodList"}, Items: tt.pods,
					}))
				case "/api/v1/namespaces/test-ns/pods/http:selected:8080/proxy/rolldice":
					if tt.getStatus != 0 {
						http.Error(w, "app unavailable", tt.getStatus)
						return
					}
					_, err := w.Write([]byte("six"))
					assert.NoError(t, err)
				default:
					t.Errorf("unexpected request: %s", r.URL)
					http.NotFound(w, r)
				}
			})
			body, err := PodHTTPGet(t.Context(), cfg, "test-ns", "app=sample", 8080, "/rolldice")
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				assert.Empty(t, body)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "six", string(body))
			}
		})
	}
}

func TestServiceHTTPGet(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusBadGateway} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			cfg := proxyTestConfig(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/api/v1/namespaces/test-ns/services/http:otlp-sink:4319/proxy/received/traces", r.URL.Path)
				w.WriteHeader(status)
				_, err := w.Write([]byte(`{"requests":[]}`))
				assert.NoError(t, err)
			})
			body, err := ServiceHTTPGet(t.Context(), cfg, "test-ns", "otlp-sink", 4319, "/received/traces")
			if status == http.StatusOK {
				require.NoError(t, err)
				assert.JSONEq(t, `{"requests":[]}`, string(body))
			} else {
				require.ErrorContains(t, err, `GET service otlp-sink port 4319 path "/received/traces"`)
				assert.Empty(t, body)
			}
		})
	}
}

func TestHTTPGetCancellation(t *testing.T) {
	cfg := proxyTestConfig(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := PodHTTPGet(ctx, cfg, "test-ns", "app=sample", 8080, "/")
	require.ErrorIs(t, err, context.Canceled)
	_, err = ServiceHTTPGet(ctx, cfg, "test-ns", "otlp-sink", 4319, "/received/traces")
	require.ErrorIs(t, err, context.Canceled)
}
