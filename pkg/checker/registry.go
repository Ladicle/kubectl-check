package checker

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// ResourceInfo holds metadata and check functions for a resource kind
type ResourceInfo struct {
	GVR        schema.GroupVersionResource
	Aliases    []string
	Namespaced bool // true if the resource is namespaced, false for cluster-scoped
	CheckFunc  func(obj *unstructured.Unstructured, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error)
	ListChild  func(obj *unstructured.Unstructured, dc dynamic.Interface) (*unstructured.Unstructured, error)
}

// Resources defines all supported resources with their metadata
var Resources map[string]ResourceInfo

func init() {
	Resources = map[string]ResourceInfo{
		"Deployment": {
			GVR:        schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"},
			Aliases:    []string{"deployments", "deployment", "deploy"},
			Namespaced: true,
			CheckFunc:  makeCheckFunc(checkDeployment),
			ListChild:  makeListChildFunc(listDeploymentChild),
		},
		"ReplicaSet": {
			GVR:        schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"},
			Aliases:    []string{"replicasets", "replicaset", "rs"},
			Namespaced: true,
			CheckFunc:  makeCheckFunc(checkReplicaSet),
			ListChild:  makeListChildFunc(listReplicaSetChild),
		},
		"StatefulSet": {
			GVR:        schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"},
			Aliases:    []string{"statefulsets", "statefulset", "sts"},
			Namespaced: true,
			CheckFunc:  makeCheckFunc(checkStatefulSet),
			ListChild:  makeListChildFunc(listStatefulSetChild),
		},
		"DaemonSet": {
			GVR:        schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"},
			Aliases:    []string{"daemonsets", "daemonset", "ds"},
			Namespaced: true,
			CheckFunc:  makeCheckFunc(checkDaemonSet),
			ListChild:  makeListChildFunc(listDaemonSetChild),
		},
		"Pod": {
			GVR:        schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Aliases:    []string{"pods", "pod", "po"},
			Namespaced: true,
			CheckFunc:  makeCheckFunc(checkPod),
			ListChild:  nil,
		},
		"PersistentVolumeClaim": {
			GVR:        schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
			Aliases:    []string{"persistentvolumeclaims", "persistentvolumeclaim", "pvc"},
			Namespaced: true,
			CheckFunc:  makeCheckFunc(checkPersistentVolumeClaim),
			ListChild:  makeListChildFunc(listPersistentVolumeClaimChild),
		},
		"PersistentVolume": {
			GVR:        schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumes"},
			Aliases:    []string{"persistentvolumes", "persistentvolume", "pv"},
			Namespaced: false,
			CheckFunc:  makePVCheckFunc(),
			ListChild:  nil,
		},
		"Job": {
			GVR:        schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"},
			Aliases:    []string{"jobs", "job"},
			Namespaced: true,
			CheckFunc:  makeCheckFunc(checkJob),
			ListChild:  makeListChildFunc(listJobChild),
		},
		"Service": {
			GVR:        schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"},
			Aliases:    []string{"services", "service", "svc"},
			Namespaced: true,
			CheckFunc:  makeCheckFunc(checkService),
			ListChild:  nil,
		},
		"CronJob": {
			GVR:        schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"},
			Aliases:    []string{"cronjobs", "cronjob", "cj"},
			Namespaced: true,
			CheckFunc:  makeCheckFunc(checkCronJob),
			ListChild:  makeListChildFunc(listCronJobChild),
		},
		"JobSet": {
			GVR:        schema.GroupVersionResource{Group: "jobset.x-k8s.io", Version: "v1alpha2", Resource: "jobsets"},
			Aliases:    []string{"jobsets", "jobset", "js"},
			Namespaced: true,
			CheckFunc:  makeUnstructuredCheckFunc(checkJobSet),
			ListChild:  makeUnstructuredListChildFunc(listJobSetChild),
		},
		// Workload is not directly checkable but used as intermediate node in hierarchy
		"Workload": {
			GVR:        schema.GroupVersionResource{Group: "kueue.x-k8s.io", Version: "v1beta1", Resource: "workloads"},
			Aliases:    nil, // No aliases - not directly checkable
			Namespaced: true,
			CheckFunc:  makeCheckFunc(checkWorkload),
			ListChild:  nil, // Children are provided via _childPods field
		},
		// Gateway API resources
		"Gateway": {
			GVR:        schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gateways"},
			Aliases:    []string{"gateways", "gateway", "gtw"},
			Namespaced: true,
			CheckFunc:  makeUnstructuredCheckFunc(checkGateway),
			ListChild:  makeUnstructuredListChildFunc(listGatewayChild),
		},
		"HTTPRoute": {
			GVR:        schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "httproutes"},
			Aliases:    []string{"httproutes", "httproute"},
			Namespaced: true,
			CheckFunc:  makeUnstructuredCheckFunc(checkHTTPRoute),
			ListChild:  makeUnstructuredListChildFunc(listHTTPRouteChild),
		},
		"GRPCRoute": {
			GVR:        schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "grpcroutes"},
			Aliases:    []string{"grpcroutes", "grpcroute"},
			Namespaced: true,
			CheckFunc:  makeUnstructuredCheckFunc(checkGRPCRoute),
			ListChild:  makeUnstructuredListChildFunc(listGRPCRouteChild),
		},
	}
}

