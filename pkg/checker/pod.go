package checker

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func checkPod(p *corev1.Pod, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error) {
	status := &ResourceStatus{
		Kind: "Pod",
		Name: p.Name,
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

	// Calculate total resources
	var totalCPUReq, totalMemReq, totalCPULim, totalMemLim int64
	for _, container := range p.Spec.Containers {
		if cpu := container.Resources.Requests.Cpu(); cpu != nil {
			totalCPUReq += cpu.MilliValue()
		}
		if mem := container.Resources.Requests.Memory(); mem != nil {
			totalMemReq += mem.Value()
		}
		if cpu := container.Resources.Limits.Cpu(); cpu != nil {
			totalCPULim += cpu.MilliValue()
		}
		if mem := container.Resources.Limits.Memory(); mem != nil {
			totalMemLim += mem.Value()
		}
	}

	age := formatAge(p.CreationTimestamp.Time)
	readyStatus := fmt.Sprintf("%d/%d", readyContainers, totalContainers)

	status.AddField("Ready:   ", readyStatus)
	status.AddField("Status:  ", string(p.Status.Phase))
	status.AddField("Restarts:", fmt.Sprintf("%d", restarts))
	status.AddField("Age:     ", age)
	status.AddField("IP:      ", p.Status.PodIP)
	status.AddField("Node:    ", p.Spec.NodeName)

	// Display resources
	if totalCPUReq > 0 || totalMemReq > 0 {
		status.AddField("Requests:", fmt.Sprintf("%dm CPU, %dMi Memory", totalCPUReq, totalMemReq/(1024*1024)))
	}
	if totalCPULim > 0 || totalMemLim > 0 {
		status.AddField("Limits:  ", fmt.Sprintf("%dm CPU, %dMi Memory", totalCPULim, totalMemLim/(1024*1024)))
	}

	// Collect failed containers for log retrieval
	if !ready {
		status.FailedContainers = collectFailedContainers(c, p)
	}

	return status, nil
}

// collectFailedContainers identifies containers that are not ready and should have their logs collected
func collectFailedContainers(c kubernetes.Interface, pod *corev1.Pod) []FailedContainer {
	var failed []FailedContainer

	// Check init containers
	for _, cs := range pod.Status.InitContainerStatuses {
		if !cs.Ready && isContainerStarted(cs) {
			failed = append(failed, FailedContainer{
				Namespace:     pod.Namespace,
				PodName:       pod.Name,
				ContainerName: cs.Name,
			})
		}
	}

	// Check regular containers
	for _, cs := range pod.Status.ContainerStatuses {
		if !cs.Ready && isContainerStarted(cs) {
			failed = append(failed, FailedContainer{
				Namespace:     pod.Namespace,
				PodName:       pod.Name,
				ContainerName: cs.Name,
			})
		}
	}

	return failed
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
