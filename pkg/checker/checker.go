package checker

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/Ladicle/kubectl-check/pkg/client"
	"github.com/Ladicle/kubectl-check/pkg/pritty"
)

// formatAge returns a human-readable age string
func formatAge(t time.Time) string {
	return duration.HumanDuration(time.Since(t))
}

// formatTimestampWithAge returns a timestamp with timezone and relative age
// Example: "2025-12-20 10:30:45 JST (11s ago)"
func formatTimestampWithAge(t time.Time) string {
	// Format: "2006-01-02 15:04:05 MST"
	timestamp := t.Local().Format("2006-01-02 15:04:05 MST")
	age := duration.HumanDuration(time.Since(t))
	return fmt.Sprintf("%s (%s ago)", timestamp, age)
}

// formatBool returns a title case string for boolean values
// Example: true -> "True", false -> "False"
func formatBool(b bool) string {
	if b {
		return "True"
	}
	return "False"
}

// getResourceEvents retrieves events for a specific resource
func getResourceEvents(c kubernetes.Interface, namespace, kind, name string) ([]Event, error) {
	eventList, err := c.CoreV1().Events(namespace).List(context.Background(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.kind=%s,involvedObject.name=%s", kind, name),
	})
	if err != nil {
		return nil, err
	}

	events := make([]Event, 0, len(eventList.Items))
	for _, e := range eventList.Items {
		// Build From field from source
		from := e.Source.Component
		if e.Source.Host != "" {
			from = fmt.Sprintf("%s, %s", e.Source.Component, e.Source.Host)
		}

		// Build Object field
		object := fmt.Sprintf("%s/%s", e.InvolvedObject.Kind, e.InvolvedObject.Name)
		if e.InvolvedObject.FieldPath != "" {
			object = fmt.Sprintf("%s/%s", object, e.InvolvedObject.FieldPath)
		}

		events = append(events, Event{
			Timestamp:    e.LastTimestamp.Time,
			Type:         e.Type,
			Reason:       e.Reason,
			Message:      e.Message,
			ResourceKind: kind,
			ResourceName: name,
			From:         from,
			Object:       object,
		})
	}
	return events, nil
}

// Config holds configuration for checking a target resource.
type Config struct {
	Target        types.NamespacedName
	ClientManager *client.Manager

	// Log display options
	ShowAllLogs  bool  // Show logs for all containers, not just failed ones
	LogTailLines int64 // Number of log lines to display

	// Owner reference checking
	CheckOwner bool // Recursively check and display owner resources
}

// Checker checks resources using a hierarchical approach
type Checker struct {
	*Config
	Kind string
}

// NewChecker creates a new Checker
func NewChecker(cfg *Config, kind string) *Checker {
	return &Checker{
		Config: cfg,
		Kind:   kind,
	}
}

// unstructuredToTyped converts unstructured to typed object
func unstructuredToTyped(obj *unstructured.Unstructured, target interface{}) error {
	return runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, target)
}

// ListPods lists pods using label selector and returns as unstructured object
func ListPods(dc dynamic.Interface, namespace, labelSelector string) (*unstructured.Unstructured, error) {
	podInfo, ok := Resources["Pod"]
	if !ok {
		return nil, fmt.Errorf("could not resolve Pod resource")
	}
	podList, err := dc.Resource(podInfo.GVR).Namespace(namespace).List(
		context.Background(),
		metav1.ListOptions{LabelSelector: labelSelector},
	)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: podList.UnstructuredContent()}, nil
}

