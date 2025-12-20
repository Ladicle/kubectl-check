package checker

import (
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func checkPod(p *corev1.Pod, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error) {
	status := &ResourceStatus{
		Kind:      "Pod",
		Name:      p.Name,
		Namespace: p.Namespace,
	}

	// Collect events for this Pod
	events, err := getResourceEvents(c, p.Namespace, "Pod", p.Name)
	if err == nil {
		status.Events = events
	}

	ready := p.Status.Phase == corev1.PodRunning
	readyContainers := 0
	totalContainers := len(p.Spec.Containers)
	for _, cond := range p.Status.Conditions {
		if cond.Type == corev1.PodReady {
			ready = ready && cond.Status == corev1.ConditionTrue
			break
		}
	}
	for _, cs := range p.Status.ContainerStatuses {
		if cs.Ready {
			readyContainers++
		}
	}

	// Calculate total restarts
	restarts := int32(0)
	for _, cs := range p.Status.ContainerStatuses {
		restarts += cs.RestartCount
	}

	// Calculate total resources from containers
	totalReq := make(map[corev1.ResourceName]resource.Quantity)
	totalLim := make(map[corev1.ResourceName]resource.Quantity)
	for _, container := range p.Spec.Containers {
		for resourceName, quantity := range container.Resources.Requests {
			if q, ok := totalReq[resourceName]; ok {
				q.Add(quantity)
				totalReq[resourceName] = q
			} else {
				totalReq[resourceName] = quantity.DeepCopy()
			}
		}
		for resourceName, quantity := range container.Resources.Limits {
			if q, ok := totalLim[resourceName]; ok {
				q.Add(quantity)
				totalLim[resourceName] = q
			} else {
				totalLim[resourceName] = quantity.DeepCopy()
			}
		}
	}

	age := formatAge(p.CreationTimestamp.Time)
	readyStatus := fmt.Sprintf("%d/%d (%d restarts)", readyContainers, totalContainers, restarts)

	status.AddField("Ready:   ", readyStatus)
	status.AddField("Status:  ", string(p.Status.Phase))
	status.AddField("Node:    ", p.Spec.NodeName)
	status.AddField("IP:      ", p.Status.PodIP)

	// Display pod-level resources (v1.34 beta)
	podResources := p.Spec.Resources
	hasPodResources := false
	podReq := make(map[corev1.ResourceName]resource.Quantity)
	podLim := make(map[corev1.ResourceName]resource.Quantity)

	if podResources != nil {
		for resourceName, quantity := range podResources.Requests {
			if !quantity.IsZero() {
				podReq[resourceName] = quantity.DeepCopy()
				hasPodResources = true
			}
		}
		for resourceName, quantity := range podResources.Limits {
			if !quantity.IsZero() {
				podLim[resourceName] = quantity.DeepCopy()
				hasPodResources = true
			}
		}
	}

	// Display resources based on what's available
	if hasPodResources {
		// Show pod-level resources (v1.34+)
		status.AddField("Pod Resources:", "")
		if len(podReq) > 0 {
			status.AddField("   Request:", formatResources(podReq))
		}
		if len(podLim) > 0 {
			status.AddField("   Limits: ", formatResources(podLim))
		}
		// Also show container totals for comparison
		if len(totalReq) > 0 || len(totalLim) > 0 {
			status.AddField("Container Resources Total:", "")
			if len(totalReq) > 0 {
				status.AddField("   Request:", formatResources(totalReq))
			}
			if len(totalLim) > 0 {
				status.AddField("   Limits: ", formatResources(totalLim))
			}
		}
	} else {
		// Traditional display: sum of container resources
		if len(totalReq) > 0 || len(totalLim) > 0 {
			status.AddField("Container Resources Total:", "")
			if len(totalReq) > 0 {
				status.AddField("   Request:", formatResources(totalReq))
			}
			if len(totalLim) > 0 {
				status.AddField("   Limits: ", formatResources(totalLim))
			}
		}
	}

	status.AddField("Age:     ", age)

	// Add container information as children
	for _, containerSpec := range p.Spec.Containers {
		containerStatus := getContainerStatus(p, containerSpec.Name)
		status.AddChild(buildContainerStatus(containerSpec, containerStatus))
	}

	// Collect containers for log retrieval
	// Note: We always collect based on ready state here
	// The Checker will decide whether to show all logs or just failed ones
	if !ready {
		status.FailedContainers = collectContainers(c, p, false)
	}

	// Also store all containers for potential use with --show-all-logs
	// Store them in a way that Checker can access if needed
	status.AllContainers = collectContainers(c, p, true)

	return status, nil
}

// getContainerStatus finds the status for a given container name
func getContainerStatus(pod *corev1.Pod, containerName string) *corev1.ContainerStatus {
	for i := range pod.Status.ContainerStatuses {
		if pod.Status.ContainerStatuses[i].Name == containerName {
			return &pod.Status.ContainerStatuses[i]
		}
	}
	return nil
}

// buildContainerStatus creates a ResourceStatus for a container
func buildContainerStatus(spec corev1.Container, status *corev1.ContainerStatus) *ResourceStatus {
	containerStatus := &ResourceStatus{
		Kind: "Container",
		Name: spec.Name,
	}

	if status != nil {
		// Container state
		state := "Unknown"
		reason := ""
		message := ""
		if status.State.Running != nil {
			state = "Running"
		} else if status.State.Waiting != nil {
			state = "Waiting"
			reason = status.State.Waiting.Reason
			message = status.State.Waiting.Message
		} else if status.State.Terminated != nil {
			state = "Terminated"
			reason = fmt.Sprintf("%s (exit code %d)", status.State.Terminated.Reason, status.State.Terminated.ExitCode)
			message = status.State.Terminated.Message
		}

		containerStatus.AddField("State:   ", state)
		if reason != "" {
			containerStatus.AddField("Reason:  ", reason)
		}
		if message != "" {
			containerStatus.AddField("Message: ", message)
		}

		// Restart count
		if status.RestartCount > 0 {
			containerStatus.AddField("Restarts:", fmt.Sprintf("%d", status.RestartCount))
		}
	}

	return containerStatus
}

// collectContainers identifies containers whose logs should be collected
// If showAll is true, collects all containers; otherwise only failed ones
func collectContainers(c kubernetes.Interface, pod *corev1.Pod, showAll bool) []FailedContainer {
	var containers []FailedContainer

	// Check init containers
	for _, cs := range pod.Status.InitContainerStatuses {
		if showAll {
			// Collect all init containers that have started
			if isContainerStarted(cs) {
				containers = append(containers, FailedContainer{
					Namespace:     pod.Namespace,
					PodName:       pod.Name,
					ContainerName: cs.Name,
				})
			}
		} else {
			// Only collect failed init containers (skip successfully completed pods)
			if pod.Status.Phase != corev1.PodSucceeded && !cs.Ready && isContainerStarted(cs) && !isContainerSucceeded(cs) {
				containers = append(containers, FailedContainer{
					Namespace:     pod.Namespace,
					PodName:       pod.Name,
					ContainerName: cs.Name,
				})
			}
		}
	}

	// Check regular containers
	for _, cs := range pod.Status.ContainerStatuses {
		if showAll {
			// Collect all containers that have started
			if isContainerStarted(cs) {
				containers = append(containers, FailedContainer{
					Namespace:     pod.Namespace,
					PodName:       pod.Name,
					ContainerName: cs.Name,
				})
			}
		} else {
			// Only collect failed containers (skip successfully completed pods)
			if pod.Status.Phase != corev1.PodSucceeded && !cs.Ready && isContainerStarted(cs) && !isContainerSucceeded(cs) {
				containers = append(containers, FailedContainer{
					Namespace:     pod.Namespace,
					PodName:       pod.Name,
					ContainerName: cs.Name,
				})
			}
		}
	}

	return containers
}

// isContainerSucceeded checks if a container has successfully completed
func isContainerSucceeded(cs corev1.ContainerStatus) bool {
	return cs.State.Terminated != nil && cs.State.Terminated.ExitCode == 0
}

// isContainerStarted is a function to checks if the current or the last container holds the ContainerID.
// In other words, it is determined whether the container has been started even once.
func isContainerStarted(cs corev1.ContainerStatus) bool {
	switch {
	case cs.ContainerID != "":
		return true
	case cs.State.Terminated != nil && cs.State.Terminated.ContainerID != "":
		return true
	case cs.LastTerminationState.Terminated != nil && cs.LastTerminationState.Terminated.ContainerID != "":
		return true
	}
	return false
}

// shortenResourceName shortens resource names like "nvidia.com/gpu" to "gpu"
func shortenResourceName(name string) string {
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

// formatResources formats resource map into a human-readable string
func formatResources(resources map[corev1.ResourceName]resource.Quantity) string {
	if len(resources) == 0 {
		return ""
	}

	// Sort resource names for consistent output
	names := make([]string, 0, len(resources))
	for name := range resources {
		names = append(names, string(name))
	}
	sort.Strings(names)

	parts := make([]string, 0, len(names))
	for _, name := range names {
		quantity := resources[corev1.ResourceName(name)]
		shortName := shortenResourceName(name)

		// Use Kubernetes' canonical formatting
		formattedValue := quantity.String()

		switch name {
		case "cpu":
			parts = append(parts, fmt.Sprintf("%s cpu", formattedValue))
		case "memory":
			parts = append(parts, fmt.Sprintf("%s mem", formattedValue))
		default:
			parts = append(parts, fmt.Sprintf("%s %s", formattedValue, shortName))
		}
	}

	return strings.Join(parts, ", ")
}
