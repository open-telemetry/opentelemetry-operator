package controllers

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	k8sreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
)

func TestNewDistributionOnReconciliation(t *testing.T) {
	// prepare
	cfg := config.New()
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	reconciler := NewDistributionReconciler(Params{
		Client: k8sClient,
		Log:    logger,
		Scheme: testScheme,
		Config: cfg,
	})
	created := &v1alpha1.OpenTelemetryCollectorDistribution{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Command: []string{"/path/to/command"},
		Image:   "quay.io/myns/my-dist:1.0.0",
	}
	err := k8sClient.Create(context.Background(), created)
	require.NoError(t, err)

	// sanity check
	require.Nil(t, cfg.Distribution(created.Namespace, created.Name))

	// test
	req := k8sreconcile.Request{
		NamespacedName: nsn,
	}
	_, err = reconciler.Reconcile(req)
	require.NoError(t, err)

	// verify
	dist := cfg.Distribution(created.Namespace, created.Name)
	assert.NotNil(t, dist)
	assert.Equal(t, []string{"/path/to/command"}, dist.Command)
	assert.Equal(t, "quay.io/myns/my-dist:1.0.0", dist.Image)

	// cleanup
	require.NoError(t, k8sClient.Delete(context.Background(), created))
}

func TestChangedDistribution(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	cfg := config.New()
	reconciler := NewDistributionReconciler(Params{
		Client: k8sClient,
		Log:    logger,
		Scheme: testScheme,
		Config: cfg,
	})

	{
		existing := &v1alpha1.OpenTelemetryCollectorDistribution{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nsn.Name,
				Namespace: nsn.Namespace,
			},
			Command: []string{"/path/to/command"},
			Image:   "quay.io/myns/my-dist:1.0.0",
		}
		err := k8sClient.Create(context.Background(), existing)
		require.NoError(t, err)

		req := k8sreconcile.Request{
			NamespacedName: nsn,
		}
		_, err = reconciler.Reconcile(req)
		require.NoError(t, err)

		// sanity check
		require.NotNil(t, cfg.Distribution(existing.Namespace, existing.Name))
		require.Equal(t, []string{"/path/to/command"}, existing.Command)
		require.Equal(t, "quay.io/myns/my-dist:1.0.0", existing.Image)
	}

	// test
	changed := &v1alpha1.OpenTelemetryCollectorDistribution{}
	err := k8sClient.Get(context.Background(), nsn, changed)
	require.NoError(t, err)

	changed.Command = []string{"/path/to/command/changed"}
	changed.Image = "quay.io/myns/my-dist:1.1.0"

	err = k8sClient.Update(context.Background(), changed)
	require.NoError(t, err)

	req := k8sreconcile.Request{
		NamespacedName: nsn,
	}
	_, err = reconciler.Reconcile(req)
	require.NoError(t, err)

	// verify
	dist := cfg.Distribution(nsn.Namespace, nsn.Name)
	assert.NotNil(t, dist)
	assert.Equal(t, []string{"/path/to/command/changed"}, dist.Command)
	assert.Equal(t, "quay.io/myns/my-dist:1.1.0", dist.Image)

	// cleanup
	require.NoError(t, k8sClient.Delete(context.Background(), dist))
}

func TestDeleteDistribution(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	cfg := config.New()
	reconciler := NewDistributionReconciler(Params{
		Client: k8sClient,
		Log:    logger,
		Scheme: testScheme,
		Config: cfg,
	})

	{
		existing := &v1alpha1.OpenTelemetryCollectorDistribution{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nsn.Name,
				Namespace: nsn.Namespace,
			},
			Command: []string{"/path/to/command"},
			Image:   "quay.io/myns/my-dist:1.0.0",
		}
		err := k8sClient.Create(context.Background(), existing)
		require.NoError(t, err)

		req := k8sreconcile.Request{
			NamespacedName: nsn,
		}
		_, err = reconciler.Reconcile(req)
		require.NoError(t, err)

		// sanity check
		require.NotNil(t, cfg.Distribution(existing.Namespace, existing.Name))
		require.Equal(t, []string{"/path/to/command"}, existing.Command)
		require.Equal(t, "quay.io/myns/my-dist:1.0.0", existing.Image)
	}

	// test
	deleted := &v1alpha1.OpenTelemetryCollectorDistribution{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
	}
	err := k8sClient.Delete(context.Background(), deleted)
	require.NoError(t, err)

	req := k8sreconcile.Request{
		NamespacedName: nsn,
	}
	_, err = reconciler.Reconcile(req)
	require.NoError(t, err)

	// verify
	dist := cfg.Distribution(nsn.Namespace, nsn.Name)
	assert.Nil(t, dist)
}

