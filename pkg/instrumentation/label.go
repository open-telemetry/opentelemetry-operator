package instrumentation

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// labelValue returns the effective labelInjectJava value, based on the labels from the pod and namespace.
func labelValue(ns metav1.ObjectMeta, pod metav1.ObjectMeta, label string) string {
	// is the pod labeled with instructions to inject sidecars? is the namespace labeled?
	// if any of those is true, a sidecar might be desired.
	podLabelValue := pod.Labels[label]
	nsLabelValue := ns.Labels[label]

	// if the namespace value is empty, the pod label should be used, whatever it is
	if len(nsLabelValue) == 0 {
		return podLabelValue
	}

	// if the pod value is empty, the label should be used (true, false, instance)
	if len(podLabelValue) == 0 {
		return nsLabelValue
	}

	// the pod label isn't empty -- if it's an instance name, or false, that's the decision
	if !strings.EqualFold(podLabelValue, "true") {
		return podLabelValue
	}

	// pod label is 'true', and if the namespace annotation is false, we just return 'true'
	if strings.EqualFold(nsLabelValue, "false") {
		return podLabelValue
	}

	// by now, the pod label is 'true', and the namespace label is either true or an instance name
	// so, the namespace label can be used
	return nsLabelValue
}