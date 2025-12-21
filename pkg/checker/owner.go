package checker

import (
	"context"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/client-go/rest"
)

// ownerChainItem holds both the unstructured object and its status
type ownerChainItem struct {
	obj    *unstructured.Unstructured
	status *ResourceStatus
}

// checkWithOwners checks the resource and recursively checks its owners
func (c *Checker) checkWithOwners(obj *unstructured.Unstructured, resInfo ResourceInfo) (*ResourceStatus, error) {
	// First, check the target resource with all its children
	targetStatus, err := c.checkResource(obj, resInfo)
	if err != nil {
		return nil, err
	}

	// Then, recursively check owners and build the hierarchy from top to bottom
	// This now fetches and checks owners in a single API call per resource
	ownerChain, err := c.collectOwnerChainWithStatus(obj)
	if err != nil {
		return nil, err
	}

	// If there are no owners, return the target status as-is
	if len(ownerChain) == 0 {
		return targetStatus, nil
	}

	// Build the hierarchy from the topmost owner down to the target
	// Status is already created during collection, so just build the tree
	var rootStatus *ResourceStatus
	var currentStatus *ResourceStatus

	for i := len(ownerChain) - 1; i >= 0; i-- {
		ownerStatus := ownerChain[i].status
		if ownerStatus == nil {
			continue
		}

		if rootStatus == nil {
			rootStatus = ownerStatus
			currentStatus = ownerStatus
		} else {
			currentStatus.AddChild(ownerStatus)
			currentStatus = ownerStatus
		}
	}

	// Add the target resource as a child of the immediate owner
	if currentStatus != nil {
		currentStatus.AddChild(targetStatus)
		return rootStatus, nil
	}

	return targetStatus, nil
}

// collectOwnerChainWithStatus recursively collects owner references up to the top
// and creates their ResourceStatus in a single API call per resource
func (c *Checker) collectOwnerChainWithStatus(obj *unstructured.Unstructured) ([]*ownerChainItem, error) {
	var chain []*ownerChainItem

	current := obj
	for current.GetKind() != "Workload" {
		ownerRefs := current.GetOwnerReferences()
		if len(ownerRefs) == 0 {
			break
		}

		// Take the first owner reference (controller)
		var ownerRef *metav1.OwnerReference
		for i := range ownerRefs {
			if ownerRefs[i].Controller != nil && *ownerRefs[i].Controller {
				ownerRef = &ownerRefs[i]
				break
			}
		}
		if ownerRef == nil && len(ownerRefs) > 0 {
			ownerRef = &ownerRefs[0]
		}
		if ownerRef == nil {
			break
		}

		// Get GVR from owner reference's APIVersion and Kind
		gvr, err := parseGVRFromOwnerRef(ownerRef)
		if err != nil {
			break
		}

		// Check if this is a registered resource type
		ownerKind := ownerRef.Kind
		ownerInfo, isRegistered := Resources[ownerKind]

		var ownerObj *unstructured.Unstructured
		var ownerStatus *ResourceStatus

		if isRegistered {
			// For registered resources, fetch using dynamic client
			ownerObj, err = c.DynamicClient.Resource(gvr).Namespace(current.GetNamespace()).Get(
				context.Background(), ownerRef.Name, metav1.GetOptions{})
			if err != nil {
				break
			}

			// Create status using registered CheckFunc
			ownerStatus, err = ownerInfo.CheckFunc(ownerObj, c.Clientset, c.DynamicClient)
			if err != nil {
				break
			}
		} else {
			// For unknown resources, fetch as Table format in a single API call
			ownerObj, ownerStatus, err = c.fetchUnknownResourceWithStatus(gvr, current.GetNamespace(), ownerRef.Name)
			if err != nil {
				break
			}
		}

		chain = append(chain, &ownerChainItem{
			obj:    ownerObj,
			status: ownerStatus,
		})
		current = ownerObj
	}

	return chain, nil
}

// parseGVRFromOwnerRef parses GroupVersionResource from OwnerReference
func parseGVRFromOwnerRef(ownerRef *metav1.OwnerReference) (schema.GroupVersionResource, error) {
	// Parse APIVersion (format: "group/version" or just "version" for core resources)
	gv, err := schema.ParseGroupVersion(ownerRef.APIVersion)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}

	// Convert Kind to resource name (pluralize and lowercase)
	// This is a simple heuristic - for more complex cases, we'd need discovery
	resource := strings.ToLower(ownerRef.Kind) + "s"

	return schema.GroupVersionResource{
		Group:    gv.Group,
		Version:  gv.Version,
		Resource: resource,
	}, nil
}

// fetchUnknownResourceWithStatus fetches an unknown resource as Table format
// and creates both the unstructured object and ResourceStatus in a single API call
func (c *Checker) fetchUnknownResourceWithStatus(gvr schema.GroupVersionResource, namespace, name string) (*unstructured.Unstructured, *ResourceStatus, error) {
	// Create REST client with Table accept header
	restClient := c.Clientset.CoreV1().RESTClient()
	var err error
	if gvr.Group != "" {
		// For non-core resources, use APIPath with group
		restConfig := rest.CopyConfig(c.RESTConfig)
		restConfig.GroupVersion = &schema.GroupVersion{Group: gvr.Group, Version: gvr.Version}
		restConfig.APIPath = "/apis"
		restClient, err = rest.RESTClientFor(restConfig)
		if err != nil {
			return nil, nil, err
		}
	}

	// Request as Table format
	var table metav1.Table
	err = restClient.Get().
		Namespace(namespace).
		Resource(gvr.Resource).
		Name(name).
		SetHeader("Accept", "application/json;as=Table;v=v1;g=meta.k8s.io").
		Do(context.Background()).
		Into(&table)
	if err != nil {
		return nil, nil, err
	}

	if len(table.Rows) == 0 {
		return nil, nil, fmt.Errorf("no rows returned from table")
	}

	// Extract the object from the table row
	row := table.Rows[0]
	obj, ok := row.Object.Object.(*unstructured.Unstructured)
	if !ok {
		return nil, nil, fmt.Errorf("failed to convert row object to unstructured")
	}

	// Create status from table data
	status := &ResourceStatus{
		Kind:      obj.GetKind(),
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}

	// Extract fields from table columns
	for i, col := range table.ColumnDefinitions {
		// Skip Name column (index 0) and Age column as we'll add it separately
		if i == 0 || col.Name == "Age" {
			continue
		}
		// Skip columns with priority > 0 (these are typically for wide output)
		if col.Priority > 0 {
			continue
		}

		if i < len(row.Cells) {
			cellValue := row.Cells[i]
			if cellValue != nil {
				// Format the field
				fieldName := fmt.Sprintf("%-11s", col.Name+":")
				fieldValue := fmt.Sprintf("%v", cellValue)
				if fieldValue != "" && fieldValue != "<nil>" && fieldValue != "<none>" {
					status.AddField(fieldName, fieldValue)
				}
			}
		}
	}

	// Age (always add this last)
	creationTimestamp := obj.GetCreationTimestamp()
	if !creationTimestamp.IsZero() {
		age := duration.HumanDuration(time.Since(creationTimestamp.Time))
		status.AddField("Age:       ", age)
	}

	return obj, status, nil
}
