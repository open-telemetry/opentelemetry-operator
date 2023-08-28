package openshift

import (
	"context"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDomain(t *testing.T) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "openshift-ingress-operator",
		},
	}
	err := k8sClient.Create(context.Background(), ns)
	require.NoError(t, err)

	clusterDNS := &configv1.DNS{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
			//Namespace: "openshift-ingress-operator",
		},
		Spec: configv1.DNSSpec{
			BaseDomain: "test.crc",
		},
	}
	err = k8sClient.Create(context.Background(), clusterDNS)
	require.NoError(t, err)
	domain, err := GetOpenShiftBaseDomain(context.Background(), k8sClient)
	require.NoError(t, err)
	assert.Equal(t, "test.crc", domain)
}
