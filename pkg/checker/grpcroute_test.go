package checker

import (
	"testing"

	"k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestCheckGRPCRoute(t *testing.T) {
	grpcRoute := map[string]interface{}{
		"apiVersion": "gateway.networking.k8s.io/v1",
		"kind":       "GRPCRoute",
		"metadata": map[string]interface{}{
			"name":      "test-grpcroute",
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"parentRefs": []interface{}{
				map[string]interface{}{
					"name": "test-gateway",
				},
			},
			"hostnames": []interface{}{"grpc.example.com"},
			"rules": []interface{}{
				map[string]interface{}{
					"backendRefs": []interface{}{
						map[string]interface{}{
							"name": "grpc-service",
							"port": int64(50051),
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

	status, err := checkGRPCRoute(grpcRoute, clientset, dc)
	if err != nil {
		t.Fatalf("checkGRPCRoute failed: %v", err)
	}

	if status.Kind != "GRPCRoute" {
		t.Errorf("expected Kind=GRPCRoute, got %s", status.Kind)
	}

	if status.Name != "test-grpcroute" {
		t.Errorf("expected Name=test-grpcroute, got %s", status.Name)
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
			if field.Value != "grpc.example.com" {
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
			if field.Value != "grpc-service" {
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
