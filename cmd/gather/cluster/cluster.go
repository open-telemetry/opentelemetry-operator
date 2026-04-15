// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"

	routev1 "github.com/openshift/api/route/v1"
	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policy1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/cmd/gather/config"
)

type Cluster struct {
	config               *config.Config
	apiAvailabilityCache map[schema.GroupVersionResource]bool
}

func NewCluster(cfg *config.Config) Cluster {
	return Cluster{
		config:               cfg,
		apiAvailabilityCache: make(map[schema.GroupVersionResource]bool),
	}
}

func (c *Cluster) getOperatorNamespace() (string, error) {
	if c.config.OperatorNamespace != "" {
		return c.config.OperatorNamespace, nil
	}

	deployment, err := c.getOperatorDeployment()
	if err != nil {
		return "", err
	}

	c.config.OperatorNamespace = deployment.Namespace

	return c.config.OperatorNamespace, nil
}

func (c *Cluster) getOperatorDeployment() (appsv1.Deployment, error) {
	operatorDeployments := appsv1.DeploymentList{}
	err := c.config.KubernetesClient.List(context.TODO(), &operatorDeployments, &client.ListOptions{
		Limit: 1,
		LabelSelector: labels.SelectorFromSet(labels.Set{
			"app.kubernetes.io/name": "opentelemetry-operator",
		}),
	})
	if err != nil {
		return appsv1.Deployment{}, err
	}

	if len(operatorDeployments.Items) == 0 {
		return appsv1.Deployment{}, errors.New("operator not found")
	}

	return operatorDeployments.Items[0], nil
}

func (c *Cluster) GetOperatorLogs() error {
	deployment, err := c.getOperatorDeployment()
	if err != nil {
		return err
	}

	labelSelector := labels.Set(deployment.Spec.Selector.MatchLabels).AsSelectorPreValidated()
	operatorPods := corev1.PodList{}
	err = c.config.KubernetesClient.List(context.TODO(), &operatorPods, &client.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return err
	}

	if len(operatorPods.Items) == 0 {
		return errors.New("no operator pods found")
	}

	for i := range operatorPods.Items {
		pod := &operatorPods.Items[i]
		writeToFile(c.config.CollectionDir, pod, c.config.Scheme)
		c.getPodLogs(pod.Name, pod.Namespace, "manager")
	}
	return nil
}

func (c *Cluster) getPodLogs(podName, namespace, container string) {
	pods := c.config.KubernetesClientSet.CoreV1().Pods(namespace)
	writeLogToFile(c.config.CollectionDir, namespace, podName, container, pods)
}

func (c *Cluster) GetOperatorDeploymentInfo() error {
	deployment, err := c.getOperatorDeployment()
	if err != nil {
		return err
	}
	writeToFile(c.config.CollectionDir, &deployment, c.config.Scheme)
	return nil
}

func (c *Cluster) GetOLMInfo() error {
	if !c.isAPIAvailable(schema.GroupVersionResource{
		Group:    operatorsv1.SchemeGroupVersion.Group,
		Version:  operatorsv1.SchemeGroupVersion.Version,
		Resource: "Operator",
	}) {
		log.Println("OLM info not available")
		return nil
	}

	operatorNamespace, err := c.getOperatorNamespace()
	if err != nil {
		return err
	}

	// Operators are cluster-scoped — list without namespace filter.
	operators := operatorsv1.OperatorList{}
	if err := c.config.KubernetesClient.List(context.TODO(), &operators); err != nil {
		return err
	}
	for i := range operators.Items {
		writeToFile(c.config.CollectionDir, &operators.Items[i], c.config.Scheme)
	}

	// OperatorGroups
	operatorGroups := operatorsv1.OperatorGroupList{}
	if err := c.config.KubernetesClient.List(context.TODO(), &operatorGroups, &client.ListOptions{
		Namespace: operatorNamespace,
	}); err != nil {
		return err
	}
	for i := range operatorGroups.Items {
		if strings.Contains(operatorGroups.Items[i].Name, "opentelemetry") {
			writeToFile(c.config.CollectionDir, &operatorGroups.Items[i], c.config.Scheme)
		}
	}

	// Subscriptions
	subscriptions := operatorsv1alpha1.SubscriptionList{}
	if err := c.config.KubernetesClient.List(context.TODO(), &subscriptions, &client.ListOptions{
		Namespace: operatorNamespace,
	}); err != nil {
		return err
	}
	for i := range subscriptions.Items {
		writeToFile(c.config.CollectionDir, &subscriptions.Items[i], c.config.Scheme)
	}

	// InstallPlans
	ips := operatorsv1alpha1.InstallPlanList{}
	if err := c.config.KubernetesClient.List(context.TODO(), &ips, &client.ListOptions{
		Namespace: operatorNamespace,
	}); err != nil {
		return err
	}
	for i := range ips.Items {
		writeToFile(c.config.CollectionDir, &ips.Items[i], c.config.Scheme)
	}

	// ClusterServiceVersions
	csvs := operatorsv1alpha1.ClusterServiceVersionList{}
	if err := c.config.KubernetesClient.List(context.TODO(), &csvs, &client.ListOptions{
		Namespace: operatorNamespace,
	}); err != nil {
		return err
	}
	for i := range csvs.Items {
		if strings.Contains(csvs.Items[i].Name, "opentelemetry") {
			writeToFile(c.config.CollectionDir, &csvs.Items[i], c.config.Scheme)
		}
	}

	return nil
}

