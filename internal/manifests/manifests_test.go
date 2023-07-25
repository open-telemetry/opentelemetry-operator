package manifests

import (
	"github.com/go-logr/logr"
	"github.com/open-telemetry/opentelemetry-operator/apis/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/internal/reconcileutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

func TestBuildAll(t *testing.T) {
	var goodConfig = `receivers:
  examplereceiver:
    endpoint: "0.0.0.0:12345"
service:
  pipelines:
    metrics:
      receivers: [examplereceiver]
      exporters: [logging]
`
	one := int32(1)
	type args struct {
		params reconcileutil.Params
	}
	tests := []struct {
		name    string
		args    args
		want    []client.Object
		wantErr bool
	}{
		{
			name: "base case",
			args: args{
				params: reconcileutil.Params{
					Log: logr.Discard(),
					Instance: v1alpha1.OpenTelemetryCollector{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "test",
						},
						Spec: v1alpha1.OpenTelemetryCollectorSpec{
							Replicas: &one,
							Mode:     "deployment",
							Image:    "test",
							Config:   goodConfig,
						},
					},
					Config: config.New(),
				},
			},
			want: []client.Object{
				&appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-collector",
						Namespace: "test",
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: &one,
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app.kubernetes.io/managed-by": "opentelemetry-operator",
								"app.kubernetes.io/part-of":    "opentelemetry",
								"app.kubernetes.io/component":  "opentelemetry-collector",
								"app.kubernetes.io/instance":   "test.test",
							},
						},
						Template:                corev1.PodTemplateSpec{
							Spec:
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildAll(tt.args.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildAll() got = %v, want %v", got, tt.want)
			}
		})
	}
}
