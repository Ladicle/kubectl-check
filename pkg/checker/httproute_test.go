package checker

import (
	"testing"

	"k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestCheckHTTPRoute(t *testing.T) {
	httpRoute := map[string]interface{}{
		"apiVersion": "gateway.networking.k8s.io/v1",
		"kind":       "HTTPRoute",
		"metadata": map[string]interface{}{
			"name":      "test-httproute",
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"parentRefs": []interface{}{
				map[string]interface{}{
					"name": "test-gateway",
				},
			},
			"hostnames": []interface{}{"example.com"},
			"rules": []interface{}{
				map[string]interface{}{
					"backendRefs": []interface{}{
						map[string]interface{}{
							"name": "test-service",
							"port": int64(80),
						},
					},
				},
			},
		},
		"status": map[string]interface{}{
			"parents": []interface{}{
				map[string]interface{}{
					"parentRef": map[string]interface{}{
						"name": "test-gateway",
					},
					"conditions": []interface{}{
						map[string]interface{}{
							"type":   "Accepted",
							"status": "True",
							"reason": "Accepted",
						},
					},
				},
			},
		},
	}

	clientset := kubefake.NewClientset()
	dc := fake.NewSimpleDynamicClient(scheme.Scheme)

	status, err := checkHTTPRoute(httpRoute, clientset, dc)
	if err != nil {
		t.Fatalf("checkHTTPRoute failed: %v", err)
	}

	if status.Kind != "HTTPRoute" {
		t.Errorf("expected Kind=HTTPRoute, got %s", status.Kind)
	}

	if status.Name != "test-httproute" {
		t.Errorf("expected Name=test-httproute, got %s", status.Name)
	}

	if status.Namespace != "default" {
		t.Errorf("expected Namespace=default, got %s", status.Namespace)
	}

	// Check fields
	foundParents := false
	foundHostnames := false
	foundRules := false
	foundBackends := false
	for _, field := range status.Fields {
		if field.Key == "Parents:   " {
			foundParents = true
			if field.Value != "test-gateway" {
				t.Errorf("unexpected Parents: %s", field.Value)
			}
		}
		if field.Key == "Hostnames: " {
			foundHostnames = true
			if field.Value != "example.com" {
				t.Errorf("unexpected Hostnames: %s", field.Value)
			}
		}
		if field.Key == "Rules:     " {
			foundRules = true
			if field.Value != "1" {
				t.Errorf("unexpected Rules: %s", field.Value)
			}
		}
		if field.Key == "Backends:  " {
			foundBackends = true
			if field.Value != "test-service" {
				t.Errorf("unexpected Backends: %s", field.Value)
			}
		}
	}
	if !foundParents {
		t.Error("Parents field not found")
	}
	if !foundHostnames {
		t.Error("Hostnames field not found")
	}
	if !foundRules {
		t.Error("Rules field not found")
	}
	if !foundBackends {
		t.Error("Backends field not found")
	}
}
