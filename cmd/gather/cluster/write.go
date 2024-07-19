package cluster

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/open-telemetry/opentelemetry-operator/apis/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createOTELFolder(collectionDir string, otelCol *v1beta1.OpenTelemetryCollector) (string, error) {
	outputDir := filepath.Join(collectionDir, "namespaces", otelCol.Namespace, otelCol.Name)
	err := os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		return "", err
	}
	return outputDir, nil
}

func createFile(outputDir string, obj client.Object) (*os.File, error) {
	kind := obj.GetObjectKind().GroupVersionKind().Kind

	if kind == "" {
		// reflect.TypeOf(obj) will return something like *v1.Deployment. We remove the first part
		prefix, typeName, found := strings.Cut(reflect.TypeOf(obj).String(), ".")
		if found {
			kind = typeName
		} else {
			kind = prefix
		}
	}

	kind = strings.ToLower(kind)
	name := strings.ReplaceAll(obj.GetName(), ".", "-")

	path := filepath.Join(outputDir, fmt.Sprintf("%s-%s.yaml", kind, name))
	return os.Create(path)
}

func writeToFile(outputDir string, o client.Object) {
	// Open or create the file for writing
	outputFile, err := createFile(outputDir, o)
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer outputFile.Close()

	unstructuredDeployment, err := runtime.DefaultUnstructuredConverter.ToUnstructured(o)
	if err != nil {
		log.Fatalf("Error converting deployment to unstructured: %v", err)
	}

	unstructuredObj := &unstructured.Unstructured{Object: unstructuredDeployment}

	// Serialize the unstructured object to YAML
	serializer := json.NewYAMLSerializer(json.DefaultMetaFactory, nil, nil)
	err = serializer.Encode(unstructuredObj, outputFile)
	if err != nil {
		log.Fatalf("Error encoding to YAML: %v", err)
	}
}
