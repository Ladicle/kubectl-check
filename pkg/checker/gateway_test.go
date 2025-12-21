package checker

import (
	"testing"

	"k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestCheckGateway(t *testing.T) {
	gateway := map[string]interface{}{
		"apiVersion": "gateway.networking.k8s.io/v1",
		"kind":       "Gateway",
		"metadata": map[string]interface{}{
			"name":      "test-gateway",
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"gatewayClassName": "test-class",
			"listeners": []interface{}{
				map[string]interface{}{
					"name":     "http",
					"protocol": "HTTP",
					"port":     int64(80),
				},
			},
		},
		"status": map[string]interface{}{
			"conditions": []interface{}{
				map[string]interface{}{
					"type":   "Accepted",
					"status": "True",
					"reason": "Accepted",
				},
			},
		},
	}

	clientset := kubefake.NewClientset()
	dc := fake.NewSimpleDynamicClient(scheme.Scheme)

	status, err := checkGateway(gateway, clientset, dc)
	if err != nil {
		t.Fatalf("checkGateway failed: %v", err)
	}

	if status.Kind != "Gateway" {
		t.Errorf("expected Kind=Gateway, got %s", status.Kind)
	}

	if status.Name != "test-gateway" {
		t.Errorf("expected Name=test-gateway, got %s", status.Name)
	}

	if status.Namespace != "default" {
		t.Errorf("expected Namespace=default, got %s", status.Namespace)
	}

	// Check fields
	foundClass := false
	foundListeners := false
	for _, field := range status.Fields {
		if field.Key == "Class:     " {
			foundClass = true
			if field.Value != "test-class" {
				t.Errorf("unexpected Class: %s", field.Value)
			}
		}
		if field.Key == "Listeners: " {
			foundListeners = true
			if field.Value != "http:HTTP:80" {
				t.Errorf("unexpected Listeners: %s", field.Value)
			}
		}
	}
	if !foundClass {
		t.Error("Class field not found")
	}
	if !foundListeners {
		t.Error("Listeners field not found")
	}
}
