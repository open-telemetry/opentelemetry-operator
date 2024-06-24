package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	scheme *runtime.Scheme
	log    logr.Logger
	config config.Config
}

// PodReconcilerParams is the set of options to build a new PodReconciler.
type PodReconcilerParams struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
	Config config.Config
}

// var PodConfigCache map[string]string

// func init() {
// 	PodConfigCache = make(map[string]string)
// }

// NewPodReconciler creates a new pod reconciler for pod objects.
func NewPodReconciler(p PodReconcilerParams) *PodReconciler {
	r := &PodReconciler{
		Client: p.Client,
		log:    p.Log,
		scheme: p.Scheme,
		config: p.Config,
	}
	return r
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Pod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.

// Pod Addition/Creation
//1. Check if pod has inject annotation. Record the annValue.
//2. Check if pod is in running state
//3. Query OpenTelemetry CRO with the namespace/name equal to annValue.
//4. Get the spec.ConfigMap name of opsramp agent configmap. Currently hard code.
//5. Query the configMap from the 4th step and unmarshal the yaml. Name it as Target Config Map.
//6. Read the Receiver, Exporters, Processors from CRO spec section and add suffix pod name to each rx, px, ex and then add it to target config map.
//7. Update configmap
//8. Maintain a cache of all the pod name to configMap that we have taken action.

// Pod Deletion
//1. Check if the pod deleted and if it is in the cache and get the corresponding configMap name from cache.
//2. Query ConfigMap, remove all the rx, px, ex data related to this pod.
//3. update configmap.

func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.log.WithValues("opentelemetrypodcontroller", req.NamespacedName)

	// Fetch the Pod instance
	pod := &corev1.Pod{}
	err := r.Get(ctx, req.NamespacedName, pod)
	if err != nil {
		if errors.IsNotFound(err) {
			// Pod not found. Return and don't requeue
			log.Info("Pod resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Pod")
		return ctrl.Result{}, err
	}

	//fmt.Println("Mithun", pod)

	//1. Check if pod has inject annotation. Record the annValue.
	if annVal, ok := pod.Annotations["inject"]; ok {
		fmt.Println("Mithun   : Annotation Inject Exists")

		//2. Check if pod is in running state
		if pod.Status.Phase == "Running" {

			//3. Query OpenTelemetry CRO with the namespace/name equal to annValue.'
			cro := strings.Split(annVal, "/")
			if len(cro) == 1 {
				fmt.Println("failed to split the annotation value : ", annVal)
			} else {
				croNamespace := cro[0]
				croName := cro[1]
				fmt.Println("cro-namespace : ", croNamespace, "cro-name : ", croName)

				//4. Get the spec.ConfigMap name of opsramp agent configmap. Currently hard code.
				configMapName := "k8sobjectsconfig"
				configMapNamespace := "default"

				//5. Query the configMap from the 4th step and unmarshal the yaml. Name it as Target Config Map.
				configMap := &corev1.ConfigMap{}
				err = r.Get(ctx, client.ObjectKey{Namespace: configMapNamespace, Name: configMapName}, configMap)
				if err != nil {
					fmt.Println(err, "Failed to get ConfigMap")
					return ctrl.Result{}, err
				}
				// Read and update the ConfigMap
				targetConfig, ok := configMap.Data["config.yaml"]
				if !ok {
					log.Error(fmt.Errorf("config.yaml not found in ConfigMap"), "Failed to find config.yaml in ConfigMap")
					return ctrl.Result{}, nil
				}

				fmt.Println("mithun Existing configmap ", targetConfig)
				// Unmarshal the YAML content into a map
				var targetConfigMap map[string]interface{}
				err = yaml.Unmarshal([]byte(targetConfig), &targetConfigMap)
				if err != nil {
					log.Error(err, "Failed to unmarshal config.yaml")
					return ctrl.Result{}, err
				}

				//6. Read the Receiver, Exporters, Processors from CRO spec section and
				// add suffix pod name to each rx, px, ex and then add it to target config map.

				var instance v1beta1.OpenTelemetryCollector
				if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
					if !apierrors.IsNotFound(err) {
						log.Error(err, "unable to fetch OpenTelemetryCollector")
					}

					// we'll ignore not-found errors, since they can't be fixed by an immediate
					// requeue (we'll need to wait for a new notification), and we can get them
					// on deleted requests.
					return ctrl.Result{}, client.IgnoreNotFound(err)
				}

				if instance.Spec.Config.Receivers.Object != nil {
					// Marshal the Receivers object to JSON
					receiversJSON, err := json.Marshal(instance.Spec.Config.Receivers.Object)
					if err != nil {
						log.Error(err, "failed to marshal receivers object to JSON")
						return ctrl.Result{}, err
					}

					// Unmarshal JSON into a map[string]interface{} for easier manipulation
					var receiversMap map[string]interface{}
					err = json.Unmarshal(receiversJSON, &receiversMap)
					if err != nil {
						log.Error(err, "failed to unmarshal JSON to receivers map")
						return ctrl.Result{}, err
					}

					// Iterate over each receiver and update the target ConfigMap
					for key, value := range receiversMap {
						newKey := fmt.Sprintf("%s/%s", key, pod.Status.PodIP)
						targetConfigMap[newKey] = fmt.Sprintf("%v", value)
						log.Info(fmt.Sprintf("Updated target ConfigMap with key '%s'", newKey))
					}
				} else {
					log.Info("There is no Receiver section exists")
				}

				// Processor Section Update
				if instance.Spec.Config.Processors.Object != nil {
					// Marshal the Processors object to JSON
					processorJSON, err := json.Marshal(instance.Spec.Config.Processors.Object)
					if err != nil {
						log.Error(err, "failed to marshal processor object to JSON")
						return ctrl.Result{}, err
					}

					// Unmarshal JSON into a map[string]interface{} for easier manipulation
					var processorMap map[string]interface{}
					err = json.Unmarshal(processorJSON, &processorMap)
					if err != nil {
						log.Error(err, "failed to unmarshal JSON to processor map")
						return ctrl.Result{}, err
					}

					// Iterate over each receiver and update the target ConfigMap
					for key, value := range processorMap {
						newKey := fmt.Sprintf("%s/%s", key, pod.Status.PodIP)
						targetConfigMap[newKey] = fmt.Sprintf("%v", value)
						log.Info(fmt.Sprintf("Updated target ConfigMap with key '%s'", newKey))
					}
				} else {
					log.Info("There is no Processor section exists")
				}

				// Exporter Section Update
				if instance.Spec.Config.Exporters.Object != nil {
					// Marshal the Exporter object to JSON
					exporterJSON, err := json.Marshal(instance.Spec.Config.Exporters.Object)
					if err != nil {
						log.Error(err, "failed to marshal exporters object to JSON")
						return ctrl.Result{}, err
					}

					// Unmarshal JSON into a map[string]interface{} for easier manipulation
					var exporterMap map[string]interface{}
					err = json.Unmarshal(exporterJSON, &exporterMap)
					if err != nil {
						log.Error(err, "failed to unmarshal JSON to exporter map")
						return ctrl.Result{}, err
					}

					// Iterate over each receiver and update the target ConfigMap
					for key, value := range exporterMap {
						newKey := fmt.Sprintf("%s/%s", key, pod.Status.PodIP)
						targetConfigMap[newKey] = fmt.Sprintf("%v", value)
						log.Info(fmt.Sprintf("Updated target ConfigMap with key '%s'", newKey))
					}
				} else {
					log.Info("There is no exporter section exists")
				}

				targetConfigYaml, err := yaml.Marshal(targetConfigMap)
				if err != nil {
					log.Error(err, "Failed to marshal updated config.yaml")
					return ctrl.Result{}, err
				}

				fmt.Println("updated configmap : ", string(targetConfigYaml))

				// 7. Update the ConfigMap with the new config
				configMap.Data["config.yaml"] = string(targetConfigYaml)
				err = r.Update(ctx, configMap)
				if err != nil {
					log.Error(err, "Failed to update ConfigMap")
					return ctrl.Result{}, err
				}
				fmt.Println("ConfigMap updated successfully")

				//8. Maintain a cache of all the pod name to configMap that we have taken action.

			}
		}

	} else {
		fmt.Println("Annotations Inject does not exists ")
	}

	// Return and requeue after a specified duration
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}
