// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"strconv"

	"github.com/open-telemetry/opentelemetry-operator/pkg/autodetect"
	"github.com/open-telemetry/opentelemetry-operator/pkg/platform"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

func parseId(namespace *corev1.Namespace, annotation string)(int64){
	raw:=namespace.GetAnnotations()["openshift.io/sa.scc.supplemental-groups"]
	if raw == "" {
		fmt.Println("The annotation ", annotation, " is not present")
		os.Exit(1)
	}

	lowBound := strings.Split(raw, "/")[0]
	id, err :=strconv.ParseInt(lowBound, 0, 64)
	if err != nil {
		fmt.Println("It was not possible to convert the number to int64: ", lowBound)
		os.Exit(1)
	}

	return id
}

func getGroupID(namespace *v1.Namespace) (int64){
	return parseId(namespace, "openshift.io/sa.scc.supplemental-groups")
}

func getUserID(namespace *v1.Namespace) (int64){
	return parseId(namespace, "openshift.io/sa.scc.uid-range")
}


func main(){

	if len(os.Args) == 1 {
		fmt.Println("You need to provide the name of the deployment")
		os.Exit(1)
	}

	deploymentName := os.Args[1]

	restConfig := ctrl.GetConfigOrDie()
	ad, err := autodetect.New(restConfig)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	runningPlatform, err := ad.Platform()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if runningPlatform == platform.OpenShift {
		fmt.Println("Connected to an OpenShift cluster")
	} else {
		fmt.Println("Nothing extra needs to be done")
		os.Exit(0)
	}

	client, err :=kubernetes.NewForConfig(restConfig)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	namespace, err:= client.CoreV1().Namespaces().Get(context.Background(), os.Getenv("NAMESPACE"),  metav1.GetOptions{})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	deploymentsClient := client.AppsV1().Deployments(namespace.Name)
	deployment,err := deploymentsClient.Get(
		context.Background(),
		deploymentName,
		metav1.GetOptions{},
	)

	if err != nil {
		fmt.Println("Deployment was not found")
		os.Exit(1)
	}


	var userId *int64 = new(int64)
	*userId=getUserID(namespace)

	var groupdId *int64 = new(int64)
	*groupdId=getGroupID(namespace)

	deployment.Spec.Template.Spec.SecurityContext.RunAsUser = userId
	deployment.Spec.Template.Spec.SecurityContext.RunAsGroup = groupdId
	deployment.Spec.Template.Spec.SecurityContext.FSGroup = groupdId

	deploymentsClient.Update(context.Background(), deployment, metav1.UpdateOptions{})
}