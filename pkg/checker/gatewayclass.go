package checker

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func checkGatewayClass(gatewayClass map[string]interface{}, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error) {
	name, _, _ := unstructured.NestedString(gatewayClass, "metadata", "name")

	status := &ResourceStatus{
		Kind: "GatewayClass",
		Name: name,
		// GatewayClass is cluster-scoped
	}

	// Collect events for this GatewayClass
	events, err := getResourceEvents(c, "", "GatewayClass", name)
	if err == nil {
		status.Events = events
	}

	// Controller
	controller, found, _ := unstructured.NestedString(gatewayClass, "spec", "controllerName")
	if found {
		status.AddField("Controller:", controller)
	}

	// ParametersRef
	paramsRef, found, _ := unstructured.NestedMap(gatewayClass, "spec", "parametersRef")
	if found {
		group, _, _ := unstructured.NestedString(paramsRef, "group")
		kind, _, _ := unstructured.NestedString(paramsRef, "kind")
		paramName, _, _ := unstructured.NestedString(paramsRef, "name")
		namespace, foundNs, _ := unstructured.NestedString(paramsRef, "namespace")

		paramStr := fmt.Sprintf("%s.%s/%s", kind, group, paramName)
		if foundNs {
			paramStr = fmt.Sprintf("%s.%s/%s/%s", kind, group, namespace, paramName)
		}
		status.AddField("Parameters:", paramStr)
	}

	// Description
	description, found, _ := unstructured.NestedString(gatewayClass, "spec", "description")
	if found && description != "" {
		status.AddField("Desc:      ", description)
	}

	// Status conditions
	conditions, found, _ := unstructured.NestedSlice(gatewayClass, "status", "conditions")
	if found && len(conditions) > 0 {
		condStrs := make([]string, 0)
		for _, cond := range conditions {
			condMap, ok := cond.(map[string]interface{})
			if !ok {
				continue
			}
			condType, _, _ := unstructured.NestedString(condMap, "type")
			condStatus, _, _ := unstructured.NestedString(condMap, "status")
			reason, _, _ := unstructured.NestedString(condMap, "reason")

			condStr := fmt.Sprintf("%s=%s", condType, condStatus)
			if reason != "" {
				condStr += fmt.Sprintf(" (%s)", reason)
			}
			condStrs = append(condStrs, condStr)
		}
		if len(condStrs) > 0 {
			status.AddField("Conditions:", strings.Join(condStrs, ", "))
		}
	}

	return status, nil
}
