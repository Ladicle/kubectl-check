package checker

import (
	"errors"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func listReplicaSetChild(rs *appsv1.ReplicaSet, dc dynamic.Interface) (*unstructured.Unstructured, error) {
	tplhash, ok := rs.Labels[appsv1.DefaultDeploymentUniqueLabelKey]
	if !ok {
		return nil, errors.New("ReplicaSet does not have pod-template-hash")
	}

	return ListPods(dc, rs.Namespace, fmt.Sprintf("%s=%s", appsv1.DefaultDeploymentUniqueLabelKey, tplhash))
}

func checkReplicaSet(rs *appsv1.ReplicaSet, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error) {
	status := &ResourceStatus{
		Kind: "ReplicaSet",
		Name: rs.Name,
	}

	// Collect events for this ReplicaSet
	events, err := getResourceEvents(c, rs.Namespace, "ReplicaSet", rs.Name)
	if err == nil {
		status.Events = events
	}

	age := formatAge(rs.CreationTimestamp.Time)
	podsStatus := fmt.Sprintf("%d Ready / %d Current (of %d desired)",
		rs.Status.ReadyReplicas,
		rs.Status.Replicas,
		*rs.Spec.Replicas)

	status.AddField("Pods Status:", podsStatus)
	status.AddField("Age:        ", age)

	return status, nil
}
