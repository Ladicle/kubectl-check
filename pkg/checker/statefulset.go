package checker

import (
	"errors"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func listStatefulSetChild(sts *appsv1.StatefulSet, dc dynamic.Interface) (*unstructured.Unstructured, error) {
	if sts.Status.UpdateRevision == "" {
		return nil, errors.New(".status.updateRevision is empty")
	}

	return ListPods(dc, sts.Namespace, fmt.Sprintf("%s=%s", appsv1.ControllerRevisionHashLabelKey, sts.Status.UpdateRevision))
}

func checkStatefulSet(sts *appsv1.StatefulSet, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error) {
	status := &ResourceStatus{
		Kind:      "StatefulSet",
		Name:      sts.Name,
		Namespace: sts.Namespace,
	}

	// Collect events for this StatefulSet
	events, err := getResourceEvents(c, sts.Namespace, "StatefulSet", sts.Name)
	if err == nil {
		status.Events = events
	}

	// Collect container images
	images := []string{}
	for _, container := range sts.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}

	age := formatAge(sts.CreationTimestamp.Time)
	podsStatus := fmt.Sprintf("%d Ready / %d Updated (of %d desired)",
		sts.Status.ReadyReplicas,
		sts.Status.UpdatedReplicas,
		*sts.Spec.Replicas)

	status.AddField("Pods Status:", podsStatus)
	status.AddField("Images:     ", strings.Join(images, ", "))
	status.AddField("Age:        ", age)

	return status, nil
}
