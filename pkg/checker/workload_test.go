package checker

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	kueuev1beta1 "sigs.k8s.io/kueue/apis/kueue/v1beta1"
)

func TestCheckWorkload(t *testing.T) {
	// Register Kueue types
	_ = kueuev1beta1.AddToScheme(scheme.Scheme)

	workload := &kueuev1beta1.Workload{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-workload",
			Namespace: "default",
		},
		Spec: kueuev1beta1.WorkloadSpec{
			QueueName: "user-queue",
		},
		Status: kueuev1beta1.WorkloadStatus{
			Admission: &kueuev1beta1.Admission{
				ClusterQueue: "cluster-queue",
			},
			Conditions: []metav1.Condition{
				{
					Type:   kueuev1beta1.WorkloadAdmitted,
					Status: metav1.ConditionTrue,
				},
			},
		},
	}

	clientset := kubefake.NewClientset()
	dc := fake.NewSimpleDynamicClient(scheme.Scheme)

	status, err := checkWorkload(workload, clientset, dc)
	if err != nil {
		t.Fatalf("checkWorkload failed: %v", err)
	}

	if status.Kind != "Workload" {
		t.Errorf("expected Kind=Workload, got %s", status.Kind)
	}

	if status.Name != "test-workload" {
		t.Errorf("expected Name=test-workload, got %s", status.Name)
	}

	// Check that fields are present
	foundQueue := false
	foundStatus := false
	foundClusterQueue := false
	foundConditions := false

	for _, field := range status.Fields {
		switch field.Key {
		case "Queue:           ":
			foundQueue = true
			if field.Value != "user-queue" {
				t.Errorf("expected Queue=user-queue, got %s", field.Value)
			}
		case "Status:          ":
			foundStatus = true
			if field.Value != "Admitted" {
				t.Errorf("expected Status=Admitted, got %s", field.Value)
			}
		case "Cluster Queue:   ":
			foundClusterQueue = true
			if field.Value != "cluster-queue" {
				t.Errorf("expected ClusterQueue=cluster-queue, got %s", field.Value)
			}
		case "Conditions:      ":
			foundConditions = true
			if field.Value != "Admitted" {
				t.Errorf("expected Conditions=Admitted, got %s", field.Value)
			}
		}
	}

	if !foundQueue {
		t.Error("Queue field not found")
	}
	if !foundStatus {
		t.Error("Status field not found")
	}
	if !foundClusterQueue {
		t.Error("ClusterQueue field not found")
	}
	if !foundConditions {
		t.Error("Conditions field not found")
	}
}

func TestCheckWorkload_Pending(t *testing.T) {
	// Register Kueue types
	_ = kueuev1beta1.AddToScheme(scheme.Scheme)

	workload := &kueuev1beta1.Workload{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pending-workload",
			Namespace: "default",
		},
		Spec: kueuev1beta1.WorkloadSpec{
			QueueName: "user-queue",
		},
		Status: kueuev1beta1.WorkloadStatus{
			// No admission - pending state
			Admission: nil,
		},
	}

	clientset := kubefake.NewClientset()
	dc := fake.NewSimpleDynamicClient(scheme.Scheme)

	status, err := checkWorkload(workload, clientset, dc)
	if err != nil {
		t.Fatalf("checkWorkload failed: %v", err)
	}

	// Check that status is Pending
	foundStatus := false
	for _, field := range status.Fields {
		if field.Key == "Status:          " {
			foundStatus = true
			if field.Value != "Pending" {
				t.Errorf("expected Status=Pending, got %s", field.Value)
			}
		}
	}

	if !foundStatus {
		t.Error("Status field not found")
	}
}

func TestListWorkloadChild(t *testing.T) {
	// Register Kueue types
	_ = kueuev1beta1.AddToScheme(scheme.Scheme)

	workload := &kueuev1beta1.Workload{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-workload",
			Namespace: "default",
		},
	}

	// Create pods with workload annotation
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "default",
			Annotations: map[string]string{
				"kueue.x-k8s.io/workload": "test-workload",
			},
		},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod2",
			Namespace: "default",
			Annotations: map[string]string{
				"kueue.x-k8s.io/workload": "test-workload",
			},
		},
	}

	pod3 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod3",
			Namespace: "default",
			Annotations: map[string]string{
				"kueue.x-k8s.io/workload": "other-workload",
			},
		},
	}

	pod4 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod4",
			Namespace: "default",
			// No workload annotation
		},
	}

	unstructuredPod1, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(pod1)
	unstructuredPod2, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(pod2)
	unstructuredPod3, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(pod3)
	unstructuredPod4, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(pod4)

	dc := fake.NewSimpleDynamicClient(scheme.Scheme, pod1, pod2, pod3, pod4)

	result, err := listWorkloadChild(workload, dc)
	if err != nil {
		t.Fatalf("listWorkloadChild failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Check that result is a list
	if result.Object["kind"] != "List" {
		t.Errorf("expected kind=List, got %v", result.Object["kind"])
	}

	// Get items from result
	items, ok := result.Object["items"].([]unstructured.Unstructured)
	if !ok {
		t.Fatalf("failed to get items from result")
	}

	if len(items) != 2 {
		t.Errorf("expected 2 pods, got %d", len(items))
	}

	// Verify the pods are the correct ones
	foundPod1 := false
	foundPod2 := false
	for _, item := range items {
		name := item.GetName()

		if name == "pod1" {
			foundPod1 = true
		}
		if name == "pod2" {
			foundPod2 = true
		}
	}

	if !foundPod1 {
		t.Error("pod1 not found in result")
	}
	if !foundPod2 {
		t.Error("pod2 not found in result")
	}

	_ = unstructuredPod1
	_ = unstructuredPod2
	_ = unstructuredPod3
	_ = unstructuredPod4
}

func TestListWorkloadChild_NoPods(t *testing.T) {
	// Register Kueue types
	_ = kueuev1beta1.AddToScheme(scheme.Scheme)

	workload := &kueuev1beta1.Workload{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-workload",
			Namespace: "default",
		},
	}

	dc := fake.NewSimpleDynamicClient(scheme.Scheme)

	result, err := listWorkloadChild(workload, dc)
	if err != nil {
		t.Fatalf("listWorkloadChild failed: %v", err)
	}

	if result != nil {
		t.Error("expected nil result when no matching pods")
	}
}

func TestFormatConditionList(t *testing.T) {
	tests := []struct {
		name       string
		conditions []string
		expected   string
	}{
		{
			name:       "single condition",
			conditions: []string{"Admitted"},
			expected:   "Admitted",
		},
		{
			name:       "multiple conditions",
			conditions: []string{"Admitted", "QuotaReserved"},
			expected:   "Admitted, QuotaReserved",
		},
		{
			name:       "empty conditions",
			conditions: []string{},
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatConditionList(tt.conditions)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
