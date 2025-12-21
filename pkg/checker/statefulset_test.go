package checker

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestCheckStatefulSet(t *testing.T) {
	replicas := int32(3)
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-sts",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: "test-service",
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
			},
		},
		Status: appsv1.StatefulSetStatus{
			ReadyReplicas:   2,
			UpdatedReplicas: 2,
			Replicas:        3,
			UpdateRevision:  "test-sts-abc123",
			CurrentRevision: "test-sts-abc123",
		},
	}

	clientset := kubefake.NewClientset()
	dc := fake.NewSimpleDynamicClient(scheme.Scheme)

	status, err := checkStatefulSet(sts, clientset, dc)
	if err != nil {
		t.Fatalf("checkStatefulSet failed: %v", err)
	}

	if status.Kind != "StatefulSet" {
		t.Errorf("expected Kind=StatefulSet, got %s", status.Kind)
	}

	if status.Name != "test-sts" {
		t.Errorf("expected Name=test-sts, got %s", status.Name)
	}

	if status.Namespace != "default" {
		t.Errorf("expected Namespace=default, got %s", status.Namespace)
	}

	// Check fields
	foundPodsStatus := false
	foundImages := false
	foundAge := false
	for _, field := range status.Fields {
		if field.Key == "Pods Status:" {
			foundPodsStatus = true
			if field.Value != "2 Ready / 2 Updated (of 3 desired)" {
				t.Errorf("unexpected Pods Status: %s", field.Value)
			}
		}
		if field.Key == "Images:     " {
			foundImages = true
			if field.Value != "nginx:latest" {
				t.Errorf("unexpected Images: %s", field.Value)
			}
		}
		if field.Key == "Age:        " {
			foundAge = true
		}
	}
	if !foundPodsStatus {
		t.Error("Pods Status field not found")
	}
	if !foundImages {
		t.Error("Images field not found")
	}
	if !foundAge {
		t.Error("Age field not found")
	}
}
