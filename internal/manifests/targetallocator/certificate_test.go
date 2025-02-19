// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

type CACertificateConfig struct {
	Name       string
	Namespace  string
	SecretName string
	IssuerName string
}

type ServingCertificateConfig struct {
	Name       string
	Namespace  string
	SecretName string
	IssuerName string
}

type ClientCertificateConfig struct {
	Name       string
	Namespace  string
	SecretName string
	IssuerName string
}

func TestCACertificate(t *testing.T) {
	tests := []struct {
		name             string
		targetAllocator  v1alpha1.TargetAllocator
		expectedCAConfig CACertificateConfig
		expectedLabels   map[string]string
	}{
		{
			name: "Default CA Certificate",
			targetAllocator: v1alpha1.TargetAllocator{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-instance",
					Namespace: "my-namespace",
				},
			},
			expectedCAConfig: CACertificateConfig{
				Name:       "my-instance-ca-cert",
				Namespace:  "my-namespace",
				SecretName: "my-instance-ca-cert",
				IssuerName: "my-instance-self-signed-issuer",
			},
			expectedLabels: map[string]string{
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
				"app.kubernetes.io/instance":   "my-namespace.my-instance",
				"app.kubernetes.io/part-of":    "opentelemetry",
				"app.kubernetes.io/component":  "opentelemetry-targetallocator",
				"app.kubernetes.io/name":       "my-instance-ca-cert",
				"app.kubernetes.io/version":    "latest",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := Params{
				TargetAllocator: tt.targetAllocator,
				Config:          config.New(),
			}

			caCert := CACertificate(params)

			assert.Equal(t, tt.expectedCAConfig.Name, caCert.Name)
			assert.Equal(t, tt.expectedCAConfig.Namespace, caCert.Namespace)
			assert.Equal(t, tt.expectedCAConfig.SecretName, caCert.Spec.SecretName)
			assert.Equal(t, tt.expectedCAConfig.IssuerName, caCert.Spec.IssuerRef.Name)
			assert.True(t, caCert.Spec.IsCA)
			assert.Equal(t, "Issuer", caCert.Spec.IssuerRef.Kind)
			assert.Equal(t, []string{"opentelemetry-operator"}, caCert.Spec.Subject.OrganizationalUnits)
			assert.Equal(t, tt.expectedLabels, caCert.Labels)
		})
	}
}

func TestServingCertificate(t *testing.T) {
	tests := []struct {
		name                     string
		targetAllocator          v1alpha1.TargetAllocator
		expectedServingConfig    ServingCertificateConfig
		expectedDNSNames         []string
		expectedOrganizationUnit []string
		expectedLabels           map[string]string
	}{
		{
			name: "Default Serving Certificate",
			targetAllocator: v1alpha1.TargetAllocator{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-instance",
					Namespace: "my-namespace",
				},
			},
			expectedServingConfig: ServingCertificateConfig{
				Name:       "my-instance-ta-server-cert",
				Namespace:  "my-namespace",
				SecretName: "my-instance-ta-server-cert",
				IssuerName: "my-instance-ca-issuer",
			},
			expectedDNSNames: []string{
				"my-instance-targetallocator",
				"my-instance-targetallocator.my-namespace.svc",
				"my-instance-targetallocator.my-namespace.svc.cluster.local",
			},
			expectedOrganizationUnit: []string{"opentelemetry-operator"},
			expectedLabels: map[string]string{
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
				"app.kubernetes.io/instance":   "my-namespace.my-instance",
				"app.kubernetes.io/part-of":    "opentelemetry",
				"app.kubernetes.io/component":  "opentelemetry-targetallocator",
				"app.kubernetes.io/name":       "my-instance-ta-server-cert",
				"app.kubernetes.io/version":    "latest",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := Params{
				TargetAllocator: tt.targetAllocator,
				Config:          config.New(),
			}

			servingCert := ServingCertificate(params)

			assert.Equal(t, tt.expectedServingConfig.Name, servingCert.Name)
			assert.Equal(t, tt.expectedServingConfig.Namespace, servingCert.Namespace)
			assert.Equal(t, tt.expectedServingConfig.SecretName, servingCert.Spec.SecretName)
			assert.Equal(t, tt.expectedServingConfig.IssuerName, servingCert.Spec.IssuerRef.Name)
			assert.Equal(t, "Issuer", servingCert.Spec.IssuerRef.Kind)
			assert.ElementsMatch(t, tt.expectedDNSNames, servingCert.Spec.DNSNames)
			assert.ElementsMatch(t, tt.expectedOrganizationUnit, servingCert.Spec.Subject.OrganizationalUnits)
			assert.Equal(t, tt.expectedLabels, servingCert.Labels)
		})
	}
}

func TestClientCertificate(t *testing.T) {
	tests := []struct {
		name                     string
		targetAllocator          v1alpha1.TargetAllocator
		expectedClientConfig     ClientCertificateConfig
		expectedDNSNames         []string
		expectedOrganizationUnit []string
		expectedLabels           map[string]string
	}{
		{
			name: "Default Client Certificate",
			targetAllocator: v1alpha1.TargetAllocator{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-instance",
					Namespace: "my-namespace",
				},
			},
			expectedClientConfig: ClientCertificateConfig{
				Name:       "my-instance-ta-client-cert",
				Namespace:  "my-namespace",
				SecretName: "my-instance-ta-client-cert",
				IssuerName: "my-instance-ca-issuer",
			},
			expectedDNSNames: []string{
				"my-instance-targetallocator",
				"my-instance-targetallocator.my-namespace.svc",
				"my-instance-targetallocator.my-namespace.svc.cluster.local",
			},
			expectedOrganizationUnit: []string{"opentelemetry-operator"},
			expectedLabels: map[string]string{
				"app.kubernetes.io/managed-by": "opentelemetry-operator",
				"app.kubernetes.io/instance":   "my-namespace.my-instance",
				"app.kubernetes.io/part-of":    "opentelemetry",
				"app.kubernetes.io/component":  "opentelemetry-targetallocator",
				"app.kubernetes.io/name":       "my-instance-ta-client-cert",
				"app.kubernetes.io/version":    "latest",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := Params{
				TargetAllocator: tt.targetAllocator,
				Config:          config.New(),
			}

			clientCert := ClientCertificate(params)

			assert.Equal(t, tt.expectedClientConfig.Name, clientCert.Name)
			assert.Equal(t, tt.expectedClientConfig.Namespace, clientCert.Namespace)
			assert.Equal(t, tt.expectedClientConfig.SecretName, clientCert.Spec.SecretName)
			assert.Equal(t, tt.expectedClientConfig.IssuerName, clientCert.Spec.IssuerRef.Name)
			assert.Equal(t, "Issuer", clientCert.Spec.IssuerRef.Kind)
			assert.ElementsMatch(t, tt.expectedDNSNames, clientCert.Spec.DNSNames)
			assert.ElementsMatch(t, tt.expectedOrganizationUnit, clientCert.Spec.Subject.OrganizationalUnits)
			assert.Equal(t, tt.expectedLabels, clientCert.Labels)
		})
	}
}
