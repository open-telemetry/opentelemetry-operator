package upgrade_test

import (
    "context"
    "testing"

	"github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
     metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/types"

    "github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
    "github.com/open-telemetry/opentelemetry-operator/internal/version"
    "github.com/open-telemetry/opentelemetry-operator/pkg/collector/upgrade"
)

func TestRemoveMetricsArgs(t *testing.T) {
    // prepare
    nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
    existing := v1alpha1.OpenTelemetryCollector{
        ObjectMeta: metav1.ObjectMeta{
            Name:      nsn.Name,
            Namespace: nsn.Namespace,
            Labels: map[string]string{
                "app.kubernetes.io/managed-by": "opentelemetry-operator",
            },
        },
        Spec: v1alpha1.OpenTelemetryCollectorSpec{
            Args: map[string]string{
                "--metrics-addr":  ":8988",
                "--metrics-level": "detailed",
            },
        },
    }
    existing.Status.Version = "0.42.0"

    require.Contains(t, existing.Spec.Args, "--metrics-addr")
    require.Contains(t, existing.Spec.Args, "--metrics-level")

    // test
    res, err := upgrade.ManagedInstance(context.Background(), logger, version.Get(), nil, existing)
    assert.NoError(t, err)

    // verify
    assert.NotContains(t, res.Spec.Args, "--metrics-addr")
    assert.NotContains(t, res.Spec.Args, "--metrics-level")
}