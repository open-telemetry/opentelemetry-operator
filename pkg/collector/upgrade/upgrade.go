package upgrade

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
)

// ManagedInstances finds all the otelcol instances for the current operator and upgrades them, if necessary
func ManagedInstances(ctx context.Context, logger logr.Logger, cl client.Client) error {
	logger.Info("looking for managed instances to upgrade")

	opts := []client.ListOption{
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	}
	list := &v1alpha1.OpenTelemetryCollectorList{}
	if err := cl.List(ctx, list, opts...); err != nil {
		return fmt.Errorf("failed to list: %w", err)
	}

	for _, j := range list.Items {
		otelcol, err := ManagedInstance(ctx, logger, cl, &j)
		if err != nil {
			// nothing to do at this level, just go to the next instance
			continue
		}

		if !reflect.DeepEqual(otelcol, j) {
			// the resource update overrides the status, so, keep it so that we can reset it later
			st := otelcol.Status
			if err := cl.Update(ctx, otelcol); err != nil {
				logger.Error(err, "failed to apply changes to instance", "name", otelcol.Name, "namespace", otelcol.Namespace)
				continue
			}

			// the status object requires its own update
			otelcol.Status = st
			if err := cl.Status().Update(ctx, otelcol); err != nil {
				logger.Error(err, "failed to apply changes to instance's status object", "name", otelcol.Name, "namespace", otelcol.Namespace)
				continue
			}

			logger.Info("instance upgraded", "name", otelcol.Name, "namespace", otelcol.Namespace, "version", otelcol.Status.Version)
		}
	}

	if len(list.Items) == 0 {
		logger.Info("no instances to upgrade")
	}

	return nil
}

// ManagedInstance performs the necessary changes to bring the given otelcol instance to the current version
func ManagedInstance(ctx context.Context, logger logr.Logger, cl client.Client, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {
	if v, ok := versions[otelcol.Status.Version]; ok {
		// we don't need to run the upgrade function for the version 'v', only the next ones
		for n := v.next; n != nil; n = n.next {
			// performs the upgrade to version 'n'
			upgraded, err := n.upgrade(cl, otelcol)
			if err != nil {
				logger.Error(err, "failed to upgrade managed otelcol instances", "name", otelcol.Name, "namespace", otelcol.Namespace)
				return otelcol, err
			}

			logger.V(1).Info("step upgrade", "name", otelcol.Name, "namespace", otelcol.Namespace, "version", n.Tag)
			upgraded.Status.Version = n.Tag
			otelcol = upgraded
		}
	}

	logger.V(1).Info("final version", "name", otelcol.Name, "namespace", otelcol.Namespace, "version", otelcol.Status.Version)
	return otelcol, nil
}
