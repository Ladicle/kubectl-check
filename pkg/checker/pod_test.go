package checker

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestCheckPod(t *testing.T) {
	tests := []struct {
		name                 string
		pod                  *corev1.Pod
		expectedFields       map[string]string
		wantFailedContainers int
	}{
		{
			name: "ready pod with resources",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-pod",
					Namespace:         "default",
					CreationTimestamp: metav1.Now(),
				},
				Spec: corev1.PodSpec{
					NodeName: "test-node",
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("200m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					PodIP: "10.244.0.5",
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:         "nginx",
							Ready:        true,
							RestartCount: 0,
							ContainerID:  "docker://123",
						},
					},
				},
			},
			expectedFields: map[string]string{
				"Ready:   ": "1/1 (0 restarts)",
				"Status:  ": "Running",
				"IP:      ": "10.244.0.5",
				"Node:    ": "test-node",
			},
			wantFailedContainers: 0,
		},
		{
			name: "not ready pod with failed container",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-pod",
					Namespace:         "default",
					CreationTimestamp: metav1.Now(),
				},
				Spec: corev1.PodSpec{
					NodeName: "test-node",
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionFalse,
						},
					},
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:        "nginx",
							Ready:       false,
							ContainerID: "docker://abc123",
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason:  "CrashLoopBackOff",
									Message: "Container failed",
								},
							},
						},
					},
				},
			},
			expectedFields: map[string]string{
				"Ready:   ": "0/1 (0 restarts)",
				"Status:  ": "Pending",
			},
			wantFailedContainers: 1,
		},
		{
			name: "pod with container resources",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-pod",
					Namespace:         "default",
					CreationTimestamp: metav1.Now(),
				},
				Spec: corev1.PodSpec{
					NodeName: "test-node",
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "app:latest",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("200m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					PodIP: "10.244.0.6",
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:         "app",
							Ready:        true,
							RestartCount: 0,
							ContainerID:  "docker://app123",
						},
					},
				},
			},
			expectedFields: map[string]string{
				"Ready:   ":                  "1/1 (0 restarts)",
				"Status:  ":                  "Running",
				"Container Resources Total:": "",
				"   Request:":                "200m cpu, 256Mi mem",
				"   Limits: ":                "500m cpu, 512Mi mem",
			},
			wantFailedContainers: 0,
		},
		{
			name: "pod with pod-level resources",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-pod",
					Namespace:         "default",
					CreationTimestamp: metav1.Now(),
				},
				Spec: corev1.PodSpec{
					NodeName: "test-node",
					Resources: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("300m"),
							corev1.ResourceMemory: resource.MustParse("384Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("600m"),
							corev1.ResourceMemory: resource.MustParse("768Mi"),
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "app:latest",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("200m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("400m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					PodIP: "10.244.0.7",
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:         "app",
							Ready:        true,
							RestartCount: 0,
							ContainerID:  "docker://app456",
						},
					},
				},
			},
			expectedFields: map[string]string{
				"Ready:   ":                  "1/1 (0 restarts)",
				"Status:  ":                  "Running",
				"Pod Resources:":             "",
				"Container Resources Total:": "",
			},
			wantFailedContainers: 0,
		},
		{
			name: "pod with extended resources",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-pod",
					Namespace:         "default",
					CreationTimestamp: metav1.Now(),
				},
				Spec: corev1.PodSpec{
					NodeName: "test-node",
					Containers: []corev1.Container{
						{
							Name:  "gpu-app",
							Image: "gpu-app:latest",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:                    resource.MustParse("100m"),
									corev1.ResourceMemory:                 resource.MustParse("128Mi"),
									corev1.ResourceName("nvidia.com/gpu"): resource.MustParse("1"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:                    resource.MustParse("200m"),
									corev1.ResourceMemory:                 resource.MustParse("256Mi"),
									corev1.ResourceName("nvidia.com/gpu"): resource.MustParse("1"),
								},
							},
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					PodIP: "10.244.0.8",
					Conditions: []corev1.PodCondition{
						{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						},
					},
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name:         "gpu-app",
							Ready:        true,
							RestartCount: 0,
							ContainerID:  "docker://gpu123",
						},
					},
				},
			},
			expectedFields: map[string]string{
				"Ready:   ":                  "1/1 (0 restarts)",
				"Status:  ":                  "Running",
				"Container Resources Total:": "",
				"   Request:":                "100m cpu, 128Mi mem, 1 gpu",
				"   Limits: ":                "200m cpu, 256Mi mem, 1 gpu"},
			wantFailedContainers: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := kubefake.NewClientset()
			dc := fake.NewSimpleDynamicClient(scheme.Scheme)

			status, err := checkPod(tt.pod, clientset, dc)
			if err != nil {
				t.Fatalf("checkPod failed: %v", err)
			}

			if status.Kind != "Pod" {
				t.Errorf("expected Kind=Pod, got %s", status.Kind)
			}

			if status.Name != "test-pod" {
				t.Errorf("expected Name=test-pod, got %s", status.Name)
			}

			// Check fields
			if tt.expectedFields != nil {
				foundFields := make(map[string]bool)
				for _, field := range status.Fields {
					if expected, ok := tt.expectedFields[field.Key]; ok {
						foundFields[field.Key] = true
						if field.Value != expected {
							t.Errorf("field %s: expected %q, got %q", field.Key, expected, field.Value)
						}
					}
				}

				for key := range tt.expectedFields {
					if !foundFields[key] {
						t.Errorf("field %s not found", key)
					}
				}
			}

			// Check failed containers
			if len(status.FailedContainers) != tt.wantFailedContainers {
				t.Errorf("expected %d failed containers, got %d", tt.wantFailedContainers, len(status.FailedContainers))
			}

			if tt.wantFailedContainers > 0 && len(status.FailedContainers) > 0 {
				fc := status.FailedContainers[0]
				if fc.PodName != "test-pod" {
					t.Errorf("expected PodName=test-pod, got %s", fc.PodName)
				}
				if fc.Namespace != "default" {
					t.Errorf("expected Namespace=default, got %s", fc.Namespace)
				}
			}
		})
	}
}