func (c *Cluster) GetCRDs() error {
	crds := apiextensionsv1.CustomResourceDefinitionList{}
	if err := c.config.KubernetesClient.List(context.TODO(), &crds); err != nil {
		return err
	}

	log.Println("CRDs found:", len(crds.Items))

	for i := range crds.Items {
		if strings.HasSuffix(crds.Items[i].Name, ".opentelemetry.io") {
			writeToFile(c.config.CollectionDir, &crds.Items[i], c.config.Scheme)
		}
	}
	return nil
}

func (c *Cluster) GetOpenTelemetryCollectors() error {
	otelCols := otelv1beta1.OpenTelemetryCollectorList{}

	err := c.config.KubernetesClient.List(context.TODO(), &otelCols)
	if err != nil {
		return err
	}

	log.Println("OpenTelemetryCollectors found:", len(otelCols.Items))

	errorDetected := false

	for i := range otelCols.Items {
		if err := c.processOTELCollector(&otelCols.Items[i]); err != nil {
			log.Fatalln(err)
			errorDetected = true
		}
	}

	if errorDetected {
		return errors.New("something failed while getting the opentelemtrycollectors")
	}
	return nil
}

func (c *Cluster) GetTargetAllocators() error {
	tas := otelv1alpha1.TargetAllocatorList{}

	err := c.config.KubernetesClient.List(context.TODO(), &tas)
	if err != nil {
		return err
	}

	log.Println("TargetAllocators found:", len(tas.Items))

	errorDetected := false

	for i := range tas.Items {
		if err := c.processOTELTargetAllocator(&tas.Items[i]); err != nil {
			log.Fatalln(err)
			errorDetected = true
		}
	}

	if errorDetected {
		return errors.New("something failed while getting the targetallocators")
	}
	return nil
}

func (c *Cluster) GetOpAMPBridges() error {
	bridges := otelv1alpha1.OpAMPBridgeList{}

	err := c.config.KubernetesClient.List(context.TODO(), &bridges)
	if err != nil {
		return err
	}

	log.Println("OpAMPBridges found:", len(bridges.Items))

	errorDetected := false

	for i := range bridges.Items {
		writeToFile(c.config.CollectionDir, &bridges.Items[i], c.config.Scheme)
		if err := c.processOwnedResources(&bridges.Items[i]); err != nil {
			log.Fatalln(err)
			errorDetected = true
		}
	}

	if errorDetected {
		return errors.New("something failed while getting opampbridges")
	}
	return nil
}

func (c *Cluster) GetInstrumentations() error {
	instrumentations := otelv1alpha1.InstrumentationList{}

	err := c.config.KubernetesClient.List(context.TODO(), &instrumentations)
	if err != nil {
		return err
	}

	log.Println("Instrumentations found:", len(instrumentations.Items))

	for i := range instrumentations.Items {
		writeToFile(c.config.CollectionDir, &instrumentations.Items[i], c.config.Scheme)
	}
	return nil
}

func (c *Cluster) processOTELCollector(otelCol *otelv1beta1.OpenTelemetryCollector) error {
	log.Printf("Processing OpenTelemetryCollector %s/%s", otelCol.Namespace, otelCol.Name)
	writeToFile(c.config.CollectionDir, otelCol, c.config.Scheme)
	return c.processOwnedResources(otelCol)
}

func (c *Cluster) processOTELTargetAllocator(ta *otelv1alpha1.TargetAllocator) error {
	log.Printf("Processing TargetAllocator %s/%s", ta.Namespace, ta.Name)
	writeToFile(c.config.CollectionDir, ta, c.config.Scheme)
	return c.processOwnedResources(ta)
}

