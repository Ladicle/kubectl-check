package checker

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func listJobSetChild(jobset map[string]interface{}, dc dynamic.Interface) (*unstructured.Unstructured, error) {
	// Get namespace and name
	namespace, _, err := unstructured.NestedString(jobset, "metadata", "namespace")
	if err != nil {
		return nil, err
	}
	name, _, err := unstructured.NestedString(jobset, "metadata", "name")
	if err != nil {
		return nil, err
	}

	// List Jobs using dynamic client with label selector
	// JobSet creates Jobs with label jobset.sigs.k8s.io/jobset-name=<jobset-name>
	jobInfo, ok := Resources["Job"]
	if !ok {
		return nil, fmt.Errorf("could not resolve Job resource")
	}
	serviceInfo, ok := Resources["Service"]
	if !ok {
		return nil, fmt.Errorf("could not resolve Service resource")
	}

	labelSelector := labels.Set{
		"jobset.sigs.k8s.io/jobset-name": name,
	}.String()

	jobList, err := dc.Resource(jobInfo.GVR).Namespace(namespace).List(
		context.Background(),
		metav1.ListOptions{LabelSelector: labelSelector},
	)
	if err != nil {
		return nil, err
	}

	serviceList, err := dc.Resource(serviceInfo.GVR).Namespace(namespace).List(
		context.Background(),
		metav1.ListOptions{LabelSelector: labelSelector},
	)
	if err != nil {
		return nil, err
	}

	items := make([]interface{}, 0, len(jobList.Items)+len(serviceList.Items))
	for i := range jobList.Items {
		items = append(items, jobList.Items[i].Object)
	}
	for i := range serviceList.Items {
		items = append(items, serviceList.Items[i].Object)
	}

	if len(items) == 0 {
		return nil, nil
	}

	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "List",
			"items":      items,
		},
	}, nil
}

func checkJobSet(jobset map[string]interface{}, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error) {
	// Get metadata
	name, _, _ := unstructured.NestedString(jobset, "metadata", "name")
	namespace, _, _ := unstructured.NestedString(jobset, "metadata", "namespace")

	status := &ResourceStatus{
		Kind:      "JobSet",
		Name:      name,
		Namespace: namespace,
	}

	// Collect events for this JobSet
	events, err := getResourceEvents(c, namespace, "JobSet", name)
	if err == nil {
		status.Events = events
	}

	// Get status information
	succeededJobs, _, _ := unstructured.NestedInt64(jobset, "status", "succeededJobs")
	failedJobs, _, _ := unstructured.NestedInt64(jobset, "status", "failedJobs")
	activeJobs, _, _ := unstructured.NestedInt64(jobset, "status", "active")

	// Get total number of jobs from replicatedJobs
	replicatedJobs, found, _ := unstructured.NestedSlice(jobset, "spec", "replicatedJobs")
	totalJobs := int64(0)
	if found {
		for _, rj := range replicatedJobs {
			rjMap, ok := rj.(map[string]interface{})
			if !ok {
				continue
			}
			replicas, _, _ := unstructured.NestedInt64(rjMap, "replicas")
			if replicas == 0 {
				replicas = 1 // default is 1
			}
			totalJobs += replicas
		}
	}

	// Job status
	jobStatus := fmt.Sprintf("%d/%d Succeeded, %d Active, %d Failed",
		succeededJobs, totalJobs, activeJobs, failedJobs)
	status.AddField("Status:    ", jobStatus)

	// Suspend status
	suspended, found, _ := unstructured.NestedBool(jobset, "spec", "suspend")
	if found {
		status.AddField("Suspended: ", formatBool(suspended))
	}

	// Conditions
	conditionsRaw, found, _ := unstructured.NestedSlice(jobset, "status", "conditions")
	if found && len(conditionsRaw) > 0 {
		var conditions []string
		for _, condRaw := range conditionsRaw {
			condMap, ok := condRaw.(map[string]interface{})
			if !ok {
				continue
			}
			condStatus, _, _ := unstructured.NestedString(condMap, "status")
			condType, _, _ := unstructured.NestedString(condMap, "type")
			if condStatus == "True" {
				conditions = append(conditions, condType)
			}
		}
		if len(conditions) > 0 {
			status.AddField("Conditions:", formatConditionList(conditions))
		}
	}

	// Age
	creationTimestamp, found, _ := unstructured.NestedString(jobset, "metadata", "creationTimestamp")
	if found && creationTimestamp != "" {
		t, err := time.Parse(time.RFC3339, creationTimestamp)
		if err == nil {
			age := formatAge(t)
			status.AddField("Age:       ", age)
		}
	}

	return status, nil
}
