package checker

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	"github.com/Ladicle/kubectl-check/pkg/client"
)

func TestFetchUnknownResourceWithStatus(t *testing.T) {
	tests := []struct {
		name       string
		gvr        schema.GroupVersionResource
		namespace  string
		resName    string
		response   *metav1.Table
		wantErr    bool
		wantName   string
		wantKind   string
		wantFields int
	}{
		{
			name: "Unknown resource with custom columns",
			gvr: schema.GroupVersionResource{
				Group:    "example.com",
				Version:  "v1",
				Resource: "unknownresources",
			},
			namespace: "test-namespace",
			resName:   "test-unknown-resource",
			response: &metav1.Table{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Table",
					APIVersion: "meta.k8s.io/v1",
				},
				ColumnDefinitions: []metav1.TableColumnDefinition{
					{Name: "Name", Type: "string", Format: "name"},
					{Name: "Phase", Type: "string"},
					{Name: "Ready", Type: "string"},
				},
				Rows: []metav1.TableRow{
					{
						Cells: []interface{}{"test-unknown-resource", "Running", "2/2"},
						Object: runtime.RawExtension{
							Raw: []byte(`{
								"apiVersion": "example.com/v1",
								"kind": "UnknownResource",
								"metadata": {
									"name": "test-unknown-resource",
									"namespace": "test-namespace",
									"uid": "test-uid-12345",
									"creationTimestamp": "2025-12-15T13:00:00Z"
								},
								"spec": {
									"replicas": 2
								},
								"status": {
									"phase": "Running",
									"ready": "2/2"
								}
							}`),
						},
					},
				},
			},
			wantErr:    false,
			wantName:   "test-unknown-resource",
			wantKind:   "UnknownResource",
			wantFields: 2, // Phase + Ready (Age is always added separately)
		},
		{
			name: "Empty table response",
			gvr: schema.GroupVersionResource{
				Group:    "example.com",
				Version:  "v1",
				Resource: "examples",
			},
			namespace: "default",
			resName:   "test",
			response: &metav1.Table{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Table",
					APIVersion: "meta.k8s.io/v1",
				},
				Rows: []metav1.TableRow{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server that returns the Table response
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			// Create REST config pointing to test server with proper serializer
			restConfig := &rest.Config{
				Host: server.URL,
				ContentConfig: rest.ContentConfig{
					NegotiatedSerializer: serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs},
				},
			}

			// Create fake clientset
			clientset := fake.NewSimpleClientset()

			// Create ClientManager
			clientManager := &client.Manager{
				Clientset:  clientset,
				RESTConfig: restConfig,
			}

			// Create checker config and checker
			config := &Config{
				ClientManager: clientManager,
			}
			checker := &Checker{
				Config: config,
			}

			// Test fetchUnknownResourceWithStatus
			obj, status, err := checker.fetchUnknownResourceWithStatus(tt.gvr, tt.namespace, tt.resName, tt.wantKind)

			if (err != nil) != tt.wantErr {
				t.Errorf("fetchUnknownResourceWithStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if obj.GetName() != tt.wantName {
				t.Errorf("obj.GetName() = %v, want %v", obj.GetName(), tt.wantName)
			}

			if obj.GetKind() != tt.wantKind {
				t.Errorf("obj.GetKind() = %v, want %v", obj.GetKind(), tt.wantKind)
			}

			if status.Name != tt.wantName {
				t.Errorf("status.Name = %v, want %v", status.Name, tt.wantName)
			}

			if status.Kind != tt.wantKind {
				t.Errorf("status.Kind = %v, want %v", status.Kind, tt.wantKind)
			}

			// Check that we got some fields (excluding Age which is always added)
			if len(status.Fields) < tt.wantFields {
				t.Errorf("status.Fields has %d fields, want at least %d", len(status.Fields), tt.wantFields)
			}
		})
	}
}

func TestParseGVRFromOwnerRef_UnknownResource(t *testing.T) {
	ownerRef := &metav1.OwnerReference{
		APIVersion: "example.com/v1",
		Kind:       "UnknownResource",
		Name:       "test-unknown-resource",
	}

	gvr, err := parseGVRFromOwnerRef(ownerRef)
	if err != nil {
		t.Fatalf("parseGVRFromOwnerRef() error = %v", err)
	}

	if gvr.Group != "example.com" {
		t.Errorf("Group = %v, want example.com", gvr.Group)
	}

	if gvr.Version != "v1" {
		t.Errorf("Version = %v, want v1", gvr.Version)
	}

	if gvr.Resource != "unknownresources" {
		t.Errorf("Resource = %v, want unknownresources", gvr.Resource)
	}
}
