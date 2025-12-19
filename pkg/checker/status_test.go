package checker

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/Ladicle/kubectl-check/pkg/pritty"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

func TestResourceStatus_AddField(t *testing.T) {
	rs := &ResourceStatus{
		Kind: "Pod",
		Name: "test-pod",
	}

	rs.AddField("Status:", "Running")
	rs.AddField("Age:", "5m")

	if len(rs.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(rs.Fields))
	}

	if rs.Fields[0].Key != "Status:" || rs.Fields[0].Value != "Running" {
		t.Errorf("unexpected field 0: %+v", rs.Fields[0])
	}
}

func TestResourceStatus_AddChild(t *testing.T) {
	parent := &ResourceStatus{
		Kind: "Deployment",
		Name: "test-deploy",
	}

	child := &ResourceStatus{
		Kind: "ReplicaSet",
		Name: "test-rs",
	}

	parent.AddChild(child)

	if len(parent.Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(parent.Children))
	}

	if parent.Children[0] != child {
		t.Errorf("child not added correctly")
	}
}

func TestResourceStatus_CollectAllEvents(t *testing.T) {
	now := time.Now()

	parent := &ResourceStatus{
		Kind: "Deployment",
		Name: "test-deploy",
		Events: []Event{
			{Timestamp: now, Type: "Normal", Reason: "Created"},
		},
	}

	child := &ResourceStatus{
		Kind: "Pod",
		Name: "test-pod",
		Events: []Event{
			{Timestamp: now.Add(-time.Minute), Type: "Warning", Reason: "Failed"},
		},
	}

	parent.AddChild(child)

	allEvents := parent.CollectAllEvents()

	if len(allEvents) != 2 {
		t.Errorf("expected 2 events, got %d", len(allEvents))
	}
}

func TestResourceStatus_CollectAllFailedContainers(t *testing.T) {
	parent := &ResourceStatus{
		Kind: "Deployment",
		Name: "test-deploy",
	}

	child := &ResourceStatus{
		Kind: "Pod",
		Name: "test-pod",
		FailedContainers: []FailedContainer{
			{Namespace: "default", PodName: "test-pod", ContainerName: "app"},
		},
	}

	parent.AddChild(child)

	failed := parent.CollectAllFailedContainers()

	if len(failed) != 1 {
		t.Errorf("expected 1 failed container, got %d", len(failed))
	}

	if failed[0].ContainerName != "app" {
		t.Errorf("unexpected container name: %s", failed[0].ContainerName)
	}
}

func TestResourceStatus_Render(t *testing.T) {
	tests := []struct {
		name     string
		status   *ResourceStatus
		indent   int
		wantText []string
	}{
		{
			name: "top-level with namespace",
			status: &ResourceStatus{
				Kind:      "Deployment",
				Name:      "nginx",
				Namespace: "default",
				Fields: []StatusField{
					{Key: "Age:", Value: "5m"},
				},
			},
			indent: 0,
			wantText: []string{
				"Deployment/nginx",
				"Namespace:   default",
				"Age: 5m",
			},
		},
		{
			name: "child without namespace",
			status: &ResourceStatus{
				Kind: "Pod",
				Name: "nginx-pod",
				Fields: []StatusField{
					{Key: "Status:", Value: "Running"},
				},
			},
			indent: 2,
			wantText: []string{
				"Pod",
				"nginx-pod",
				"    Status: Running",
			},
		},
		{
			name: "with children",
			status: func() *ResourceStatus {
				parent := &ResourceStatus{
					Kind:      "Deployment",
					Name:      "nginx",
					Namespace: "default",
					Fields: []StatusField{
						{Key: "Age:", Value: "10m"},
					},
				}
				child := &ResourceStatus{
					Kind: "Pod",
					Name: "nginx-pod",
					Fields: []StatusField{
						{Key: "Status:", Value: "Running"},
					},
				}
				parent.AddChild(child)
				return parent
			}(),
			indent: 0,
			wantText: []string{
				"Deployment/nginx",
				"Namespace:   default",
				"Age: 10m",
				"Pod/nginx-pod",
				"    Status: Running",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			printer := &pritty.Printer{
				IOStreams: genericiooptions.IOStreams{
					Out: &buf,
				},
				Color: false,
			}

			tt.status.Render(printer, tt.indent)

			output := buf.String()
			for _, want := range tt.wantText {
				if !strings.Contains(output, want) {
					t.Errorf("output missing expected text %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

func TestRenderEvents(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name         string
		events       []Event
		wantEmpty    bool
		wantContains []string
	}{
		{
			name:      "empty events",
			events:    []Event{},
			wantEmpty: true,
		},
		{
			name: "multiple events in hierarchical order",
			events: []Event{
				{
					Timestamp:    now.Add(-time.Minute),
					Type:         "Normal",
					Reason:       "Created",
					Message:      "Pod created",
					ResourceKind: "Pod",
					ResourceName: "test-pod",
				},
				{
					Timestamp:    now,
					Type:         "Warning",
					Reason:       "Failed",
					Message:      "Container failed to start",
					ResourceKind: "Pod",
					ResourceName: "test-pod",
				},
			},
			wantEmpty: false,
			wantContains: []string{
				"Events:",
				"TYPE",
				"REASON",
				"Normal",
				"Warning",
				"Created",
				"Failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			printer := &pritty.Printer{
				IOStreams: genericiooptions.IOStreams{
					Out: &buf,
				},
				Color: false,
			}

			RenderEvents(printer, tt.events)

			output := buf.String()

			if tt.wantEmpty {
				if output != "" {
					t.Errorf("expected no output for empty events, got: %s", output)
				}
				return
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("output missing expected text %q", want)
				}
			}
		})
	}
}
