package checker

import (
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestCheckJob(t *testing.T) {
	completions := int32(1)
	parallelism := int32(1)
	backoffLimit := int32(6)
	startTime := metav1.Now()

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-job",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: batchv1.JobSpec{
			Completions:  &completions,
			Parallelism:  &parallelism,
			BackoffLimit: &backoffLimit,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"job-name": "test-job",
				},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test",
							Image: "busybox:latest",
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
		Status: batchv1.JobStatus{
			Succeeded: 1,
			Active:    0,
			Failed:    0,
			StartTime: &startTime,
		},
	}

	clientset := kubefake.NewClientset()
	dc := fake.NewSimpleDynamicClient(scheme.Scheme)

	status, err := checkJob(job, clientset, dc)
	if err != nil {
		t.Fatalf("checkJob failed: %v", err)
	}

	if status.Kind != "Job" {
		t.Errorf("expected Kind=Job, got %s", status.Kind)
	}

	if status.Name != "test-job" {
		t.Errorf("expected Name=test-job, got %s", status.Name)
	}

	if status.Namespace != "default" {
		t.Errorf("expected Namespace=default, got %s", status.Namespace)
	}

	// Check fields
	foundStatus := false

	for _, field := range status.Fields {
		if field.Key == "Status:    " {
			foundStatus = true
			if field.Value != "1/1 Succeeded, 0 Active, 0/6 Failed" {
				t.Errorf("unexpected Status: %s", field.Value)
			}
		}
	}

	if !foundStatus {
		t.Error("Status field not found")
	}
}

func TestCheckJob_WithConditions(t *testing.T) {
	completions := int32(1)
	parallelism := int32(1)
	backoffLimit := int32(6)
	startTime := metav1.Now()
	completionTime := metav1.Now()

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-job",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: batchv1.JobSpec{
			Completions:  &completions,
			Parallelism:  &parallelism,
			BackoffLimit: &backoffLimit,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"job-name": "completed-job",
				},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test",
							Image: "busybox:latest",
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
		Status: batchv1.JobStatus{
			Succeeded:      1,
			Active:         0,
			Failed:         0,
			StartTime:      &startTime,
			CompletionTime: &completionTime,
			Conditions: []batchv1.JobCondition{
				{
					Type:   batchv1.JobComplete,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	clientset := kubefake.NewClientset()
	dc := fake.NewSimpleDynamicClient(scheme.Scheme)

	status, err := checkJob(job, clientset, dc)
	if err != nil {
		t.Fatalf("checkJob failed: %v", err)
	}

	// Check for Conditions and Age fields
	foundConditions := false
	foundAge := false

	for _, field := range status.Fields {
		if field.Key == "Conditions:" {
			foundConditions = true
			if field.Value != "Complete" {
				t.Errorf("unexpected Conditions: %s", field.Value)
			}
		}
		if field.Key == "Age:       " {
			foundAge = true
		}
	}

	if !foundConditions {
		t.Error("Conditions field not found")
	}
	if !foundAge {
		t.Error("Age field not found")
	}
}
