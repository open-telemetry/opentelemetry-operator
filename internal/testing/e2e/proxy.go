// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// PodHTTPGet finds a pod by label selector and performs an HTTP GET against its port
// via the API server's pod proxy, so tests can drive workloads that have no Service.
// It returns an error (rather than failing the test) so callers can retry it, e.g.
// while repeatedly driving a sample app so it keeps emitting telemetry.
func PodHTTPGet(ctx context.Context, cfg *envconf.Config, ns, selector string, port int, path string) ([]byte, error) {
	cs, err := clientSet(cfg)
	if err != nil {
		return nil, err
	}
	pods, err := cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, fmt.Errorf("list pods %q in %s: %w", selector, ns, err)
	}
	pod := ""
	for _, p := range pods.Items {
		if p.Status.Phase == corev1.PodRunning && p.DeletionTimestamp == nil {
			pod = p.Name
			break
		}
	}
	if pod == "" {
		return nil, fmt.Errorf("no running pod matches %q in %s (%d pods)", selector, ns, len(pods.Items))
	}
	body, err := cs.CoreV1().Pods(ns).
		ProxyGet("http", pod, strconv.Itoa(port), path, nil).
		DoRaw(ctx)
	if err != nil {
		return nil, fmt.Errorf("GET pod %s port %d path %q: %w", pod, port, path, err)
	}
	return body, nil
}

// ServiceHTTPGet performs an HTTP GET against a Service port via the API server's
// service proxy (works for headless Services too). Like PodHTTPGet it returns an
// error so it can be polled.
func ServiceHTTPGet(ctx context.Context, cfg *envconf.Config, ns, service string, port int, path string) ([]byte, error) {
	cs, err := clientSet(cfg)
	if err != nil {
		return nil, err
	}
	body, err := cs.CoreV1().Services(ns).
		ProxyGet("http", service, strconv.Itoa(port), path, nil).
		DoRaw(ctx)
	if err != nil {
		return nil, fmt.Errorf("GET service %s port %d path %q: %w", service, port, path, err)
	}
	return body, nil
}
