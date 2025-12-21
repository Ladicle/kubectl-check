package checker

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"

	kueuev1beta1 "sigs.k8s.io/kueue/apis/kueue/v1beta1"
)

// TestGroupPodsByWorkload tests the grouping of pods by workload annotation
func TestGroupPodsByWorkload(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = kueuev1beta1.AddToScheme(scheme)

	// Create a workload
	workload := &kueuev1beta1.Workload{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kueue.x-k8s.io/v1beta1",
			Kind:       "Workload",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-workload",
			Namespace: "default",
		},
		Spec: kueuev1beta1.WorkloadSpec{
			QueueName: "test-queue",
		},
		Status: kueuev1beta1.WorkloadStatus{
			Admission: &kueuev1beta1.Admission{
				ClusterQueue: "test-cluster-queue",
			},
		},
	}

	// Create pods with and without workload annotation
	pod1 := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-with-workload-1",
			Namespace: "default",
			Annotations: map[string]string{
				"kueue.x-k8s.io/workload": "test-workload",
			},
		},
	}
	pod2 := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-with-workload-2",
			Namespace: "default",
			Annotations: map[string]string{
				"kueue.x-k8s.io/workload": "test-workload",
			},
		},
	}
	pod3 := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-without-workload",
			Namespace: "default",
		},
	}

	// Convert to unstructured
	workloadUnstructured := &unstructured.Unstructured{}
	workloadMap, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(workload)
	workloadUnstructured.Object = workloadMap

	pod1Map, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(pod1)
	pod2Map, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(pod2)
	pod3Map, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(pod3)

	// Create dynamic client with workload
	dynamicClient := fake.NewSimpleDynamicClient(scheme, workloadUnstructured)

	// Create pod list
	podList := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "List",
			"items": []interface{}{
				pod1Map,
				pod2Map,
				pod3Map,
			},
		},
	}

	// Group pods by workload
	result, err := GroupPodsByWorkload(dynamicClient, podList, "default", nil)
	if err != nil {
		t.Fatalf("GroupPodsByWorkload failed: %v", err)
	}

	// Verify result structure
	items, found, err := unstructured.NestedSlice(result.Object, "items")
	if err != nil || !found {
		t.Fatalf("Failed to get items from result: err=%v, found=%v", err, found)
	}

	// Should have 2 items: 1 workload (with 2 child pods) and 1 ungrouped pod
	if len(items) != 2 {
		t.Fatalf("Expected 2 items (1 workload, 1 pod), got %d", len(items))
	}

	// First item should be the workload
	firstItem := items[0].(map[string]interface{})
	kind, _, _ := unstructured.NestedString(firstItem, "kind")
	if kind != "Workload" {
		t.Errorf("Expected first item to be Workload, got %s", kind)
	}

	// Workload should have _childPods field with 2 pods
	childPods, found, _ := unstructured.NestedSlice(firstItem, "_childPods")
	if !found {
		t.Error("Workload should have _childPods field")
	}
	if len(childPods) != 2 {
		t.Errorf("Expected 2 child pods in workload, got %d", len(childPods))
	}

	// Second item should be the ungrouped pod
	secondItem := items[1].(map[string]interface{})
	secondKind, _, _ := unstructured.NestedString(secondItem, "kind")
	if secondKind != "Pod" {
		t.Errorf("Expected second item to be Pod, got %s", secondKind)
	}
	podName, _, _ := unstructured.NestedString(secondItem, "metadata", "name")
	if podName != "pod-without-workload" {
		t.Errorf("Expected pod name to be 'pod-without-workload', got %s", podName)
	}
}

// TestGroupPodsByWorkload_NoWorkloadAnnotations tests that pods without workload annotations are returned unchanged
func TestGroupPodsByWorkload_NoWorkloadAnnotations(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	pod1 := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "default",
		},
	}
	pod2 := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod2",
			Namespace: "default",
		},
	}

	pod1Map, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(pod1)
	pod2Map, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(pod2)

	dynamicClient := fake.NewSimpleDynamicClient(scheme)

	podList := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "List",
			"items": []interface{}{
				pod1Map,
				pod2Map,
			},
		},
	}

	result, err := GroupPodsByWorkload(dynamicClient, podList, "default", nil)
	if err != nil {
		t.Fatalf("GroupPodsByWorkload failed: %v", err)
	}

	// Result should be the same as input
	items, found, err := unstructured.NestedSlice(result.Object, "items")
	if err != nil || !found {
		t.Fatalf("Failed to get items from result: err=%v, found=%v", err, found)
	}

	if len(items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(items))
	}

	// All items should be pods
	for i, item := range items {
		itemMap := item.(map[string]interface{})
		kind, _, _ := unstructured.NestedString(itemMap, "kind")
		if kind != "Pod" {
			t.Errorf("Expected item %d to be Pod, got %s", i, kind)
		}
	}
}
