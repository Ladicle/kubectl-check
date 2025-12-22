package checker

import (
	"context"
	"fmt"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func listJobChild(job *batchv1.Job, dc dynamic.Interface) (*unstructured.Unstructured, error) {
	var allItems []unstructured.Unstructured

	// Collect PVCs referenced in the pod template
	pvcNames := make(map[string]bool)
	for _, vol := range job.Spec.Template.Spec.Volumes {
		if vol.PersistentVolumeClaim != nil {
			pvcNames[vol.PersistentVolumeClaim.ClaimName] = true
		}
	}

	// Fetch PVCs
	if len(pvcNames) > 0 {
		pvcGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumeclaims"}
		for pvcName := range pvcNames {
			pvc, err := dc.Resource(pvcGVR).Namespace(job.Namespace).Get(context.Background(), pvcName, metav1.GetOptions{})
			if err == nil {
				allItems = append(allItems, *pvc)
			}
		}
	}

	// List Pods using dynamic client
	podList, err := ListPods(dc, job.Namespace, labels.Set(job.Spec.Selector.MatchLabels).String())
	if err != nil {
		return nil, err
	}

	// Group pods by workload if they have kueue annotations
	podsResult, err := GroupPodsByWorkload(dc, podList, job.Namespace, nil)
	if err != nil {
		return nil, err
	}

	if podsResult != nil {
		if podsResult.IsList() {
			list, _ := podsResult.ToList()
			allItems = append(allItems, list.Items...)
		} else {
			allItems = append(allItems, *podsResult)
		}
	}

	if len(allItems) == 0 {
		return nil, nil
	}

	// Convert allItems to raw objects for the list
	items := make([]interface{}, len(allItems))
	for i := range allItems {
		items[i] = allItems[i].Object
	}

	// Convert to Unstructured with list content
	result := &unstructured.Unstructured{}
	result.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "List",
		"items":      items,
	})
	return result, nil
}

func checkJob(job *batchv1.Job, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error) {
	status := &ResourceStatus{
		Kind:      "Job",
		Name:      job.Name,
		Namespace: job.Namespace,
	}

	// Job status
	completions := "?"
	if job.Spec.Completions != nil {
		completions = fmt.Sprintf("%d", *job.Spec.Completions)
	}
	backoffLimit := "?"
	if job.Spec.BackoffLimit != nil {
		backoffLimit = fmt.Sprintf("%d", *job.Spec.BackoffLimit)
	}

	jobStatus := fmt.Sprintf("%d/%s Succeeded, %d Active, %d/%s Failed",
		job.Status.Succeeded, completions, job.Status.Active, job.Status.Failed, backoffLimit)
	status.AddField("Status:    ", jobStatus)

	// Check conditions
	conditions := []string{}
	for _, cond := range job.Status.Conditions {
		if cond.Status == "True" {
			conditions = append(conditions, string(cond.Type))
		}
	}
	if len(conditions) > 0 {
		status.AddField("Conditions:", strings.Join(conditions, ", "))
	}

	// Age and Duration
	if job.Status.StartTime != nil {
		age := formatAge(job.Status.StartTime.Time)
		if job.Status.CompletionTime != nil {
			duration := job.Status.CompletionTime.Sub(job.Status.StartTime.Time)
			age = fmt.Sprintf("%s (completed in %s)", age, duration.String())
		}
		status.AddField("Age:       ", age)
	}

	return status, nil
}
