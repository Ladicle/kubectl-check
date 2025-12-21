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

func TestCheckReplicaSet(t *testing.T) {
	replicas := int32(3)
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-rs",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
			Labels: map[string]string{
				appsv1.DefaultDeploymentUniqueLabelKey: "abc123",
			},
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &replicas,
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
		Status: appsv1.ReplicaSetStatus{
			ReadyReplicas: 2,
			Replicas:      3,
		},
	}

	clientset := kubefake.NewClientset()
	dc := fake.NewSimpleDynamicClient(scheme.Scheme)

	status, err := checkReplicaSet(rs, clientset, dc)
	if err != nil {
		t.Fatalf("checkReplicaSet failed: %v", err)
	}

	if status.Kind != "ReplicaSet" {
		t.Errorf("expected Kind=ReplicaSet, got %s", status.Kind)
	}

	if status.Name != "test-rs" {
		t.Errorf("expected Name=test-rs, got %s", status.Name)
	}

	if status.Namespace != "default" {
		t.Errorf("expected Namespace=default, got %s", status.Namespace)
	}

	// Check fields
	foundPodsStatus := false
	foundAge := false
	for _, field := range status.Fields {
		if field.Key == "Pods Status:" {
			foundPodsStatus = true
			if field.Value != "2 Ready / 3 Current (of 3 desired)" {
				t.Errorf("unexpected Pods Status: %s", field.Value)
			}
		}
		if field.Key == "Age:        " {
			foundAge = true
		}
	}
	if !foundPodsStatus {
		t.Error("Pods Status field not found")
	}
	if !foundAge {
		t.Error("Age field not found")
	}
}
