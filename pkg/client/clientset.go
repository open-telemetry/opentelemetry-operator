package client

import (
	monclient "github.com/coreos/prometheus-operator/pkg/client/versioned"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	otclient "github.com/open-telemetry/opentelemetry-operator/pkg/client/versioned"
)

// Clientset holds all the clients that the reconciler might need to hold
type Clientset struct {
	Kubernetes    kubernetes.Interface
	Monitoring    monclient.Interface
	OpenTelemetry otclient.Interface
}

// ForManager returns a Clientset based on the information from the given manager
func ForManager(mgr manager.Manager) (*Clientset, error) {
	cl, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil, err
	}

	monclient, err := monclient.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil, err
	}

	otclient, err := otclient.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil, err
	}

	return &Clientset{
		Kubernetes:    cl,
		Monitoring:    monclient,
		OpenTelemetry: otclient,
	}, nil
}
