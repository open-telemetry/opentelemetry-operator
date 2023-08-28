// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package openshift

import (
	"context"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetOpenShiftBaseDomain returns base domain of OCP cluster.
func GetOpenShiftBaseDomain(ctx context.Context, k8sClient client.Client) (string, error) {
	ictrl := &operatorv1.IngressController{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: "default", Namespace: "openshift-ingress-operator"}, ictrl)
	if err != nil {
		// The preferred way to get the base domain is via OCP ingress controller
		// this approach works well with CRC and normal OCP cluster.
		// Fallback on cluster DNS might not work with CRC because CRC uses .apps-crc. and not .apps.
		var clusterDNS configv1.DNS
		if errDNS := k8sClient.Get(ctx, client.ObjectKey{Name: "cluster"}, &clusterDNS); errDNS != nil {
			if apierrors.IsNotFound(errDNS) {
				return "", fmt.Errorf("missing OpenShift IngressController and cluster DNS configuration to read base domain: %w", err)
			}
			return "", fmt.Errorf("failed to lookup base domain: %w", err)
		}

		return clusterDNS.Spec.BaseDomain, nil
	}
	return ictrl.Status.Domain, nil
}
