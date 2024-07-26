package cluster

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/cmd/gather/config"
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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Cluster struct {
	config *config.Config
}

func NewCluster(cfg *config.Config) Cluster {
	return Cluster{config: cfg}
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
		LabelSelector: labels.SelectorFromSet(labels.Set{
			"app.kubernetes.io/name": "opentelemetry-operator",
		}),
	})

	if err != nil {
		return appsv1.Deployment{}, err
	}

	if len(operatorDeployments.Items) == 0 {
		return appsv1.Deployment{}, fmt.Errorf("operator not found")
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

func (c *Cluster) processOwnedResources(otelCol *otelv1beta1.OpenTelemetryCollector, folder string) error {
	errorDetected := false

	////////////////////////////////////////////////////////////////// apps/v1
	// DaemonSets
	daemonsets, err := c.getOwnerResources(&appsv1.DaemonSetList{}, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, d := range daemonsets {
		writeToFile(folder, d)
	}

	// Deployments
	deployments, err := c.getOwnerResources(&appsv1.DeploymentList{}, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, d := range deployments {
		writeToFile(folder, d)
	}

	// StatefulSets
	statefulsets, err := c.getOwnerResources(&appsv1.StatefulSetList{}, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, s := range statefulsets {
		writeToFile(folder, s)
	}

	////////////////////////////////////////////////////////////////// rbac/v1
	// ClusterRole
	crs, err := c.getOwnerResources(&rbacv1.ClusterRoleList{}, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, cr := range crs {
		writeToFile(folder, cr)
	}

	// ClusterRoleBindings
	crbs, err := c.getOwnerResources(&rbacv1.ClusterRoleBindingList{}, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, crb := range crbs {
		writeToFile(folder, crb)
	}

	////////////////////////////////////////////////////////////////// core/v1
	// ConfigMaps
	cms, err := c.getOwnerResources(&corev1.ConfigMapList{}, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, c := range cms {
		writeToFile(folder, c)
	}

	// PersistentVolumes
	pvs, err := c.getOwnerResources(&corev1.PersistentVolumeList{}, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, p := range pvs {
		writeToFile(folder, p)
	}

	// PersistentVolumeClaims
	pvcs, err := c.getOwnerResources(&corev1.PersistentVolumeClaimList{}, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, p := range pvcs {
		writeToFile(folder, p)
	}

	// Pods
	pods, err := c.getOwnerResources(&corev1.PodList{}, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, p := range pods {
		writeToFile(folder, p)
	}

	// Services
	services, err := c.getOwnerResources(&corev1.ServiceList{}, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, s := range services {
		writeToFile(folder, s)
	}

	// ServiceAccounts
	sas, err := c.getOwnerResources(&corev1.ServiceAccountList{}, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, s := range sas {
		writeToFile(folder, s)
	}

	////////////////////////////////////////////////////////////////// autoscaling/v2
	// HPAs
	hpas, err := c.getOwnerResources(&autoscalingv2.HorizontalPodAutoscalerList{}, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, h := range hpas {
		writeToFile(folder, h)
	}

	////////////////////////////////////////////////////////////////// networking/v1
	// Ingresses
	ingresses, err := c.getOwnerResources(&networkingv1.IngressList{}, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, i := range ingresses {
		writeToFile(folder, i)
	}

	////////////////////////////////////////////////////////////////// policy/v1
	// PodDisruptionBudge
	pdbs, err := c.getOwnerResources(&policy1.PodDisruptionBudgetList{}, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, pdb := range pdbs {
		writeToFile(folder, pdb)
	}

	////////////////////////////////////////////////////////////////// monitoring/v1
	if c.isAPIAvailable(schema.GroupVersionResource{
		Group:    monitoringv1.SchemeGroupVersion.Group,
		Version:  monitoringv1.SchemeGroupVersion.Version,
		Resource: "ServiceMonitor",
	}) {
		// PodMonitors
		pms, err := c.getOwnerResources(&monitoringv1.PodMonitorList{}, otelCol)
		if err != nil {
			errorDetected = true
			log.Fatalln(err)
		}
		for _, pm := range pms {
			writeToFile(folder, pm)
		}

		// ServiceMonitors
		sms, err := c.getOwnerResources(&monitoringv1.ServiceMonitorList{}, otelCol)
		if err != nil {
			errorDetected = true
			log.Fatalln(err)
		}
		for _, s := range sms {
			writeToFile(folder, s)
		}
	}

	////////////////////////////////////////////////////////////////// route/v1
	// Routes
	if c.isAPIAvailable(schema.GroupVersionResource{
		Group:    routev1.GroupName,
		Version:  routev1.GroupVersion.Version,
		Resource: "Route",
	}) {
		rs, err := c.getOwnerResources(&routev1.RouteList{}, otelCol)
		if err != nil {
			errorDetected = true
			log.Fatalln(err)
		}
		for _, r := range rs {
			writeToFile(folder, r)
		}
	}

	if errorDetected {
		return fmt.Errorf("something failed while getting the associated resources")
	}

	return nil
}

func (c *Cluster) getOwnerResources(objList client.ObjectList, otelCol *otelv1beta1.OpenTelemetryCollector) ([]client.Object, error) {
	err := c.config.KubernetesClient.List(context.TODO(), objList, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
		}),
	})
	if err != nil {
		return nil, err
	}

	resources := []client.Object{}

	items := reflect.ValueOf(objList).Elem().FieldByName("Items")
	for i := 0; i < items.Len(); i++ {
		item := items.Index(i).Addr().Interface().(client.Object)
		if hasOwnerReference(item, otelCol) {
			resources = append(resources, item)
		}
	}
	return resources, nil

}

func (c *Cluster) isAPIAvailable(gvr schema.GroupVersionResource) bool {
	rm := c.config.KubernetesClient.RESTMapper()

	gvk, err := rm.KindFor(gvr)
	if err != nil {
		return false
	}

	return !gvk.Empty()
}

func hasOwnerReference(obj client.Object, otelCol *otelv1beta1.OpenTelemetryCollector) bool {
	for _, ownerRef := range obj.GetOwnerReferences() {
		if ownerRef.Kind == otelCol.Kind && ownerRef.UID == otelCol.UID {
			return true
		}
	}
	return false
}