// makeCheckFunc converts a typed check function to unstructured
func makeCheckFunc[T any](fn func(T, kubernetes.Interface, dynamic.Interface) (*ResourceStatus, error)) func(*unstructured.Unstructured, kubernetes.Interface, dynamic.Interface) (*ResourceStatus, error) {
	return func(obj *unstructured.Unstructured, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error) {
		var t T
		if err := unstructuredToTyped(obj, &t); err != nil {
			return nil, err
		}
		return fn(t, c, dc)
	}
}

// makeListChildFunc converts a typed listChild function to unstructured
func makeListChildFunc[T any](fn func(T, dynamic.Interface) (*unstructured.Unstructured, error)) func(*unstructured.Unstructured, dynamic.Interface) (*unstructured.Unstructured, error) {
	return func(obj *unstructured.Unstructured, dc dynamic.Interface) (*unstructured.Unstructured, error) {
		var t T
		if err := unstructuredToTyped(obj, &t); err != nil {
			return nil, err
		}
		return fn(t, dc)
	}
}

// makeUnstructuredCheckFunc wraps a check function that works with map[string]interface{}
// This is useful for CRDs where we don't have typed structs
func makeUnstructuredCheckFunc(fn func(map[string]interface{}, kubernetes.Interface, dynamic.Interface) (*ResourceStatus, error)) func(*unstructured.Unstructured, kubernetes.Interface, dynamic.Interface) (*ResourceStatus, error) {
	return func(obj *unstructured.Unstructured, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error) {
		return fn(obj.Object, c, dc)
	}
}

// makeUnstructuredListChildFunc wraps a listChild function that works with map[string]interface{}
// This is useful for CRDs where we don't have typed structs
func makeUnstructuredListChildFunc(fn func(map[string]interface{}, dynamic.Interface) (*unstructured.Unstructured, error)) func(*unstructured.Unstructured, dynamic.Interface) (*unstructured.Unstructured, error) {
	return func(obj *unstructured.Unstructured, dc dynamic.Interface) (*unstructured.Unstructured, error) {
		return fn(obj.Object, dc)
	}
}

// makePVCheckFunc creates a special check function for PersistentVolume that detects
// if it's being called from PVC (via _fromPVC flag) and uses simplified display
func makePVCheckFunc() func(*unstructured.Unstructured, kubernetes.Interface, dynamic.Interface) (*ResourceStatus, error) {
	return func(obj *unstructured.Unstructured, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error) {
		var pv corev1.PersistentVolume
		if err := unstructuredToTyped(obj, &pv); err != nil {
			return nil, err
		}

		// Check if this PV is being displayed as a child of PVC
		if fromPVC, ok := obj.Object["_fromPVC"].(bool); ok && fromPVC {
			return checkPersistentVolumeFromPVC(pv, c, dc)
		}

		return checkPersistentVolume(pv, c, dc)
	}
}
