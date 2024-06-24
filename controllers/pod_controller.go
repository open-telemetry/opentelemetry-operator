package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

	if annVal, ok := pod.Annotations["inject"]; ok {

		fmt.Println("Mithun   : Annotation Inject Exists")

		if annVal == "true" {
			// Example ConfigMap name and namespace
			configMapName := "k8sobjectsconfig"
			configMapNamespace := "default"

			fmt.Println("Mithun  configname : ", configMapName)

			// Fetch the ConfigMap instance
			configMap := &corev1.ConfigMap{}
			err = r.Get(ctx, client.ObjectKey{Namespace: configMapNamespace, Name: configMapName}, configMap)
			if err != nil {
				fmt.Println(err, "Failed to get ConfigMap")
				return ctrl.Result{}, err
			}

			// Read and update the ConfigMap
			configYaml, ok := configMap.Data["config.yaml"]

			fmt.Println("mithun ", configYaml)

			if !ok {
				log.Error(fmt.Errorf("config.yaml not found in ConfigMap"), "Failed to find config.yaml in ConfigMap")
				return ctrl.Result{}, nil
			}

			// Unmarshal the YAML content into a map
			var configData map[string]interface{}
			err = yaml.Unmarshal([]byte(configYaml), &configData)
			if err != nil {
				log.Error(err, "Failed to unmarshal config.yaml")
				return ctrl.Result{}, err
			}

			// Add the new mysql section
			if receivers, ok := configData["receivers"].(map[interface{}]interface{}); ok {
				mysqlReceiver := map[interface{}]interface{}{
					"endpoint":            "127.0.0.1:5432",
					"username":            "qwertyuitretyuuerty",
					"password":            "password123456",
					"collection_interval": "10s",
				}
				receivers["mysql"] = mysqlReceiver
			} else {
				log.Error(fmt.Errorf("receivers section not found in config.yaml"), "Failed to find receivers section in config.yaml")
				return ctrl.Result{}, nil
			}

			// Marshal the updated config back to YAML
			updatedConfigYaml, err := yaml.Marshal(configData)

			if err != nil {
				log.Error(err, "Failed to marshal updated config.yaml")
				return ctrl.Result{}, err
			}

			fmt.Println("updated configmap : ", updatedConfigYaml)

			// Update the ConfigMap with the new config
			configMap.Data["config.yaml"] = string(updatedConfigYaml)
			err = r.Update(ctx, configMap)
			if err != nil {
				log.Error(err, "Failed to update ConfigMap")
				return ctrl.Result{}, err
			}

			fmt.Println("ConfigMap updated successfully")

		}
	} else {
		fmt.Println("mithun Annotations does not exists")
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
