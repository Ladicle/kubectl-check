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

func TestCheckDaemonSet(t *testing.T) {
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-ds",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
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
		Status: appsv1.DaemonSetStatus{
			NumberReady:            2,
			DesiredNumberScheduled: 3,
			CurrentNumberScheduled: 3,
			NumberAvailable:        2,
			ObservedGeneration:     1,
		},
	}

	clientset := kubefake.NewClientset()
	dc := fake.NewSimpleDynamicClient(scheme.Scheme)

	status, err := checkDaemonSet(ds, clientset, dc)
	if err != nil {
		t.Fatalf("checkDaemonSet failed: %v", err)
	}

	if status.Kind != "DaemonSet" {
		t.Errorf("expected Kind=DaemonSet, got %s", status.Kind)
	}

	if status.Name != "test-ds" {
		t.Errorf("expected Name=test-ds, got %s", status.Name)
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
			if field.Value != "2 Ready / 3 Current / 2 Available (of 3 desired)" {
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
