// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
		Limit:         1,
		LabelSelector: labelSelector,
	})
	if err != nil {
		return err
	}

	if len(operatorPods.Items) == 0 {
		return errors.New("no operator pods found")
	}

	pod := operatorPods.Items[0]
	c.getPodLogs(pod.Name, pod.Namespace, "manager")
	return nil
}

func (c *Cluster) getPodLogs(podName, namespace, container string) {
	pods := c.config.KubernetesClientSet.CoreV1().Pods(namespace)
	writeLogToFile(c.config.CollectionDir, podName, container, pods)
}

func (c *Cluster) GetOperatorDeploymentInfo() error {
	err := os.MkdirAll(c.config.CollectionDir, os.ModePerm)
	if err != nil {
		return err
	}

	deployment, err := c.getOperatorDeployment()
	if err != nil {
		return err
	}

	writeToFile(c.config.CollectionDir, &deployment)

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

	outputDir := filepath.Join(c.config.CollectionDir, "olm")
	err := os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		return err
	}

	operatorNamespace, err := c.getOperatorNamespace()
	if err != nil {
		return err
	}

	// Operators
	operators := operatorsv1.OperatorList{}
	err = c.config.KubernetesClient.List(context.TODO(), &operators, &client.ListOptions{
		Namespace: operatorNamespace,
	})
	if err != nil {
		return err
	}
	for _, o := range operators.Items {
		o := o
		writeToFile(outputDir, &o)

	}

	// OperatorGroups
	operatorGroups := operatorsv1.OperatorGroupList{}
	err = c.config.KubernetesClient.List(context.TODO(), &operatorGroups, &client.ListOptions{
		Namespace: operatorNamespace,
	})
	if err != nil {
		return err
	}
	for _, o := range operatorGroups.Items {
		o := o
		if strings.Contains(o.Name, "opentelemetry") {
			writeToFile(outputDir, &o)
		}
	}

	// Subscription
	subscriptions := operatorsv1alpha1.SubscriptionList{}
	err = c.config.KubernetesClient.List(context.TODO(), &subscriptions, &client.ListOptions{
		Namespace: operatorNamespace,
	})
	if err != nil {
		return err
	}
	for _, o := range subscriptions.Items {
		o := o
		writeToFile(outputDir, &o)

	}

	// InstallPlan
	ips := operatorsv1alpha1.InstallPlanList{}
	err = c.config.KubernetesClient.List(context.TODO(), &ips, &client.ListOptions{
		Namespace: operatorNamespace,
	})
	if err != nil {
		return err
	}
	for _, o := range ips.Items {
		o := o
		writeToFile(outputDir, &o)
	}

	// ClusterServiceVersion
	csvs := operatorsv1alpha1.ClusterServiceVersionList{}
	err = c.config.KubernetesClient.List(context.TODO(), &csvs, &client.ListOptions{
		Namespace: operatorNamespace,
	})
	if err != nil {
		return err
	}
	for _, o := range csvs.Items {
		o := o
		if strings.Contains(o.Name, "opentelemetry") {
			writeToFile(outputDir, &o)
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

	for _, otelCol := range otelCols.Items {
		otelCol := otelCol
		err := c.processOTELCollector(&otelCol)
		if err != nil {
			log.Fatalln(err)
			errorDetected = true
		}
	}

	if errorDetected {
		return fmt.Errorf("something failed while getting the opentelemtrycollectors")
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

	errorDetected := false

	for _, instr := range instrumentations.Items {
		instr := instr
		outputDir := filepath.Join(c.config.CollectionDir, instr.Namespace)
		err := os.MkdirAll(outputDir, os.ModePerm)
		if err != nil {
			log.Fatalln(err)
			errorDetected = true
			continue
		}

		writeToFile(outputDir, &instr)
	}

	if errorDetected {
		return fmt.Errorf("something failed while getting the opentelemtrycollectors")
	}
	return nil
}

func (c *Cluster) processOTELCollector(otelCol *otelv1beta1.OpenTelemetryCollector) error {
	log.Printf("Processing OpenTelemetryCollector %s/%s", otelCol.Namespace, otelCol.Name)
	folder, err := createOTELFolder(c.config.CollectionDir, otelCol)
	if err != nil {
		return err
	}
	writeToFile(folder, otelCol)

	err = c.processOwnedResources(otelCol, folder)
	if err != nil {
		return err
	}

	return nil
}

func (c *Cluster) processOwnedResources(owner interface{}, folder string) error {
	resourceTypes := []struct {
		list     client.ObjectList
		apiCheck func() bool
	}{
		{&appsv1.DaemonSetList{}, func() bool { return true }},
		{&appsv1.DeploymentList{}, func() bool { return true }},
		{&appsv1.StatefulSetList{}, func() bool { return true }},
		{&rbacv1.ClusterRoleList{}, func() bool { return true }},
		{&rbacv1.ClusterRoleBindingList{}, func() bool { return true }},
		{&corev1.ConfigMapList{}, func() bool { return true }},
		{&corev1.PersistentVolumeList{}, func() bool { return true }},
		{&corev1.PersistentVolumeClaimList{}, func() bool { return true }},
		{&corev1.PodList{}, func() bool { return true }},
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
			if err := c.processResourceType(rt.list, owner, folder); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Cluster) getOwnerResources(objList client.ObjectList, owner interface{}) ([]client.Object, error) {
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

func (c *Cluster) processResourceType(list client.ObjectList, owner interface{}, folder string) error {
	resources, err := c.getOwnerResources(list, owner)
	if err != nil {
		return fmt.Errorf("failed to get resources: %w", err)
	}
	for _, resource := range resources {
		writeToFile(folder, resource)
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

func hasOwnerReference(obj client.Object, owner interface{}) bool {
	var ownerKind string
	var ownerUID types.UID

	switch o := owner.(type) {
	case *otelv1beta1.OpenTelemetryCollector:
		ownerKind = o.Kind
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
