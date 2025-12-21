package checker

import (
	"fmt"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func listJobChild(job *batchv1.Job, dc dynamic.Interface) (*unstructured.Unstructured, error) {
	// List Pods using dynamic client
	podList, err := ListPods(dc, job.Namespace, labels.Set(job.Spec.Selector.MatchLabels).String())
	if err != nil {
		return nil, err
	}

	// Group pods by workload if they have kueue annotations
	return GroupPodsByWorkload(dc, podList, job.Namespace, nil)
}

func checkJob(job *batchv1.Job, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error) {
	status := &ResourceStatus{
		Kind:      "Job",
		Name:      job.Name,
		Namespace: job.Namespace,
	}

	// Collect events for this job
	events, err := getResourceEvents(c, job.Namespace, "Job", job.Name)
	if err == nil {
		status.Events = events
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
