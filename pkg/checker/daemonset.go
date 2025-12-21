package checker

import (
	"errors"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func listDaemonSetChild(ds *appsv1.DaemonSet, dc dynamic.Interface) (*unstructured.Unstructured, error) {
	if ds.Status.ObservedGeneration == 0 {
		return nil, errors.New(".status.observedGeneration is empty")
	}

	// Combine DaemonSet selector with pod-template-generation
	selector := labels.Set(ds.Spec.Selector.MatchLabels).String()
	if selector != "" {
		selector = fmt.Sprintf("%s,pod-template-generation=%d", selector, ds.Status.ObservedGeneration)
	} else {
		selector = fmt.Sprintf("pod-template-generation=%d", ds.Status.ObservedGeneration)
	}

	podList, err := ListPods(dc, ds.Namespace, selector)
	if err != nil {
		return nil, err
	}

	// Group pods by workload if they have kueue annotations
	return GroupPodsByWorkload(dc, podList, ds.Namespace, nil)
}

func checkDaemonSet(ds *appsv1.DaemonSet, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error) {
	status := &ResourceStatus{
		Kind:      "DaemonSet",
		Name:      ds.Name,
		Namespace: ds.Namespace,
	}

	// Collect events for this DaemonSet
	events, err := getResourceEvents(c, ds.Namespace, "DaemonSet", ds.Name)
	if err == nil {
		status.Events = events
	}

	// Collect container images
	images := []string{}
	for _, container := range ds.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}

	age := formatAge(ds.CreationTimestamp.Time)
	podsStatus := fmt.Sprintf("%d Ready / %d Current / %d Available (of %d desired)",
		ds.Status.NumberReady,
		ds.Status.CurrentNumberScheduled,
		ds.Status.NumberAvailable,
		ds.Status.DesiredNumberScheduled)

	status.AddField("Pods Status:", podsStatus)
	status.AddField("Images:     ", strings.Join(images, ", "))
	status.AddField("Age:        ", age)

	return status, nil
}
