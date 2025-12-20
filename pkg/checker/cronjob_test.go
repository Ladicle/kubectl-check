package checker

import (
	"strings"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestCheckCronJob(t *testing.T) {
	suspend := false
	lastScheduleTime := metav1.Now()
	lastSuccessfulTime := metav1.Now()

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-cronjob",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/5 * * * *",
			Suspend:  &suspend,
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test-cronjob",
					},
				},
				Spec: batchv1.JobSpec{
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
			},
		},
		Status: batchv1.CronJobStatus{
			Active:             []corev1.ObjectReference{},
			LastScheduleTime:   &lastScheduleTime,
			LastSuccessfulTime: &lastSuccessfulTime,
		},
	}

	clientset := kubefake.NewClientset()
	dc := fake.NewSimpleDynamicClient(scheme.Scheme)

	status, err := checkCronJob(cronJob, clientset, dc)
	if err != nil {
		t.Fatalf("checkCronJob failed: %v", err)
	}

	if status.Kind != "CronJob" {
		t.Errorf("expected Kind=CronJob, got %s", status.Kind)
	}

	if status.Name != "test-cronjob" {
		t.Errorf("expected Name=test-cronjob, got %s", status.Name)
	}

	if status.Namespace != "default" {
		t.Errorf("expected Namespace=default, got %s", status.Namespace)
	}

	// Check fields
	foundSchedule := false
	foundSuspend := false
	foundActive := false
	foundLastSchedule := false
	foundLastSuccessful := false

	for _, field := range status.Fields {
		key := strings.TrimSpace(field.Key)
		if key == "Schedule:" {
			foundSchedule = true
			if field.Value != "*/5 * * * *" {
				t.Errorf("unexpected Schedule: %s", field.Value)
			}
		}
		if key == "Suspend:" {
			foundSuspend = true
			if field.Value != "False" {
				t.Errorf("unexpected Suspend: %s (expected False)", field.Value)
			}
		}
		if key == "Active:" {
			foundActive = true
			if field.Value != "0" {
				t.Errorf("unexpected Active: %s", field.Value)
			}
		}
		if key == "Last Schedule:" {
			foundLastSchedule = true
		}
		if key == "Last Successful:" {
			foundLastSuccessful = true
		}
	}

	if !foundSchedule {
		t.Error("Schedule field not found")
	}
	if !foundSuspend {
		t.Error("Suspend field not found")
	}
	if !foundActive {
		t.Error("Active field not found")
	}
	if !foundLastSchedule {
		t.Error("Last Schedule field not found")
	}
	if !foundLastSuccessful {
		t.Error("Last Successful field not found")
	}
}

func TestCheckCronJob_Suspended(t *testing.T) {
	suspend := true

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "suspended-cronjob",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "0 0 * * *",
			Suspend:  &suspend,
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "suspended-cronjob",
					},
				},
				Spec: batchv1.JobSpec{
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
			},
		},
		Status: batchv1.CronJobStatus{
			Active: []corev1.ObjectReference{},
		},
	}

	clientset := kubefake.NewClientset()
	dc := fake.NewSimpleDynamicClient(scheme.Scheme)

	status, err := checkCronJob(cronJob, clientset, dc)
	if err != nil {
		t.Fatalf("checkCronJob failed: %v", err)
	}

	// Check that Suspend is true
	foundSuspend := false
	for _, field := range status.Fields {
		key := strings.TrimSpace(field.Key)
		if key == "Suspend:" {
			foundSuspend = true
			if field.Value != "True" {
				t.Errorf("expected Suspend=True, got %s", field.Value)
			}
		}
	}

	if !foundSuspend {
		t.Error("Suspend field not found")
	}
}
