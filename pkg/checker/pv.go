package checker

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// checkPersistentVolume checks the status of a PersistentVolume
func checkPersistentVolume(pv corev1.PersistentVolume, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error) {
	status := &ResourceStatus{
		Name: pv.Name,
		Kind: "PersistentVolume",
	}

	// Check phase
	phase := pv.Status.Phase
	status.AddField("Status:        ", string(phase))

	// Add claim reference if bound (only when not called from PVC)
	if pv.Spec.ClaimRef != nil {
		status.AddField("Claim:         ", fmt.Sprintf("%s/%s", pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name))
	}

	// Add storage class
	if pv.Spec.StorageClassName != "" {
		status.AddField("StorageClass:  ", pv.Spec.StorageClassName)
	}

	// Add capacity
	if capacity, ok := pv.Spec.Capacity[corev1.ResourceStorage]; ok {
		status.AddField("Capacity:      ", capacity.String())
	}

	// Add access modes
	if len(pv.Spec.AccessModes) > 0 {
		accessModes := ""
		for i, mode := range pv.Spec.AccessModes {
			if i > 0 {
				accessModes += ","
			}
			accessModes += string(mode)
		}
		status.AddField("AccessModes:   ", accessModes)
	}

	// Add reclaim policy
	if pv.Spec.PersistentVolumeReclaimPolicy != "" {
		status.AddField("ReclaimPolicy: ", string(pv.Spec.PersistentVolumeReclaimPolicy))
	}

	// Add age
	age := formatAge(pv.CreationTimestamp.Time)
	status.AddField("Age:           ", age)

	return status, nil
}

// checkPersistentVolumeFromPVC checks PV status with simplified display when shown as PVC child
// Omits fields already shown in parent PVC (StorageClass, Capacity, AccessModes)
func checkPersistentVolumeFromPVC(pv corev1.PersistentVolume, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error) {
	status := &ResourceStatus{
		Name: pv.Name,
		Kind: "PersistentVolume",
	}

	// Check phase - important to show even if parent is Bound, PV might have different status
	phase := pv.Status.Phase
	status.AddField("Status:        ", string(phase))

	// Add reclaim policy - PV-specific information not in PVC
	if pv.Spec.PersistentVolumeReclaimPolicy != "" {
		status.AddField("ReclaimPolicy: ", string(pv.Spec.PersistentVolumeReclaimPolicy))
	}

	// Add age
	age := formatAge(pv.CreationTimestamp.Time)
	status.AddField("Age:           ", age)

	return status, nil
}
