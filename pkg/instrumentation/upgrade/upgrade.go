package upgrade

import (
	"context"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"strings"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/internal/version"
)

func ManagedInstances(ctx context.Context, logger logr.Logger, ver version.Version, cl client.Client) error {
	logger.Info("looking for managed Instrumentation instances to upgrade")

	opts := []client.ListOption{
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	}
	list := &v1alpha1.InstrumentationList{}
	if err := cl.List(ctx, list, opts...); err != nil {
		return fmt.Errorf("failed to list: %w", err)
	}

	for i := range list.Items {
		instToUpgrade := list.Items[i]
		err := upgrade(ctx, ver, instToUpgrade)
		if err != nil {
			// nothing to do
			continue
		}
	}
}

func upgrade(ctx context.Context, version version.Version, inst v1alpha1.Instrumentation) (v1alpha1.Instrumentation, error) {
	autoJavaSemver, err := semver.NewVersion(version.JavaAutoInstrumentation)
	if err != nil {
		return v1alpha1.Instrumentation{}, err
	}

	autoInstJava := inst.Annotations[v1alpha1.AnnotationDefaultAutoInstrumentationJava]
	if autoInstJava != "" {
		index := strings.Index(autoInstJava, ":")
		if index == -1 {
			return inst, fmt.Errorf("cannot upgrade")
		}
		instanceV, err := semver.NewVersion(autoInstJava[index:])
		if err != nil {
			return inst, err
		}

		if instanceV.LessThan(autoJavaSemver) {
			instanceV.String()
		}
	}
}

