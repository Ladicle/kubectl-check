package checker

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// ResourceInfo holds metadata and check functions for a resource kind
type ResourceInfo struct {
	GVR       schema.GroupVersionResource
	Aliases   []string
	CheckFunc func(obj *unstructured.Unstructured, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error)
	ListChild func(obj *unstructured.Unstructured, dc dynamic.Interface) (*unstructured.Unstructured, error)
}

// Resources defines all supported resources with their metadata
var Resources map[string]ResourceInfo

func init() {
	Resources = map[string]ResourceInfo{
		"Deployment": {
			GVR:       schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"},
			Aliases:   []string{"deployments", "deployment", "deploy"},
			CheckFunc: makeCheckFunc(checkDeployment),
			ListChild: makeListChildFunc(listDeploymentChild),
		},
		"ReplicaSet": {
			GVR:       schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"},
			Aliases:   []string{"replicasets", "replicaset", "rs"},
			CheckFunc: makeCheckFunc(checkReplicaSet),
			ListChild: makeListChildFunc(listReplicaSetChild),
		},
		"StatefulSet": {
			GVR:       schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"},
			Aliases:   []string{"statefulsets", "statefulset", "sts"},
			CheckFunc: makeCheckFunc(checkStatefulSet),
			ListChild: makeListChildFunc(listStatefulSetChild),
		},
		"DaemonSet": {
			GVR:       schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"},
			Aliases:   []string{"daemonsets", "daemonset", "ds"},
			CheckFunc: makeCheckFunc(checkDaemonSet),
			ListChild: makeListChildFunc(listDaemonSetChild),
		},
		"Pod": {
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Aliases:   []string{"pods", "pod", "po"},
			CheckFunc: makeCheckFunc(checkPod),
			ListChild: nil,
		},
		"Job": {
			GVR:       schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"},
			Aliases:   []string{"jobs", "job"},
			CheckFunc: makeCheckFunc(checkJob),
			ListChild: makeListChildFunc(listJobChild),
		},
		"Service": {
			GVR:       schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"},
			Aliases:   []string{"services", "service", "svc"},
			CheckFunc: makeCheckFunc(checkService),
			ListChild: nil,
		},
		"CronJob": {
			GVR:       schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "cronjobs"},
			Aliases:   []string{"cronjobs", "cronjob", "cj"},
			CheckFunc: makeCheckFunc(checkCronJob),
			ListChild: makeListChildFunc(listCronJobChild),
		},
		"JobSet": {
			GVR:     schema.GroupVersionResource{Group: "jobset.x-k8s.io", Version: "v1alpha2", Resource: "jobsets"},
			Aliases: []string{"jobsets", "jobset", "js"},
			CheckFunc: func(obj *unstructured.Unstructured, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error) {
				return checkJobSet(obj.Object, c, dc)
			},
			ListChild: func(obj *unstructured.Unstructured, dc dynamic.Interface) (*unstructured.Unstructured, error) {
				return listJobSetChild(obj.Object, dc)
			},
		},
		// Workload is not directly checkable but used as intermediate node in hierarchy
		"Workload": {
			GVR:       schema.GroupVersionResource{Group: "kueue.x-k8s.io", Version: "v1beta1", Resource: "workloads"},
			Aliases:   nil, // No aliases - not directly checkable
			CheckFunc: makeCheckFunc(checkWorkload),
			ListChild: nil, // Children are provided via _childPods field
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