func (c *Checker) Check(printer *pritty.Printer) error {
	resInfo, ok := Resources[c.Kind]
	if !ok {
		return fmt.Errorf("unsupported resource kind: %s", c.Kind)
	}

	// Get clients from manager
	dc := c.ClientManager.DynamicClient

	// Get the resource using dynamic client
	unstructuredObj, err := dc.Resource(resInfo.GVR).Namespace(c.Target.Namespace).Get(
		context.Background(), c.Target.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Check owner references if CheckOwner is enabled
	var rootStatus *ResourceStatus
	if c.CheckOwner {
		rootStatus, err = c.checkWithOwners(unstructuredObj, resInfo)
		if err != nil {
			return err
		}
	} else {
		// Check the resource recursively and collect status
		rootStatus, err = c.checkResource(unstructuredObj, resInfo)
		if err != nil {
			return err
		}
	}

	// Render the status tree
	rootStatus.Render(printer, 0)

	// Collect and render all events in hierarchical order
	allEvents := rootStatus.CollectAllEvents()
	RenderEvents(printer, allEvents)

	// Output logs for containers
	var containersForLogs []FailedContainer
	if c.ShowAllLogs {
		containersForLogs = rootStatus.CollectAllContainersFromPods()
	} else {
		containersForLogs = rootStatus.CollectAllFailedContainers()
	}

	if len(containersForLogs) > 0 {
		header := "Failed Container Logs:"
		if c.ShowAllLogs {
			header = "Container Logs:"
		}
		coloredHeader := pritty.Style(header).Fg(pritty.Cyan).Decorate(pritty.Bold).String()
		_, _ = fmt.Fprint(printer.IOStreams.Out, "\n"+coloredHeader)
		for _, fc := range containersForLogs {
			c.printContainerLogs(printer, fc)
		}
	}

	return nil
}

func (c *Checker) checkResource(obj *unstructured.Unstructured, resInfo ResourceInfo) (*ResourceStatus, error) {
	// Get clients from manager
	clientset := c.ClientManager.Clientset
	dc := c.ClientManager.DynamicClient

	// Check the resource and get its status
	status, err := resInfo.CheckFunc(obj, clientset, dc)
	if err != nil {
		return nil, err
	}

	// Automatically collect events for all resources
	if len(status.Events) == 0 {
		events, err := getResourceEvents(clientset, obj.GetNamespace(), obj.GetKind(), obj.GetName())
		if err == nil {
			status.Events = events
		}
	}

	// Special handling for Workload with _childPods field
	if obj.GetKind() == "Workload" {
		if childPods, found, _ := unstructured.NestedSlice(obj.Object, "_childPods"); found && len(childPods) > 0 {
			// Process child pods
			podInfo, ok := Resources["Pod"]
			if ok {
				for _, podObj := range childPods {
					podMap, ok := podObj.(map[string]interface{})
					if !ok {
						continue
					}
					podUnstructured := &unstructured.Unstructured{Object: podMap}
					childStatus, err := c.checkResource(podUnstructured, podInfo)
					if err != nil {
						return nil, err
					}
					status.AddChild(childStatus)
				}
			}
			return status, nil
		}
	}

	// Check children if ListChild is defined
	if resInfo.ListChild != nil {
		childObj, err := resInfo.ListChild(obj, dc)
		if err != nil {
			return nil, fmt.Errorf("failed to list children: %w", err)
		}
		if childObj != nil {
			var children []unstructured.Unstructured
			if childObj.IsList() {
				list, err := childObj.ToList()
				if err != nil {
					return nil, fmt.Errorf("failed to convert to list: %w", err)
				}
				children = list.Items
			} else {
				children = []unstructured.Unstructured{*childObj}
			}

			if len(children) > 0 {
				childKind := children[0].GetKind()
				childInfo, ok := Resources[childKind]
				if !ok {
					return nil, fmt.Errorf("unsupported child kind: %s", childKind)
				}
				for i := range children {
					// Recursively check each child
					childStatus, err := c.checkResource(&children[i], childInfo)
					if err != nil {
						return nil, err
					}
					status.AddChild(childStatus)
				}
			}
		}
	}

	return status, nil
}

// printContainerLogs prints logs for a container
func (c *Checker) printContainerLogs(printer *pritty.Printer, fc FailedContainer) {
	header := fmt.Sprintf("Pod/%s/%s", fc.PodName, fc.ContainerName)
	coloredHeader := pritty.Style(header).Decorate(pritty.Bold).String()
	separator := pritty.Style(strings.Repeat("-", 80)).Decorate(pritty.Bold).String()
	_, _ = fmt.Fprintf(printer.IOStreams.Out, "\n%s\n", coloredHeader)
	_, _ = fmt.Fprintf(printer.IOStreams.Out, "%s\n", separator)

	clientset := c.ClientManager.Clientset

	tailLines := c.LogTailLines
	if tailLines <= 0 {
		tailLines = 100 // default
	}
	req := clientset.CoreV1().Pods(fc.Namespace).GetLogs(fc.PodName, &corev1.PodLogOptions{
		Container: fc.ContainerName,
		TailLines: &tailLines,
	})

	logs, err := req.DoRaw(context.Background())
	if err != nil {
		_, _ = fmt.Fprintf(printer.IOStreams.Out, "Error retrieving logs: %v\n", err)
		return
	}

	_, _ = fmt.Fprintf(printer.IOStreams.Out, "%s\n", string(logs))
}
