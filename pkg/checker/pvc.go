package checker

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// checkPersistentVolumeClaim checks the status of a PersistentVolumeClaim
func checkPersistentVolumeClaim(pvc corev1.PersistentVolumeClaim, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error) {
	status := &ResourceStatus{
		Name:      pvc.Name,
		Namespace: pvc.Namespace,
		Kind:      "PersistentVolumeClaim",
	}

	// Check phase
	phase := pvc.Status.Phase
	status.AddField("Status:       ", string(phase))

	// Add storage class if specified
	if pvc.Spec.StorageClassName != nil && *pvc.Spec.StorageClassName != "" {
		status.AddField("StorageClass: ", *pvc.Spec.StorageClassName)
	}

	// Add access modes
	if len(pvc.Status.AccessModes) > 0 {
		accessModes := ""
		for i, mode := range pvc.Status.AccessModes {
			if i > 0 {
				accessModes += ","
			}
			accessModes += string(mode)
		}
		status.AddField("AccessModes:  ", accessModes)
	}

	// Add capacity
	if capacity, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
		status.AddField("Capacity:     ", capacity.String())
	}

	// Add Age
	age := formatAge(pvc.CreationTimestamp.Time)
	status.AddField("Age:          ", age)

	return status, nil
}

// listPersistentVolumeClaimChild returns the bound PersistentVolume as a child
func listPersistentVolumeClaimChild(pvc corev1.PersistentVolumeClaim, dc dynamic.Interface) (*unstructured.Unstructured, error) {
	// If the PVC is bound to a PV, return it as a child
	if pvc.Spec.VolumeName != "" {
		pvGVR := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumes"}
		pv, err := dc.Resource(pvGVR).Get(context.Background(), pvc.Spec.VolumeName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		// Mark PV as being fetched from PVC to simplify display
		if pv.Object == nil {
			pv.Object = make(map[string]interface{})
		}
		pv.Object["_fromPVC"] = true
		return pv, nil
	}

	return nil, nil
}
