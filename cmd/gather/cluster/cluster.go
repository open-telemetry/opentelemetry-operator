package cluster

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	otelv1alpha1 "github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	otelv1beta1 "github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/cmd/gather/config"
	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policy1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Cluster struct {
	config *config.Config
}

func NewCluster(cfg *config.Config) Cluster {
	return Cluster{config: cfg}
}

func (c *Cluster) GetOpenTelemetryCollectors() error {
	otelCols := otelv1beta1.OpenTelemetryCollectorList{}

	err := c.config.KubernetesClient.List(context.TODO(), &otelCols, &client.ListOptions{})
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

	err := c.config.KubernetesClient.List(context.TODO(), &instrumentations, &client.ListOptions{})
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

		if err != nil {

		}
	}

	if errorDetected {
		return fmt.Errorf("something failed while getting the opentelemtrycollectors")
	}
	return nil
}

func (c *Cluster) processOTELCollector(otelCol *otelv1beta1.OpenTelemetryCollector) error {
	log.Printf("Processing OpenTelemetryCollector %s/%s", otelCol.Namespace, otelCol.Name)
	folder, err := createFolder(c.config.CollectionDir, otelCol)
	if err != nil {
		return err
	}
	writeToFile(folder, otelCol)

	err = c.processOwnedResources(otelCol)
	if err != nil {
		return err
	}

	return nil
}

func (c *Cluster) processOwnedResources(otelCol *otelv1beta1.OpenTelemetryCollector) error {
	folder, err := createFolder(c.config.CollectionDir, otelCol)
	if err != nil {
		return err
	}
	errorDetected := false

	// ClusterRole
	crs := rbacv1.ClusterRoleList{}
	err = c.getOwnerResources(&crs, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, cr := range crs.Items {
		writeToFile(folder, &cr)
	}

	// ClusterRoleBindings
	crbs := rbacv1.ClusterRoleBindingList{}
	err = c.getOwnerResources(&crbs, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, crb := range crbs.Items {
		writeToFile(folder, &crb)
	}

	// ConfigMaps
	cms := corev1.ConfigMapList{}
	err = c.getOwnerResources(&cms, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, c := range cms.Items {
		writeToFile(folder, &c)
	}

	// DaemonSets
	daemonsets := appsv1.DaemonSetList{}
	err = c.getOwnerResources(&daemonsets, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, d := range daemonsets.Items {
		writeToFile(folder, &d)
	}

	// Deployments
	deployments := appsv1.DeploymentList{}
	err = c.getOwnerResources(&deployments, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, d := range deployments.Items {
		writeToFile(folder, &d)
	}

	// HPAs
	hpas := autoscalingv2.HorizontalPodAutoscalerList{}
	err = c.getOwnerResources(&hpas, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, h := range hpas.Items {
		writeToFile(folder, &h)
	}

	// Ingresses
	ingresses := networkingv1.IngressList{}
	err = c.getOwnerResources(&ingresses, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, i := range ingresses.Items {
		writeToFile(folder, &i)
	}

	// PersistentVolumes
	pvs := corev1.PersistentVolumeList{}
	err = c.getOwnerResources(&pvs, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, p := range pvs.Items {
		writeToFile(folder, &p)
	}

	// PersistentVolumeClaims
	pvcs := corev1.PersistentVolumeClaimList{}
	err = c.getOwnerResources(&pvcs, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, p := range pvcs.Items {
		writeToFile(folder, &p)
	}

	// PodDisruptionBudget
	pdbs := policy1.PodDisruptionBudgetList{}
	err = c.getOwnerResources(&pdbs, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, pdb := range pdbs.Items {
		writeToFile(folder, &pdb)
	}

	// PodMonitors
	pms := monitoringv1.PodMonitorList{}
	err = c.getOwnerResources(&pms, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, pm := range pms.Items {
		writeToFile(folder, pm)
	}

	// Routes
	rs := routev1.RouteList{}
	err = c.getOwnerResources(&rs, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, r := range rs.Items {
		writeToFile(folder, &r)
	}

	// Services
	services := corev1.ServiceList{}
	err = c.getOwnerResources(&services, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, s := range services.Items {
		writeToFile(folder, &s)
	}

	// ServiceMonitors
	sms := monitoringv1.ServiceMonitorList{}
	err = c.getOwnerResources(&sms, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, s := range sms.Items {
		writeToFile(folder, s)
	}

	// ServiceAccounts
	sas := corev1.ServiceAccountList{}
	err = c.getOwnerResources(&sas, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, s := range sas.Items {
		writeToFile(folder, &s)
	}

	// StatefulSets
	statefulsets := appsv1.StatefulSetList{}
	err = c.getOwnerResources(&statefulsets, otelCol)
	if err != nil {
		errorDetected = true
		log.Fatalln(err)
	}
	for _, s := range statefulsets.Items {
		writeToFile(folder, &s)
	}

	if errorDetected {
		return fmt.Errorf("something failed while getting the associated resources")
	}

	return nil
}

func (c *Cluster) getOwnerResources(objList client.ObjectList, otelCol *otelv1beta1.OpenTelemetryCollector) error {
	return c.config.KubernetesClient.List(context.TODO(), objList, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{
			"app.kubernetes.io/instance":   fmt.Sprintf("%s.%s", otelCol.Namespace, otelCol.Name),
			"app.kubernetes.io/managed-by": "opentelemetry-operator",
			"app.kubernetes.io/part-of":    "opentelemetry",
		}),
	})
}

func hasOwnerReference(obj client.Object, otelCol *otelv1beta1.OpenTelemetryCollector) bool {
	for _, ownerRef := range obj.GetOwnerReferences() {
		if ownerRef.Kind == otelCol.Kind && ownerRef.UID == otelCol.UID {
			return true
		}
	}
	return false
}
