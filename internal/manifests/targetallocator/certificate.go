// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package targetallocator

import (
	"fmt"
	"time"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-telemetry/opentelemetry-operator/internal/manifests/manifestutils"
	"github.com/open-telemetry/opentelemetry-operator/internal/naming"
)

const (
	// ClientCertDuration is the validity period for client and server certificates (90 days).
	// cert-manager defaults to renewing at 2/3 of the duration (~60 days), ensuring certificates
	// are refreshed well before expiration (we'll keep it at 90d explicitly).
	ClientCertDuration = time.Hour * 24 * 90

	// CACertRenewBefore defines when the CA certificate should begin renewal (181 days before expiry).
	// Set to 2x ClientCertDuration + 1 day to ensure:
	// 1. CA renewal doesn't coincide with client/server renewal cycles (which occur every 60 days: day 60, 120, 180, 240, 300, 360, 420, 480, 540...).
	// 2. Without the +1 day offset, CA would renew at day 540 (when 180 days remain), colliding with the 9th client cert renewal.
	// 3. With +1 day, CA renews at day 539 (when 181 days remain), avoiding the race condition.
	// 4. The CA always has sufficient remaining validity (≥181 days) to safely issue 90-day client/server certificates.
	CACertRenewBefore = ClientCertDuration*2 + 24*time.Hour

	// CACertDuration is the validity period for the CA certificate (720 days = ~2 years).
	// Set to 8x ClientCertDuration to prevent renewal race conditions where client and server
	// certificates might be signed by different CA versions during simultaneous renewal.
	// This ensures the CA remains stable through multiple client/server certificate renewal cycles.
	CACertDuration = ClientCertDuration * 8
)

// / CACertificate returns a CA Certificate for the given instance.
func CACertificate(params Params) *cmv1.Certificate {
	name := naming.CACertificate(params.TargetAllocator.Name)
	labels := manifestutils.Labels(params.TargetAllocator.ObjectMeta, name, params.TargetAllocator.Spec.Image, ComponentOpenTelemetryTargetAllocator, nil)

	return &cmv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: params.TargetAllocator.Namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: cmv1.CertificateSpec{
			IsCA:       true,
			CommonName: naming.CACertificate(params.TargetAllocator.Name),
			// Use longer duration and renewBefore to prevent renewal race conditions with client/server certs
			Duration:    &metav1.Duration{Duration: CACertDuration},
			RenewBefore: &metav1.Duration{Duration: CACertRenewBefore},
			Subject: &cmv1.X509Subject{
				OrganizationalUnits: []string{"opentelemetry-operator"},
			},
			SecretName: naming.CACertificate(params.TargetAllocator.Name),
			IssuerRef: cmmeta.ObjectReference{
				Name: naming.SelfSignedIssuer(params.TargetAllocator.Name),
				Kind: "Issuer",
			},
		},
	}
}

// ServingCertificate returns a serving Certificate for the given instance.
func ServingCertificate(params Params) *cmv1.Certificate {
	name := naming.TAServerCertificate(params.TargetAllocator.Name)
	labels := manifestutils.Labels(params.TargetAllocator.ObjectMeta, name, params.TargetAllocator.Spec.Image, ComponentOpenTelemetryTargetAllocator, nil)

	return &cmv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: params.TargetAllocator.Namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: cmv1.CertificateSpec{
			Duration: &metav1.Duration{Duration: ClientCertDuration},
			DNSNames: []string{
				naming.TAService(params.TargetAllocator.Name),
				fmt.Sprintf("%s.%s.svc", naming.TAService(params.TargetAllocator.Name), params.TargetAllocator.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster.local", naming.TAService(params.TargetAllocator.Name), params.TargetAllocator.Namespace),
			},
			IssuerRef: cmmeta.ObjectReference{
				Kind: "Issuer",
				Name: naming.CAIssuer(params.TargetAllocator.Name),
			},
			Usages: []cmv1.KeyUsage{
				cmv1.UsageClientAuth,
				cmv1.UsageServerAuth,
			},
			SecretName: naming.TAServerCertificate(params.TargetAllocator.Name),
			Subject: &cmv1.X509Subject{
				OrganizationalUnits: []string{"opentelemetry-operator"},
			},
		},
	}
}

// ClientCertificate returns a client Certificate for the given instance.
func ClientCertificate(params Params) *cmv1.Certificate {
	name := naming.TAClientCertificate(params.TargetAllocator.Name)
	labels := manifestutils.Labels(params.TargetAllocator.ObjectMeta, name, params.TargetAllocator.Spec.Image, ComponentOpenTelemetryTargetAllocator, nil)

	return &cmv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: params.TargetAllocator.Namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: cmv1.CertificateSpec{
			Duration: &metav1.Duration{Duration: ClientCertDuration},
			DNSNames: []string{
				naming.TAService(params.TargetAllocator.Name),
				fmt.Sprintf("%s.%s.svc", naming.TAService(params.TargetAllocator.Name), params.TargetAllocator.Namespace),
				fmt.Sprintf("%s.%s.svc.cluster.local", naming.TAService(params.TargetAllocator.Name), params.TargetAllocator.Namespace),
			},
			IssuerRef: cmmeta.ObjectReference{
				Kind: "Issuer",
				Name: naming.CAIssuer(params.TargetAllocator.Name),
			},
			Usages: []cmv1.KeyUsage{
				cmv1.UsageClientAuth,
				cmv1.UsageServerAuth,
			},
			SecretName: naming.TAClientCertificate(params.TargetAllocator.Name),
			Subject: &cmv1.X509Subject{
				OrganizationalUnits: []string{"opentelemetry-operator"},
			},
		},
	}
}
