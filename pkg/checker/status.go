package checker

import (
	"fmt"
	"strings"
	"time"

	"github.com/Ladicle/kubectl-check/pkg/pritty"
)

// ResourceStatus holds hierarchical status information for a resource
type ResourceStatus struct {
	Kind             string
	Name             string
	Namespace        string
	Fields           []StatusField
	Children         []*ResourceStatus
	Events           []Event
	FailedContainers []FailedContainer
}

// StatusField represents a key-value pair for status display
type StatusField struct {
	Key   string
	Value string
}

// Event represents a Kubernetes event
type Event struct {
	Timestamp    time.Time
	Type         string
	Reason       string
	Message      string
	ResourceKind string
	ResourceName string
	From         string // Source component (e.g., kubelet, scheduler)
	Object       string // Full object reference (e.g., Pod/name/spec.containers{name})
}

// FailedContainer holds information about a failed container for log retrieval
type FailedContainer struct {
	Namespace     string
	PodName       string
	ContainerName string
}

// AddField adds a status field
func (rs *ResourceStatus) AddField(key, value string) {
	rs.Fields = append(rs.Fields, StatusField{Key: key, Value: value})
}

// AddChild adds a child resource status
func (rs *ResourceStatus) AddChild(child *ResourceStatus) {
	rs.Children = append(rs.Children, child)
}

// AddEvent adds an event
func (rs *ResourceStatus) AddEvent(event Event) {
	rs.Events = append(rs.Events, event)
}

// CollectAllEvents recursively collects all events from this status and its children
func (rs *ResourceStatus) CollectAllEvents() []Event {
	var allEvents []Event
	allEvents = append(allEvents, rs.Events...)
	for _, child := range rs.Children {
		allEvents = append(allEvents, child.CollectAllEvents()...)
	}
	return allEvents
}

// CollectAllFailedContainers recursively collects all failed containers from this status and its children
func (rs *ResourceStatus) CollectAllFailedContainers() []FailedContainer {
	var allFailed []FailedContainer
	allFailed = append(allFailed, rs.FailedContainers...)
	for _, child := range rs.Children {
		allFailed = append(allFailed, child.CollectAllFailedContainers()...)
	}
	return allFailed
}

// Render renders the resource status hierarchically
func (rs *ResourceStatus) Render(printer *pritty.Printer, indent int) {
	prefix := strings.Repeat(" ", indent)
	resourceHeader := fmt.Sprintf("%s/%s", rs.Kind, rs.Name)
	coloredHeader := pritty.Style(resourceHeader).Fg(pritty.Cyan).Decorate(pritty.Bold).String()

	// Print resource header
	if rs.Namespace != "" && indent == 0 {
		// Top-level resource with namespace
		_, _ = fmt.Fprintf(printer.IOStreams.Out, "%s%s\n", prefix, coloredHeader)
		_, _ = fmt.Fprintf(printer.IOStreams.Out, "%s  Namespace:   %s\n", prefix, rs.Namespace)
	} else {
		// Child resource or resource without namespace
		_, _ = fmt.Fprintf(printer.IOStreams.Out, "%s%s\n", prefix, coloredHeader)
	}

	// Print fields
	for _, field := range rs.Fields {
		_, _ = fmt.Fprintf(printer.IOStreams.Out, "%s  %s %s\n", prefix, field.Key, field.Value)
	}

	// Print children
	for _, child := range rs.Children {
		child.Render(printer, indent+2)
	}
}

// RenderEvents renders all events in hierarchical order (parent first, then children)
func RenderEvents(printer *pritty.Printer, events []Event) {
	if len(events) == 0 {
		return
	}

	eventsHeader := pritty.Style("Events:").Fg(pritty.Cyan).Decorate(pritty.Bold).String()
	_, _ = fmt.Fprintf(printer.IOStreams.Out, "\n%s\n", eventsHeader)

	tableHeader := pritty.Style(fmt.Sprintf("  %-19s %-7s %-12s %-50s %s",
		"Reason", "Age", "From", "Object", "Message")).Decorate(pritty.Bold).String()
	_, _ = fmt.Fprintln(printer.IOStreams.Out, tableHeader)

	separator := fmt.Sprintf("  %-19s %-7s %-12s %-50s %s",
		"------", "----", "----", "------", "-------")
	_, _ = fmt.Fprintln(printer.IOStreams.Out, separator)

	for _, event := range events {
		age := formatAge(event.Timestamp)

		// Format From field: truncate if too long and convert to CamelCase
		from := toCamelCase(event.From)
		if len(from) > 12 {
			from = from[:10] + ".."
		}

		// Format reason with padding first, then apply color if Warning
		reason := fmt.Sprintf("%-19s", event.Reason)
		if event.Type == "Warning" {
			reason = pritty.Style(reason).Fg(pritty.Red).String()
		}

		_, _ = fmt.Fprintf(printer.IOStreams.Out, "  %s %-7s %-12s %-50s %s\n",
			reason, age, from, event.Object, event.Message)
	}
}

// toCamelCase converts strings like "kubelet, kind-control-plane" to "Kubelet, KindControlPlane"
func toCamelCase(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == '-' || r == '_'
	})
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, "")
}
