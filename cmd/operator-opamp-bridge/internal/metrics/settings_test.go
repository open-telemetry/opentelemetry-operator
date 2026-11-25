// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"crypto/tls"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExporterOptions_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		dest    *protobufs.TelemetryConnectionSettings
		wantErr string
	}{
		{
			name:    "nil settings",
			dest:    nil,
			wantErr: "must not be nil",
		},
		{
			name:    "empty endpoint",
			dest:    &protobufs.TelemetryConnectionSettings{},
			wantErr: "must specify DestinationEndpoint",
		},
		{
			name:    "endpoint without host",
			dest:    &protobufs.TelemetryConnectionSettings{DestinationEndpoint: "localhost:4318"},
			wantErr: "must be an absolute URL",
		},
		{
			name:    "unsupported scheme",
			dest:    &protobufs.TelemetryConnectionSettings{DestinationEndpoint: "grpc://localhost:4317"},
			wantErr: "scheme must be http or https",
		},
		{
			name: "invalid proxy url",
			dest: &protobufs.TelemetryConnectionSettings{
				DestinationEndpoint: "http://localhost:4318/v1/metrics",
				Proxy:               &protobufs.ProxyConnectionSettings{Url: "http://[::1"},
			},
			wantErr: "invalid proxy URL",
		},
		{
			name: "invalid min tls version",
			dest: &protobufs.TelemetryConnectionSettings{
				DestinationEndpoint: "https://localhost:4318/v1/metrics",
				Tls:                 &protobufs.TLSConnectionSettings{MinVersion: "1.4"},
			},
			wantErr: "invalid tls.min_version",
		},
		{
			name: "max tls version below min",
			dest: &protobufs.TelemetryConnectionSettings{
				DestinationEndpoint: "https://localhost:4318/v1/metrics",
				Tls:                 &protobufs.TLSConnectionSettings{MinVersion: "1.3", MaxVersion: "1.2"},
			},
			wantErr: "lower than the minimum",
		},
		{
			name: "unknown cipher suite",
			dest: &protobufs.TelemetryConnectionSettings{
				DestinationEndpoint: "https://localhost:4318/v1/metrics",
				Tls:                 &protobufs.TLSConnectionSettings{CipherSuites: []string{"NOT_A_CIPHER"}},
			},
			wantErr: "unsupported or insecure TLS cipher suite",
		},
		{
			name: "invalid ca pem",
			dest: &protobufs.TelemetryConnectionSettings{
				DestinationEndpoint: "https://localhost:4318/v1/metrics",
				Tls:                 &protobufs.TLSConnectionSettings{CaPemContents: "not a pem"},
			},
			wantErr: "tls.ca_pem_contents",
		},
		{
			name: "invalid certificate ca",
			dest: &protobufs.TelemetryConnectionSettings{
				DestinationEndpoint: "https://localhost:4318/v1/metrics",
				Certificate:         &protobufs.TLSCertificate{CaCert: []byte("not a pem")},
			},
			wantErr: "certificate.ca_cert",
		},
		{
			name: "invalid client certificate",
			dest: &protobufs.TelemetryConnectionSettings{
				DestinationEndpoint: "https://localhost:4318/v1/metrics",
				Certificate:         &protobufs.TLSCertificate{Cert: []byte("cert"), PrivateKey: []byte("key")},
			},
			wantErr: "unable to load client certificate",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := exporterOptions(tt.dest)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestTLSConfig(t *testing.T) {
	srv := httptest.NewTLSServer(http.NotFoundHandler())
	defer srv.Close()
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})

	t.Run("defaults", func(t *testing.T) {
		cfg, err := tlsConfig(&protobufs.TelemetryConnectionSettings{DestinationEndpoint: "https://localhost"})
		require.NoError(t, err)
		assert.Equal(t, uint16(tls.VersionTLS12), cfg.MinVersion)
		assert.Zero(t, cfg.MaxVersion)
		assert.Nil(t, cfg.RootCAs, "the host's root CAs should be used by default")
		assert.False(t, cfg.InsecureSkipVerify)
	})

	t.Run("all tls settings", func(t *testing.T) {
		cfg, err := tlsConfig(&protobufs.TelemetryConnectionSettings{
			DestinationEndpoint: "https://localhost",
			Tls: &protobufs.TLSConnectionSettings{
				CaPemContents:      string(caPEM),
				InsecureSkipVerify: true,
				MinVersion:         "1.2",
				MaxVersion:         "TLSv1.3",
				CipherSuites:       []string{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, uint16(tls.VersionTLS12), cfg.MinVersion)
		assert.Equal(t, uint16(tls.VersionTLS13), cfg.MaxVersion)
		assert.Equal(t, []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256}, cfg.CipherSuites)
		assert.True(t, cfg.InsecureSkipVerify)
		assert.NotNil(t, cfg.RootCAs)
	})

	t.Run("certificate ca", func(t *testing.T) {
		cfg, err := tlsConfig(&protobufs.TelemetryConnectionSettings{
			DestinationEndpoint: "https://localhost",
			Certificate:         &protobufs.TLSCertificate{CaCert: caPEM},
		})
		require.NoError(t, err)
		assert.NotNil(t, cfg.RootCAs)
		assert.Empty(t, cfg.Certificates)
	})
}

// otlpReceiver is a minimal OTLP/HTTP endpoint that records the requests it receives.
type otlpReceiver struct {
	mu       sync.Mutex
	requests []*http.Request
}

func (r *otlpReceiver) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	_, _ = io.Copy(io.Discard, req.Body)
	r.mu.Lock()
	r.requests = append(r.requests, req)
	r.mu.Unlock()
	w.Header().Set("Content-Type", "application/x-protobuf")
	w.WriteHeader(http.StatusOK)
}

func (r *otlpReceiver) received() []*http.Request {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*http.Request(nil), r.requests...)
}