func (c *Cluster) processOwnedResources(owner any) error {
	resourceTypes := []struct {
		list     client.ObjectList
		apiCheck func() bool
	}{
		{&otelv1alpha1.TargetAllocatorList{}, func() bool { return true }},
		{&appsv1.DaemonSetList{}, func() bool { return true }},
		{&appsv1.DeploymentList{}, func() bool { return true }},
		{&appsv1.StatefulSetList{}, func() bool { return true }},
		{&rbacv1.ClusterRoleList{}, func() bool { return true }},
		{&rbacv1.ClusterRoleBindingList{}, func() bool { return true }},
		{&corev1.ConfigMapList{}, func() bool { return true }},
		{&corev1.PersistentVolumeList{}, func() bool { return true }},
		{&corev1.PersistentVolumeClaimList{}, func() bool { return true }},
		{&corev1.ServiceList{}, func() bool { return true }},
		{&corev1.ServiceAccountList{}, func() bool { return true }},
		{&autoscalingv2.HorizontalPodAutoscalerList{}, func() bool { return true }},
		{&networkingv1.IngressList{}, func() bool { return true }},
		{&policy1.PodDisruptionBudgetList{}, func() bool { return true }},
		{&monitoringv1.PodMonitorList{}, c.isMonitoringAPIAvailable},
		{&monitoringv1.ServiceMonitorList{}, c.isMonitoringAPIAvailable},
		{&routev1.RouteList{}, c.isRouteAPIAvailable},
	}

	for _, rt := range resourceTypes {
		if rt.apiCheck() {
			if err := c.processResourceType(rt.list, owner); err != nil {
				return err
			}
		}
	}

	// Pods are owned by ReplicaSets, not directly by the CR, so they never match
	// an owner-reference check. Collect them by the instance label instead.
	return c.processPodsByInstance(owner)
}

// processPodsByInstance collects pods that belong to owner using the
// app.kubernetes.io/instance label (format: <namespace>.<name>). This is
// necessary because pods are owned by ReplicaSets, not directly by the CR.
func (c *Cluster) processPodsByInstance(owner any) error {
	var namespace, name string
	switch o := owner.(type) {
	case *otelv1beta1.OpenTelemetryCollector:
		namespace, name = o.Namespace, o.Name
	case *otelv1alpha1.TargetAllocator:
		namespace, name = o.Namespace, o.Name
	case *otelv1alpha1.OpAMPBridge:
		namespace, name = o.Namespace, o.Name
	default:
		return nil
	}

	pods := corev1.PodList{}
	if err := c.config.KubernetesClient.List(context.TODO(), &pods, &client.ListOptions{
		Namespace: namespace,
		LabelSelector: labels.SelectorFromSet(labels.Set{
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", namespace, name),
		}),
	}); err != nil {
		return fmt.Errorf("failed to list pods for %s/%s: %w", namespace, name, err)
	}

	for i := range pods.Items {
		writeToFile(c.config.CollectionDir, &pods.Items[i], c.config.Scheme)
	}
	return nil
}

func (c *Cluster) getOwnerResources(objList client.ObjectList, owner any) ([]client.Object, error) {
	err := c.config.KubernetesClient.List(context.TODO(), objList, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	})
	if err != nil {
		return nil, err
	}

	var resources []client.Object
	items := reflect.ValueOf(objList).Elem().FieldByName("Items")
	for i := 0; i < items.Len(); i++ {
		item := items.Index(i).Addr().Interface().(client.Object)
		if hasOwnerReference(item, owner) {
			resources = append(resources, item)
		}
	}
	return resources, nil
}

func (c *Cluster) processResourceType(list client.ObjectList, owner any) error {
	resources, err := c.getOwnerResources(list, owner)
	if err != nil {
		return fmt.Errorf("failed to get resources: %w", err)
	}
	for _, resource := range resources {
		writeToFile(c.config.CollectionDir, resource, c.config.Scheme)
	}
	return nil
}

func (c *Cluster) isMonitoringAPIAvailable() bool {
	return c.isAPIAvailable(schema.GroupVersionResource{
		Group:    monitoringv1.SchemeGroupVersion.Group,
		Version:  monitoringv1.SchemeGroupVersion.Version,
		Resource: "ServiceMonitor",
	})
}

func (c *Cluster) isRouteAPIAvailable() bool {
	return c.isAPIAvailable(schema.GroupVersionResource{
		Group:    routev1.GroupName,
		Version:  routev1.GroupVersion.Version,
		Resource: "Route",
	})
}

func (c *Cluster) isAPIAvailable(gvr schema.GroupVersionResource) bool {
	if result, ok := c.apiAvailabilityCache[gvr]; ok {
		return result
	}

	rm := c.config.KubernetesClient.RESTMapper()

	gvk, err := rm.KindFor(gvr)
	result := err == nil && !gvk.Empty()
	c.apiAvailabilityCache[gvr] = result

	return result
}

func hasOwnerReference(obj client.Object, owner any) bool {
	var ownerKind string
	var ownerUID types.UID

	// Use hardcoded kind strings — TypeMeta is not populated on controller-runtime List items.
	switch o := owner.(type) {
	case *otelv1beta1.OpenTelemetryCollector:
		ownerKind = "OpenTelemetryCollector"
		ownerUID = o.UID
	case *otelv1alpha1.TargetAllocator:
		ownerKind = "TargetAllocator"
		ownerUID = o.UID
	case *otelv1alpha1.OpAMPBridge:
		ownerKind = "OpAMPBridge"
		ownerUID = o.UID
	default:
		return false
	}

	for _, ownerRef := range obj.GetOwnerReferences() {
		if ownerRef.Kind == ownerKind && ownerRef.UID == ownerUID {
			return true
		}
	}
	return false
}
