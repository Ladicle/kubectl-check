package checker

import (
	"testing"

	"k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestCheckJobSet(t *testing.T) {
	jobset := map[string]interface{}{
		"apiVersion": "jobset.x-k8s.io/v1alpha2",
		"kind":       "JobSet",
		"metadata": map[string]interface{}{
			"name":      "test-jobset",
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"replicatedJobs": []interface{}{
				map[string]interface{}{
					"name":     "worker",
					"replicas": int64(3),
				},
			},
		},
		"status": map[string]interface{}{
			"conditions": []interface{}{
				map[string]interface{}{
					"type":   "Completed",
					"status": "True",
				},
			},
		},
	}

	clientset := kubefake.NewClientset()
	dc := fake.NewSimpleDynamicClient(scheme.Scheme)

	status, err := checkJobSet(jobset, clientset, dc)
	if err != nil {
		t.Fatalf("checkJobSet failed: %v", err)
	}

	if status.Kind != "JobSet" {
		t.Errorf("expected Kind=JobSet, got %s", status.Kind)
	}

	if status.Name != "test-jobset" {
		t.Errorf("expected Name=test-jobset, got %s", status.Name)
	}

	if status.Namespace != "default" {
		t.Errorf("expected Namespace=default, got %s", status.Namespace)
	}
}
