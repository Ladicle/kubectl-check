package checker

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	kueuev1beta1 "sigs.k8s.io/kueue/apis/kueue/v1beta1"
)

// formatConditionList formats a list of conditions into a comma-separated string
func formatConditionList(conditions []string) string {
	return strings.Join(conditions, ", ")
}

func checkWorkload(workload *kueuev1beta1.Workload, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error) {
	status := &ResourceStatus{
		Kind:      "Workload",
		Name:      workload.Name,
		Namespace: workload.Namespace,
	}

	// Queue name
	if workload.Spec.QueueName != "" {
		status.AddField("Queue:           ", string(workload.Spec.QueueName))
	}

	// Status
	var admitted string
	if workload.Status.Admission != nil {
		admitted = "Admitted"
		status.AddField("Status:          ", admitted)
		status.AddField("Cluster Queue:   ", string(workload.Status.Admission.ClusterQueue))
	} else {
		admitted = "Pending"
		status.AddField("Status:          ", admitted)
	}

	// Conditions
	var conditions []string
	for _, cond := range workload.Status.Conditions {
		if cond.Status == "True" {
			conditions = append(conditions, string(cond.Type))
		}
	}
	if len(conditions) > 0 {
		status.AddField("Conditions:      ", formatConditionList(conditions))
	}

	// Age
	age := formatAge(workload.CreationTimestamp.Time)
	status.AddField("Age:             ", age)

	return status, nil
}

// listWorkloadChild lists pods that belong to this workload
func listWorkloadChild(workload *kueuev1beta1.Workload, dc dynamic.Interface) (*unstructured.Unstructured, error) {
	podInfo, ok := Resources["Pod"]
	if !ok {
		return nil, fmt.Errorf("could not resolve Pod resource")
	}

	// List all pods in the namespace
	podList, err := dc.Resource(podInfo.GVR).Namespace(workload.Namespace).List(
		context.Background(),
		metav1.ListOptions{},
	)
	if err != nil {
		return nil, err
	}

	// Filter pods that have the workload annotation
	var filteredPods []unstructured.Unstructured
	for _, item := range podList.Items {
		annotations := item.GetAnnotations()
		if annotations != nil {
			if wlName, ok := annotations["kueue.x-k8s.io/workload"]; ok && wlName == workload.Name {
				filteredPods = append(filteredPods, item)
			}
		}
	}

	if len(filteredPods) == 0 {
		return nil, nil
	}

	// Return as a list
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "List",
			"items":      filteredPods,
		},
	}, nil
}

// GroupPodsByWorkload groups pods by their kueue workload annotation and returns
// a combined unstructured list containing both workload resources (with pods as children)
// and ungrouped pods. This enables hierarchical display like: StatefulSet -> Workload -> Pod
func GroupPodsByWorkload(dc dynamic.Interface, podList *unstructured.Unstructured, namespace string, c kubernetes.Interface) (*unstructured.Unstructured, error) {
	if podList == nil {
		return nil, nil
	}

	pods, found, err := unstructured.NestedSlice(podList.Object, "items")
	if err != nil || !found || len(pods) == 0 {
		return podList, nil
	}

	// Group pods by workload annotation
	workloadPods := make(map[string][]interface{})
	var ungroupedPods []interface{}

	for _, podObj := range pods {
		pod, ok := podObj.(map[string]interface{})
		if !ok {
			continue
		}

		// Check for kueue workload annotation
		annotations, found, _ := unstructured.NestedStringMap(pod, "metadata", "annotations")
		if found {
			if workloadName, ok := annotations["kueue.x-k8s.io/workload"]; ok {
				workloadPods[workloadName] = append(workloadPods[workloadName], podObj)
				continue
			}
		}
		ungroupedPods = append(ungroupedPods, podObj)
	}

	// If no pods have workload annotations, return original list
	if len(workloadPods) == 0 {
		return podList, nil
	}

	// Build combined list: workloads with their pods + ungrouped pods
	var combinedItems []interface{}

	// Process each workload group
	workloadGVR := schema.GroupVersionResource{Group: "kueue.x-k8s.io", Version: "v1beta1", Resource: "workloads"}
	for workloadName, pods := range workloadPods {
		// Fetch workload resource
		workloadObj, err := dc.Resource(workloadGVR).Namespace(namespace).Get(
			context.Background(), workloadName, metav1.GetOptions{})
		if err != nil {
			// If workload not found, treat pods as ungrouped
			ungroupedPods = append(ungroupedPods, pods...)
			continue
		}

		// Store pods in a special field so they can be processed as children
		if err := unstructured.SetNestedField(workloadObj.Object, pods, "_childPods"); err != nil {
			// On error, treat pods as ungrouped
			ungroupedPods = append(ungroupedPods, pods...)
			continue
		}

		combinedItems = append(combinedItems, workloadObj.Object)
	}

	// Add ungrouped pods
	combinedItems = append(combinedItems, ungroupedPods...)

	// Create new list with combined items
	result := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "List",
			"items":      combinedItems,
		},
	}

	return result, nil
}