// exportOnce creates a reporter for dest, records a metric, and shuts the reporter down, which flushes the metric to
// the destination.
func exportOnce(t *testing.T, dest *protobufs.TelemetryConnectionSettings) {
	t.Helper()
	reporter, err := NewMetricReporter(logr.Discard(), dest, "test-agent", "v1", uuid.New())
	require.NoError(t, err)
	reporter.RecordHeartbeat(t.Context(), nil)
	reporter.Shutdown()
}

func TestNewMetricReporter_ExportsOverHTTP(t *testing.T) {
	receiver := &otlpReceiver{}
	srv := httptest.NewServer(receiver)
	defer srv.Close()

	exportOnce(t, &protobufs.TelemetryConnectionSettings{
		DestinationEndpoint: srv.URL + "/custom/metrics",
		Headers: &protobufs.Headers{Headers: []*protobufs.Header{
			{Key: "Authorization", Value: "Bearer secret"},
		}},
	})

	require.Eventually(t, func() bool { return len(receiver.received()) > 0 }, 5*time.Second, 10*time.Millisecond)
	req := receiver.received()[0]
	assert.Equal(t, http.MethodPost, req.Method)
	assert.Equal(t, "/custom/metrics", req.URL.Path)
	assert.Equal(t, "Bearer secret", req.Header.Get("Authorization"))
}

func TestNewMetricReporter_DefaultsMetricsPath(t *testing.T) {
	receiver := &otlpReceiver{}
	srv := httptest.NewServer(receiver)
	defer srv.Close()

	exportOnce(t, &protobufs.TelemetryConnectionSettings{DestinationEndpoint: srv.URL})

	require.Eventually(t, func() bool { return len(receiver.received()) > 0 }, 5*time.Second, 10*time.Millisecond)
	assert.Equal(t, defaultMetricsPath, receiver.received()[0].URL.Path)
}

func TestNewMetricReporter_ExportsOverHTTPSWithOfferedCA(t *testing.T) {
	receiver := &otlpReceiver{}
	srv := httptest.NewTLSServer(receiver)
	defer srv.Close()
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})

	exportOnce(t, &protobufs.TelemetryConnectionSettings{
		DestinationEndpoint: srv.URL + "/v1/metrics",
		Tls:                 &protobufs.TLSConnectionSettings{CaPemContents: string(caPEM)},
	})

	require.Eventually(t, func() bool { return len(receiver.received()) > 0 }, 5*time.Second, 10*time.Millisecond)
	assert.Equal(t, "/v1/metrics", receiver.received()[0].URL.Path)
}

func TestNewMetricReporter_ExportsThroughProxy(t *testing.T) {
	// For a plain-HTTP destination, a forward proxy receives the request with the absolute destination URL.
	proxyReceiver := &otlpReceiver{}
	proxySrv := httptest.NewServer(proxyReceiver)
	defer proxySrv.Close()

	exportOnce(t, &protobufs.TelemetryConnectionSettings{
		DestinationEndpoint: "http://otlp.example.invalid:4318/v1/metrics",
		Proxy:               &protobufs.ProxyConnectionSettings{Url: proxySrv.URL},
	})

	require.Eventually(t, func() bool { return len(proxyReceiver.received()) > 0 }, 5*time.Second, 10*time.Millisecond)
	req := proxyReceiver.received()[0]
	assert.Equal(t, "otlp.example.invalid:4318", req.Host)
	assert.Equal(t, "/v1/metrics", req.URL.Path)
}
