// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/open-telemetry/opamp-go/protobufs"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
)

const (
	defaultMetricsPath = "/v1/metrics"
	exportTimeout      = 10 * time.Second
)

// exporterOptions translates the OpAMP TelemetryConnectionSettings offered by the server into the options for an
// OTLP/HTTP metric exporter. Every field of the message is honoured:
//   - destination_endpoint: full URL of the OTLP/HTTP metrics endpoint. The scheme selects plain HTTP or HTTPS, and
//     /v1/metrics is used when the URL has no path.
//   - headers: added to every export request.
//   - certificate: client certificate (mTLS) and an optional CA used to verify the destination.
//   - tls: CA, system pool, skip-verify, min/max version and cipher suites.
//   - proxy: HTTP proxy URL and CONNECT headers.
func exporterOptions(dest *protobufs.TelemetryConnectionSettings) ([]otlpmetrichttp.Option, error) {
	if dest == nil {
		return nil, errors.New("telemetry connection settings must not be nil")
	}
	if dest.GetDestinationEndpoint() == "" {
		return nil, errors.New("metric destination must specify DestinationEndpoint")
	}

	u, err := url.Parse(dest.GetDestinationEndpoint())
	if err != nil {
		return nil, fmt.Errorf("invalid DestinationEndpoint: %w", err)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("invalid DestinationEndpoint %q: must be an absolute URL including a host", dest.GetDestinationEndpoint())
	}

	path := u.EscapedPath()
	if path == "" || path == "/" {
		path = defaultMetricsPath
	}
	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(u.Host),
		otlpmetrichttp.WithURLPath(path),
	}

	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, errors.New("unexpected type for http.DefaultTransport")
	}
	transport = transport.Clone()

	switch strings.ToLower(u.Scheme) {
	case "http":
		opts = append(opts, otlpmetrichttp.WithInsecure())
	case "https":
		tlsCfg, tlsErr := tlsConfig(dest)
		if tlsErr != nil {
			return nil, tlsErr
		}
		transport.TLSClientConfig = tlsCfg
	default:
		return nil, fmt.Errorf("invalid DestinationEndpoint %q: scheme must be http or https", dest.GetDestinationEndpoint())
	}

	if proxy := dest.GetProxy(); proxy != nil && proxy.GetUrl() != "" {
		proxyURL, proxyErr := url.Parse(proxy.GetUrl())
		if proxyErr != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", proxyErr)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
		if connectHeaders := toHTTPHeader(proxy.GetConnectHeaders()); len(connectHeaders) > 0 {
			transport.ProxyConnectHeader = connectHeaders
		}
	}

	if headers := toHeaderMap(dest.GetHeaders()); len(headers) > 0 {
		opts = append(opts, otlpmetrichttp.WithHeaders(headers))
	}

	opts = append(opts, otlpmetrichttp.WithHTTPClient(&http.Client{
		Transport: transport,
		Timeout:   exportTimeout,
	}))
	return opts, nil
}

// tlsConfig builds the TLS configuration used to reach an https destination.
func tlsConfig(dest *protobufs.TelemetryConnectionSettings) (*tls.Config, error) {
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}

	var rootCAs *x509.CertPool
	if settings := dest.GetTls(); settings != nil {
		minVersion, err := tlsVersion(settings.GetMinVersion())
		if err != nil {
			return nil, fmt.Errorf("invalid tls.min_version: %w", err)
		}
		if minVersion != 0 {
			cfg.MinVersion = minVersion
		}
		maxVersion, err := tlsVersion(settings.GetMaxVersion())
		if err != nil {
			return nil, fmt.Errorf("invalid tls.max_version: %w", err)
		}
		cfg.MaxVersion = maxVersion
		if cfg.MaxVersion != 0 && cfg.MaxVersion < cfg.MinVersion {
			return nil, fmt.Errorf("tls.max_version %q is lower than the minimum TLS version", settings.GetMaxVersion())
		}

		cipherSuites, err := cipherSuiteIDs(settings.GetCipherSuites())
		if err != nil {
			return nil, err
		}
		cfg.CipherSuites = cipherSuites

		cfg.InsecureSkipVerify = settings.GetInsecureSkipVerify() //nolint:gosec // explicitly requested by the OpAMP server.

		if settings.GetIncludeSystemCaCertsPool() {
			rootCAs, err = x509.SystemCertPool()
			if err != nil {
				return nil, fmt.Errorf("unable to load system CA pool: %w", err)
			}
		}
		if ca := settings.GetCaPemContents(); ca != "" {
			if rootCAs == nil {
				rootCAs = x509.NewCertPool()
			}
			if !rootCAs.AppendCertsFromPEM([]byte(ca)) {
				return nil, errors.New("unable to parse tls.ca_pem_contents")
			}
		}
	}

	if certificate := dest.GetCertificate(); certificate != nil {
		if ca := certificate.GetCaCert(); len(ca) > 0 {
			if rootCAs == nil {
				rootCAs = x509.NewCertPool()
			}
			if !rootCAs.AppendCertsFromPEM(ca) {
				return nil, errors.New("unable to parse certificate.ca_cert")
			}
		}
		if len(certificate.GetCert()) > 0 || len(certificate.GetPrivateKey()) > 0 {
			clientCert, err := tls.X509KeyPair(certificate.GetCert(), certificate.GetPrivateKey())
			if err != nil {
				return nil, fmt.Errorf("unable to load client certificate: %w", err)
			}
			cfg.Certificates = []tls.Certificate{clientCert}
		}
	}

	// A nil pool means the host's root CAs are used.
	cfg.RootCAs = rootCAs
	return cfg, nil
}

func tlsVersion(version string) (uint16, error) {
	switch strings.ToUpper(strings.TrimSpace(version)) {
	case "":
		return 0, nil
	case "1.0", "TLSV1", "TLSV1.0":
		return tls.VersionTLS10, nil
	case "1.1", "TLSV1.1":
		return tls.VersionTLS11, nil
	case "1.2", "TLSV1.2":
		return tls.VersionTLS12, nil
	case "1.3", "TLSV1.3":
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf("unsupported TLS version %q", version)
	}
}

func cipherSuiteIDs(names []string) ([]uint16, error) {
	if len(names) == 0 {
		return nil, nil
	}
	known := map[string]uint16{}
	for _, suite := range tls.CipherSuites() {
		known[suite.Name] = suite.ID
	}
	ids := make([]uint16, 0, len(names))
	for _, name := range names {
		id, ok := known[name]
		if !ok {
			return nil, fmt.Errorf("unsupported or insecure TLS cipher suite %q", name)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func toHeaderMap(headers *protobufs.Headers) map[string]string {
	result := map[string]string{}
	for _, header := range headers.GetHeaders() {
		if header.GetKey() == "" {
			continue
		}
		result[header.GetKey()] = header.GetValue()
	}
	return result
}

func toHTTPHeader(headers *protobufs.Headers) http.Header {
	result := http.Header{}
	for _, header := range headers.GetHeaders() {
		if header.GetKey() == "" {
			continue
		}
		result.Add(header.GetKey(), header.GetValue())
	}
	return result
}
