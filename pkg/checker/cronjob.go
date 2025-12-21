package checker

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func listCronJobChild(cronJob *batchv1.CronJob, dc dynamic.Interface) (*unstructured.Unstructured, error) {
	// List Jobs using dynamic client
	jobInfo, ok := Resources["Job"]
	if !ok {
		return nil, fmt.Errorf("could not resolve Job resource")
	}

	// Find jobs owned by this CronJob
	jobList, err := dc.Resource(jobInfo.GVR).Namespace(cronJob.Namespace).List(
		context.Background(),
		metav1.ListOptions{
			LabelSelector: labels.Set(cronJob.Spec.JobTemplate.Labels).String(),
		},
	)
	if err != nil {
		return nil, err
	}

	// Find the most recent job
	if len(jobList.Items) == 0 {
		return nil, nil
	}

	latestJob := &jobList.Items[0]
	for i := range jobList.Items[1:] {
		job := &jobList.Items[i+1]
		// Compare creation timestamps
		var latestCreation, jobCreation metav1.Time
		latestCreation = latestJob.GetCreationTimestamp()
		jobCreation = job.GetCreationTimestamp()
		if latestCreation.Before(&jobCreation) {
			latestJob = job
		}
	}

	return latestJob, nil
}

func checkCronJob(cronJob *batchv1.CronJob, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error) {
	status := &ResourceStatus{
		Kind:      "CronJob",
		Name:      cronJob.Name,
		Namespace: cronJob.Namespace,
	}

	// CronJob schedule and status
	status.AddField("Schedule:        ", cronJob.Spec.Schedule)
	status.AddField("Suspend:         ", formatBool(cronJob.Spec.Suspend != nil && *cronJob.Spec.Suspend))

	// Active jobs count
	status.AddField("Active:          ", fmt.Sprintf("%d", len(cronJob.Status.Active)))

	// Last schedule time
	if cronJob.Status.LastScheduleTime != nil {
		status.AddField("Last Schedule:   ", formatTimestampWithAge(cronJob.Status.LastScheduleTime.Time))
	}

	// Last successful time
	if cronJob.Status.LastSuccessfulTime != nil {
		status.AddField("Last Successful: ", formatTimestampWithAge(cronJob.Status.LastSuccessfulTime.Time))
	}

	// Age
	age := formatAge(cronJob.CreationTimestamp.Time)
	status.AddField("Age:             ", age)

	return status, nil
}
