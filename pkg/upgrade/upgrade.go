package upgrade

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry"
	"github.com/open-telemetry/opentelemetry-operator/pkg/apis/opentelemetry/v1alpha1"
)

// ManagedInstances finds all the otelcol instances for the current operator and upgrades them, if necessary
func ManagedInstances(ctx context.Context, c client.Client) error {
	logger := ctx.Value(opentelemetry.ContextLogger).(logr.Logger)
	logger.Info("looking for managed instances to upgrade")

	list := &v1alpha1.OpenTelemetryCollectorList{}
	if err := c.List(ctx, &client.ListOptions{}, list); err != nil {
		return fmt.Errorf("failed to get list of otelcol instances: %v", err)
	}

	for _, j := range list.Items {
		otelcol, err := ManagedInstance(ctx, c, &j)
		if err != nil {
			// nothing to do at this level, just go to the next instance
			continue
		}

		if !reflect.DeepEqual(otelcol, j) {
			// the CR has changed, store it!
			if err := c.Update(ctx, otelcol); err != nil {
				logger.Error(err, "failed to store the upgraded otelcol instances", "name", otelcol.Name, "namespace", otelcol.Namespace)
				return err
			}

			logger.Info("instance upgraded", "name", otelcol.Name, "namespace", otelcol.Namespace)
		}
	}

	if len(list.Items) == 0 {
		logger.Info("no instances to upgrade")
	}

	return nil
}

// ManagedInstance performs the necessary changes to bring the given otelcol instance to the current version
func ManagedInstance(ctx context.Context, client client.Client, otelcol *v1alpha1.OpenTelemetryCollector) (*v1alpha1.OpenTelemetryCollector, error) {
	logger := ctx.Value(opentelemetry.ContextLogger).(logr.Logger)

	if v, ok := versions[otelcol.Status.Version]; ok {
		// we don't need to run the upgrade function for the version 'v', only the next ones
		for n := v.next; n != nil; n = n.next {
			// performs the upgrade to version 'n'
			upgraded, err := n.upgrade(client, otelcol)
			if err != nil {
				logger.Error(err, "failed to upgrade managed otelcol instances", "name", otelcol.Name, "namespace", otelcol.Namespace)
				return otelcol, err
			}

			upgraded.Status.Version = n.v
			otelcol = upgraded
		}
	}

	return otelcol, nil
}
