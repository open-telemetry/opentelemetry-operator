package webhookhandler

import corev1 "k8s.io/api/core/v1"

// Checks if Pod is already instrumented by checking Instrumentation InitContainer presence
func IsPodInstrumentationMissing(pod corev1.Pod) bool {
	for _, cont := range pod.Spec.InitContainers {
		if cont.Name == initContainerName {
			return false
		}
	}
	return true
}
