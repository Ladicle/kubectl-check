package checker

import (
	"context"
	"errors"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func listStatefulSetChild(sts *appsv1.StatefulSet, dc dynamic.Interface) (*unstructured.Unstructured, error) {
	if sts.Status.UpdateRevision == "" {
		return nil, errors.New(".status.updateRevision is empty")
	}

	podList, err := ListPods(dc, sts.Namespace, fmt.Sprintf("%s=%s", appsv1.ControllerRevisionHashLabelKey, sts.Status.UpdateRevision))
	if err != nil {
		return nil, err
	}

	// Add PVCs from VolumeClaimTemplates
	var allItems []unstructured.Unstructured
	if len(sts.Spec.VolumeClaimTemplates) > 0 && sts.Spec.Replicas != nil {
		pvcGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumeclaims"}
		for _, vct := range sts.Spec.VolumeClaimTemplates {
			// StatefulSet creates PVCs with pattern: <vct-name>-<sts-name>-<index>
			for i := int32(0); i < *sts.Spec.Replicas; i++ {
				pvcName := fmt.Sprintf("%s-%s-%d", vct.Name, sts.Name, i)
				pvc, err := dc.Resource(pvcGVR).Namespace(sts.Namespace).Get(context.Background(), pvcName, metav1.GetOptions{})
				if err == nil {
					allItems = append(allItems, *pvc)
				}
			}
		}
	}

	// Add pods (potentially grouped by workload)
	podsResult, err := GroupPodsByWorkload(dc, podList, sts.Namespace, nil)
	if err != nil {
		return nil, err
	}

	if podsResult != nil {
		if podsResult.IsList() {
			list, _ := podsResult.ToList()
			allItems = append(allItems, list.Items...)
		} else {
			allItems = append(allItems, *podsResult)
		}
	}

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

func checkStatefulSet(sts *appsv1.StatefulSet, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error) {
	status := &ResourceStatus{
		Kind:      "StatefulSet",
		Name:      sts.Name,
		Namespace: sts.Namespace,
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
