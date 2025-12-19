package checker

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestGetResourceEvents(t *testing.T) {
	events := []runtime.Object{
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "event1",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind: "Pod",
				Name: "test-pod",
			},
			Type:   "Normal",
			Reason: "Created",
		},
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "event2",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind: "Pod",
				Name: "test-pod",
			},
			Type:   "Warning",
			Reason: "Failed",
		},
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "event3",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind: "Pod",
				Name: "other-pod",
			},
			Type:   "Normal",
			Reason: "Started",
		},
	}

	clientset := kubefake.NewClientset(events...)

	result, err := getResourceEvents(clientset, "default", "Pod", "test-pod")
	if err != nil {
		t.Fatalf("getResourceEvents failed: %v", err)
	}

	// Fake client returns all events regardless of field selector, so we get 3
	if len(result) != 3 {
		t.Errorf("expected 3 events, got %d", len(result))
	}

	// Check event details
	foundNormal := false
	foundWarning := false
	for _, event := range result {
		if event.Type == "Normal" && event.Reason == "Created" {
			foundNormal = true
		}
		if event.Type == "Warning" && event.Reason == "Failed" {
			foundWarning = true
		}
		if event.ResourceKind != "Pod" || event.ResourceName != "test-pod" {
			t.Errorf("unexpected resource in event: %s/%s", event.ResourceKind, event.ResourceName)
		}
	}

	if !foundNormal {
		t.Error("Normal event not found")
	}
	if !foundWarning {
		t.Error("Warning event not found")
	}
}

func TestListPods(t *testing.T) {
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod2",
			Namespace: "default",
			Labels: map[string]string{
				"app": "other",
			},
		},
	}

	objects := []runtime.Object{pod1, pod2}
	dc := fake.NewSimpleDynamicClient(scheme.Scheme, objects...)

	result, err := ListPods(dc, "default", "app=test")
	if err != nil {
		t.Fatalf("ListPods failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Result should be a list
	list, err := result.ToList()
	if err != nil {
		t.Fatalf("failed to convert to list: %v", err)
	}

	if len(list.Items) != 1 {
		t.Errorf("expected 1 pod, got %d", len(list.Items))
	}

	if list.Items[0].GetName() != "pod1" {
		t.Errorf("expected pod1, got %s", list.Items[0].GetName())
	}
}

func TestChecker_CheckResource_Integration(t *testing.T) {
	replicas := int32(1)

	// Create a deployment with ReplicaSet and Pod
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-deploy",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "nginx", Image: "nginx:latest"},
					},
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas:     1,
			UpdatedReplicas:   1,
			AvailableReplicas: 1,
		},
	}

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deploy-abc",
			Namespace: "default",
			Labels: map[string]string{
				"app":               "test",
				"pod-template-hash": "abc",
			},
			CreationTimestamp: metav1.Now(),
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &replicas,
		},
		Status: appsv1.ReplicaSetStatus{
			ObservedGeneration: 1,
			Replicas:           1,
			ReadyReplicas:      1,
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels: map[string]string{
				"app":               "test",
				"pod-template-hash": "abc",
			},
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.PodSpec{
			NodeName: "node1",
			Containers: []corev1.Container{
				{Name: "nginx", Image: "nginx:latest"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "10.0.0.1",
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			},
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "nginx", Ready: true},
			},
		},
	}

	objects := []runtime.Object{deploy, rs, pod}
	clientset := kubefake.NewClientset(objects...)
	dc := fake.NewSimpleDynamicClient(scheme.Scheme, objects...)

	cfg := &Config{
		Target: types.NamespacedName{
			Namespace: "default",
			Name:      "test-deploy",
		},
		Clientset:     clientset,
		DynamicClient: dc,
	}

	checker := NewChecker(cfg, "Deployment")

	// Get the deployment as unstructured
	unstructuredDeploy, err := dc.Resource(Resources["Deployment"].GVR).
		Namespace("default").Get(context.Background(), "test-deploy", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get deployment: %v", err)
	}

	status, err := checker.checkResource(unstructuredDeploy, Resources["Deployment"])
	if err != nil {
		t.Fatalf("checkResource failed: %v", err)
	}

	// Verify deployment status
	if status.Kind != "Deployment" {
		t.Errorf("expected Deployment, got %s", status.Kind)
	}

	// Should have 1 child (ReplicaSet)
	if len(status.Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(status.Children))
	}

	// Verify ReplicaSet child
	rsStatus := status.Children[0]
	if rsStatus.Kind != "ReplicaSet" {
		t.Errorf("expected ReplicaSet, got %s", rsStatus.Kind)
	}

	// ReplicaSet should have 1 child (Pod)
	if len(rsStatus.Children) != 1 {
		t.Errorf("expected 1 pod child, got %d", len(rsStatus.Children))
	}

	// Verify Pod child
	podStatus := rsStatus.Children[0]
	if podStatus.Kind != "Pod" {
		t.Errorf("expected Pod, got %s", podStatus.Kind)
	}
}

func TestNewChecker(t *testing.T) {
	cfg := &Config{
		Target: types.NamespacedName{
			Namespace: "default",
			Name:      "test",
		},
	}

	checker := NewChecker(cfg, "Deployment")

	if checker.Config != cfg {
		t.Error("Config not set correctly")
	}

	if checker.Kind != "Deployment" {
		t.Errorf("expected Kind=Deployment, got %s", checker.Kind)
	}
}
