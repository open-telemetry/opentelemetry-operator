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

// ManagedInstances finds all the otelsvc instances for the current operator and upgrades them, if necessary
func ManagedInstances(ctx context.Context, c client.Client) error {
	logger := ctx.Value(opentelemetry.ContextLogger).(logr.Logger)
	logger.Info("looking for managed instances to upgrade")

	list := &v1alpha1.OpenTelemetryServiceList{}
	if err := c.List(ctx, &client.ListOptions{}, list); err != nil {
		return fmt.Errorf("failed to get list of otelsvc instances: %v", err)
	}

	for _, j := range list.Items {
		otelsvc, err := ManagedInstance(ctx, c, &j)
		if err != nil {
			// nothing to do at this level, just go to the next instance
			continue
		}

		if !reflect.DeepEqual(otelsvc, j) {
			// the CR has changed, store it!
			if err := c.Update(ctx, otelsvc); err != nil {
				logger.Error(err, "failed to store the upgraded otelsvc instances", "name", otelsvc.Name, "namespace", otelsvc.Namespace)
				return err
			}

			logger.Info("instance upgraded", "name", otelsvc.Name, "namespace", otelsvc.Namespace)
		}
	}

	if len(list.Items) == 0 {
		logger.Info("no instances to upgrade")
	}

	return nil
}

// ManagedInstance performs the necessary changes to bring the given otelsvc instance to the current version
func ManagedInstance(ctx context.Context, client client.Client, otelsvc *v1alpha1.OpenTelemetryService) (*v1alpha1.OpenTelemetryService, error) {
	logger := ctx.Value(opentelemetry.ContextLogger).(logr.Logger)

	if v, ok := versions[otelsvc.Status.Version]; ok {
		// we don't need to run the upgrade function for the version 'v', only the next ones
		for n := v.next; n != nil; n = n.next {
			// performs the upgrade to version 'n'
			upgraded, err := n.upgrade(client, otelsvc)
			if err != nil {
				logger.Error(err, "failed to upgrade managed otelsvc instances", "name", otelsvc.Name, "namespace", otelsvc.Namespace)
				return otelsvc, err
			}

			upgraded.Status.Version = n.v
			otelsvc = upgraded
		}
	}

	return otelsvc, nil
}