func TestCollectFailedContainers(t *testing.T) {
	tests := []struct {
		name      string
		pod       *corev1.Pod
		wantCount int
	}{
		{
			name: "all containers ready",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{Name: "c1", Ready: true, ContainerID: "docker://123"},
						{Name: "c2", Ready: true, ContainerID: "docker://456"},
					},
				},
			},
			wantCount: 0,
		},
		{
			name: "one container failed",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{Name: "c1", Ready: true, ContainerID: "docker://123"},
						{
							Name:        "c2",
							Ready:       false,
							ContainerID: "docker://456",
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"},
							},
						},
					},
				},
			},
			wantCount: 1,
		},
		{
			name: "init container failed",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					InitContainerStatuses: []corev1.ContainerStatus{
						{
							Name:        "init",
							Ready:       false,
							ContainerID: "docker://789",
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{Reason: "Error"},
							},
						},
					},
					ContainerStatuses: []corev1.ContainerStatus{
						{Name: "c1", Ready: true, ContainerID: "docker://123"},
					},
				},
			},
			wantCount: 1,
		},
		{
			name: "container not started",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{Name: "c1", Ready: false, ContainerID: ""},
					},
				},
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := kubefake.NewClientset()
			failed := collectContainers(clientset, tt.pod, false)

			if len(failed) != tt.wantCount {
				t.Errorf("expected %d failed containers, got %d", tt.wantCount, len(failed))
			}
		})
	}
}

func TestIsContainerStarted(t *testing.T) {
	tests := []struct {
		name   string
		status corev1.ContainerStatus
		want   bool
	}{
		{
			name: "ready with container ID",
			status: corev1.ContainerStatus{
				Ready:       true,
				ContainerID: "docker://abc123",
			},
			want: true,
		},
		{
			name: "terminated with container ID",
			status: corev1.ContainerStatus{
				Ready: false,
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ContainerID: "docker://abc123",
					},
				},
			},
			want: true,
		},
		{
			name: "last terminated with container ID",
			status: corev1.ContainerStatus{
				Ready: false,
				LastTerminationState: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						ContainerID: "docker://abc123",
					},
				},
			},
			want: true,
		},
		{
			name: "waiting without container ID",
			status: corev1.ContainerStatus{
				Ready: false,
				State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{
						Reason: "ContainerCreating",
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isContainerStarted(tt.status)
			if got != tt.want {
				t.Errorf("isContainerStarted() = %v, want %v", got, tt.want)
			}
		})
	}
}
