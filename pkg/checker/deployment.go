package checker

import (
	"context"
	"errors"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func listDeploymentChild(deploy *appsv1.Deployment, dc dynamic.Interface) (*unstructured.Unstructured, error) {
	var allItems []unstructured.Unstructured

	// Collect PVCs referenced in the pod template
	pvcNames := make(map[string]bool)
	for _, vol := range deploy.Spec.Template.Spec.Volumes {
		if vol.PersistentVolumeClaim != nil {
			pvcNames[vol.PersistentVolumeClaim.ClaimName] = true
		}
	}

	// Fetch PVCs
	if len(pvcNames) > 0 {
		pvcGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumeclaims"}
		for pvcName := range pvcNames {
			pvc, err := dc.Resource(pvcGVR).Namespace(deploy.Namespace).Get(context.Background(), pvcName, metav1.GetOptions{})
			if err == nil {
				allItems = append(allItems, *pvc)
			}
		}
	}

	// List ReplicaSets using dynamic client
	rsInfo, ok := Resources["ReplicaSet"]
	if !ok {
		return nil, fmt.Errorf("could not resolve ReplicaSet resource")
	}
	rsList, err := dc.Resource(rsInfo.GVR).Namespace(deploy.Namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labels.Set(deploy.Spec.Selector.MatchLabels).String(),
	})
	if err != nil {
		return nil, err
	}
	if len(rsList.Items) == 0 {
		return nil, errors.New("not found ReplicaSet")
	}

	// Find the latest ReplicaSet
	latestRS := &rsList.Items[0]
	for i := range rsList.Items[1:] {
		rs := &rsList.Items[i+1]
		var latestGen, rsGen int64
		latestGen, _, _ = unstructured.NestedInt64(latestRS.Object, "status", "observedGeneration")
		rsGen, _, _ = unstructured.NestedInt64(rs.Object, "status", "observedGeneration")
		if latestGen < rsGen {
			latestRS = rs
		}
	}

	// Mark the ReplicaSet to skip PVC collection
	latestRS.Object["_skipPVCCollection"] = true
	allItems = append(allItems, *latestRS)

	if len(allItems) == 0 {
		return nil, nil
	}

	// Convert allItems to raw objects for the list
	items := make([]interface{}, len(allItems))
	for i := range allItems {
		items[i] = allItems[i].Object
	}

	// Convert to Unstructured with list content
	result := &unstructured.Unstructured{}
	result.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "List",
		"items":      items,
	})
	return result, nil
}

func checkDeployment(deploy *appsv1.Deployment, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error) {
	status := &ResourceStatus{
		Kind:      "Deployment",
		Name:      deploy.Name,
		Namespace: deploy.Namespace,
	}

	// Collect container images
	images := []string{}
	for _, container := range deploy.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}

	age := formatAge(deploy.CreationTimestamp.Time)
	podsStatus := fmt.Sprintf("%d Ready / %d Up-to-date / %d Available (of %d desired)",
		deploy.Status.ReadyReplicas,
		deploy.Status.UpdatedReplicas,
		deploy.Status.AvailableReplicas,
		*deploy.Spec.Replicas)

	status.AddField("Pods Status:", podsStatus)
	status.AddField("Images:     ", strings.Join(images, ", "))
	status.AddField("Age:        ", age)

	return status, nil
}
