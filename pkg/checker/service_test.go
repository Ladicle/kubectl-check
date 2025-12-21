package checker

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestCheckService(t *testing.T) {
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-service",
			Namespace:         "default",
			CreationTimestamp: metav1.Now(),
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "10.0.0.1",
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(8080),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app": "test",
			},
		},
	}

	clientset := kubefake.NewClientset()
	dc := fake.NewSimpleDynamicClient(scheme.Scheme)

	status, err := checkService(svc, clientset, dc)
	if err != nil {
		t.Fatalf("checkService failed: %v", err)
	}

	if status.Kind != "Service" {
		t.Errorf("expected Kind=Service, got %s", status.Kind)
	}

	if status.Name != "test-service" {
		t.Errorf("expected Name=test-service, got %s", status.Name)
	}

	if status.Namespace != "default" {
		t.Errorf("expected Namespace=default, got %s", status.Namespace)
	}

	// Check fields
	foundType := false
	foundIP := false
	foundPorts := false
	foundSelector := false
	foundAge := false
	for _, field := range status.Fields {
		if field.Key == "Type:      " {
			foundType = true
			if field.Value != "ClusterIP" {
				t.Errorf("unexpected Type: %s", field.Value)
			}
		}
		if field.Key == "IP:        " {
			foundIP = true
			if field.Value != "10.0.0.1" {
				t.Errorf("unexpected IP: %s", field.Value)
			}
		}
		if field.Key == "Ports:     " {
			foundPorts = true
			if field.Value != "80/TCP->8080 [http]" {
				t.Errorf("unexpected Ports: %s", field.Value)
			}
		}
		if field.Key == "Selector:  " {
			foundSelector = true
			if field.Value != "app=test" {
				t.Errorf("unexpected Selector: %s", field.Value)
			}
		}
		if field.Key == "Age:       " {
			foundAge = true
		}
	}
	if !foundType {
		t.Error("Type field not found")
	}
	if !foundIP {
		t.Error("IP field not found")
	}
	if !foundPorts {
		t.Error("Ports field not found")
	}
	if !foundSelector {
		t.Error("Selector field not found")
	}
	if !foundAge {
		t.Error("Age field not found")
	}
}
