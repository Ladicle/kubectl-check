package checker

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestCheckDeployment(t *testing.T) {
	replicas := int32(3)
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-deploy",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
						{
							Name:  "sidecar",
							Image: "busybox:latest",
						},
					},
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas:     2,
			UpdatedReplicas:   3,
			AvailableReplicas: 2,
		},
	}

	clientset := kubefake.NewClientset()
	dc := fake.NewSimpleDynamicClient(scheme.Scheme)

	status, err := checkDeployment(deploy, clientset, dc)
	if err != nil {
		t.Fatalf("checkDeployment failed: %v", err)
	}

	if status.Kind != "Deployment" {
		t.Errorf("expected Kind=Deployment, got %s", status.Kind)
	}

	if status.Name != "test-deploy" {
		t.Errorf("expected Name=test-deploy, got %s", status.Name)
	}

	if status.Namespace != "default" {
		t.Errorf("expected Namespace=default, got %s", status.Namespace)
	}

	// Check fields
	foundPodsStatus := false
	foundImages := false
	for _, field := range status.Fields {
		if field.Key == "Pods Status:" {
			foundPodsStatus = true
			if field.Value != "2 Ready / 3 Up-to-date / 2 Available (of 3 desired)" {
				t.Errorf("unexpected Pods Status: %s", field.Value)
			}
		}
		if field.Key == "Images:     " {
			foundImages = true
			if field.Value != "nginx:latest, busybox:latest" {
				t.Errorf("unexpected Images: %s", field.Value)
			}
		}
	}

	if !foundPodsStatus {
		t.Error("Pods Status field not found")
	}

	if !foundImages {
		t.Error("Images field not found")
	}
}

func TestListDeploymentChild(t *testing.T) {
	replicas := int32(2)
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deploy",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
		},
	}

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deploy-12345",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Status: appsv1.ReplicaSetStatus{
			ObservedGeneration: 1,
			Replicas:           replicas,
		},
	}

	objects := []runtime.Object{rs}
	dc := fake.NewSimpleDynamicClient(scheme.Scheme, objects...)

	result, err := listDeploymentChild(deploy, dc)
	if err != nil {
		t.Fatalf("listDeploymentChild failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	kind := result.GetKind()
	if kind != "ReplicaSet" {
		t.Errorf("expected Kind=ReplicaSet, got %s", kind)
	}

	name := result.GetName()
	if name != "test-deploy-12345" {
		t.Errorf("expected Name=test-deploy-12345, got %s", name)
	}
}

func TestListDeploymentChild_NoReplicaSet(t *testing.T) {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deploy",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
		},
	}

	dc := fake.NewSimpleDynamicClient(scheme.Scheme)

	_, err := listDeploymentChild(deploy, dc)
	if err == nil {
		t.Error("expected error for no ReplicaSet, got nil")
	}

	if err.Error() != "not found ReplicaSet" {
		t.Errorf("unexpected error: %v", err)
	}
}