func TestCommandChanged(t *testing.T) {
	testCases := []struct {
		desc     string
		new      []string
		old      []string
		expected bool
	}{
		{
			"unchanged",
			[]string{"/path/to-command"},
			[]string{"/path/to-command"},
			false,
		},
		{
			"item removed",
			[]string{"/path/to-command"},
			[]string{"/path/to-command", "second part"},
			true,
		},
		{
			"item added",
			[]string{"/path/to-command", "second part"},
			[]string{"/path/to-command"},
			true,
		},
		{
			"item changed",
			[]string{"/path/to-command/updated"},
			[]string{"/path/to-command"},
			true,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			assert.Equal(t, tC.expected, commandChanged(tC.new, tC.old))
		})
	}
}

func TestRegisterDistributionControllerWithManager(t *testing.T) {
	t.Skip("this test requires a real cluster, otherwise the GetConfigOrDie will die")

	// prepare
	mgr, err := manager.New(k8sconfig.GetConfigOrDie(), manager.Options{})
	require.NoError(t, err)

	reconciler := NewDistributionReconciler(Params{
		Client: k8sClient,
		Log:    logger,
		Scheme: testScheme,
	})

	// test
	err = reconciler.SetupWithManager(mgr)

	// verify
	assert.NoError(t, err)
}

func TestLoadDistributions(t *testing.T) {
	ns1otelcol1 := v1alpha1.OpenTelemetryCollectorDistribution{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns1",
			Name:      "otelcol1",
		},
	}
	ns2otelcol1 := v1alpha1.OpenTelemetryCollectorDistribution{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns2",
			Name:      "otelcol1",
		},
	}
	ns2otelcol2 := v1alpha1.OpenTelemetryCollectorDistribution{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns2",
			Name:      "otelcol2",
		},
	}
	ns3otelcol1 := v1alpha1.OpenTelemetryCollectorDistribution{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns3",
			Name:      "otelcol1",
		},
	}

	testCases := []struct {
		desc     string
		existing []v1alpha1.OpenTelemetryCollectorDistribution
		expected []v1alpha1.OpenTelemetryCollectorDistribution
		cfg      *config.Config
	}{
		{
			"no distributions",
			[]v1alpha1.OpenTelemetryCollectorDistribution{},
			[]v1alpha1.OpenTelemetryCollectorDistribution{},
			config.New(),
		},
		{
			"watch all namespaces",
			[]v1alpha1.OpenTelemetryCollectorDistribution{ns1otelcol1, ns2otelcol1, ns2otelcol2, ns3otelcol1},
			[]v1alpha1.OpenTelemetryCollectorDistribution{ns1otelcol1, ns2otelcol1, ns2otelcol2, ns3otelcol1},
			config.New(),
		},
		{
			"watch single namespace",
			[]v1alpha1.OpenTelemetryCollectorDistribution{ns1otelcol1, ns2otelcol1, ns2otelcol2, ns3otelcol1},
			[]v1alpha1.OpenTelemetryCollectorDistribution{ns1otelcol1},
			config.New(config.WithWatchedNamespaces([]string{"ns1"})),
		},
		{
			"watch two namespaces",
			[]v1alpha1.OpenTelemetryCollectorDistribution{ns1otelcol1, ns2otelcol1, ns2otelcol2, ns3otelcol1},
			[]v1alpha1.OpenTelemetryCollectorDistribution{ns1otelcol1, ns2otelcol1, ns2otelcol2},
			config.New(config.WithWatchedNamespaces([]string{"ns1", "ns2"})),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			// prepare
			for i := range tC.existing {
				dist := tC.existing[i]
				ns := corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: dist.Namespace,
					},
				}

				err := k8sClient.Create(context.Background(), &ns)
				if err != nil && !apierrors.IsAlreadyExists(err) {
					require.FailNow(t, err.Error())
				}

				require.NoError(t, k8sClient.Create(context.Background(), &dist))
			}

			reconciler := NewDistributionReconciler(Params{
				Client: k8sClient,
				Log:    logger,
				Scheme: testScheme,
				Config: tC.cfg,
			})

			// test
			err := reconciler.LoadDistributions()

			// verify
			assert.NoError(t, err)
			assert.Len(t, reconciler.distributions, len(tC.expected))
			sort.Slice(reconciler.distributions, func(i, j int) bool {
				if reconciler.distributions[i].Namespace != reconciler.distributions[j].Namespace {
					return reconciler.distributions[i].Namespace < reconciler.distributions[j].Namespace
				}
				return reconciler.distributions[i].Name < reconciler.distributions[j].Name
			})
			for i, dist := range tC.expected {
				assert.Equal(t, tC.expected[i].Namespace, reconciler.distributions[i].Namespace)
				assert.Equal(t, tC.expected[i].Name, reconciler.distributions[i].Name)
				assert.NotNil(t, tC.cfg.Distribution(dist.Namespace, dist.Name))
			}

			// cleanup
			for i := range tC.existing {
				require.NoError(t, k8sClient.Delete(context.Background(), &tC.existing[i]))
			}
		})
	}
}
