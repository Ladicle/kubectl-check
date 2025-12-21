package checker

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func listGRPCRouteChild(grpcRoute map[string]interface{}, dc dynamic.Interface) (*unstructured.Unstructured, error) {
	namespace, _, err := unstructured.NestedString(grpcRoute, "metadata", "namespace")
	if err != nil {
		return nil, err
	}

	// Get all backend services referenced in rules
	rules, found, err := unstructured.NestedSlice(grpcRoute, "spec", "rules")
	if err != nil || !found || len(rules) == 0 {
		return nil, nil
	}

	serviceInfo, ok := Resources["Service"]
	if !ok {
		return nil, fmt.Errorf("could not resolve Service resource")
	}

	serviceNames := make(map[string]bool)
	for _, rule := range rules {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			continue
		}
		backendRefs, found, _ := unstructured.NestedSlice(ruleMap, "backendRefs")
		if !found {
			continue
		}
		for _, ref := range backendRefs {
			refMap, ok := ref.(map[string]interface{})
			if !ok {
				continue
			}
			// Default kind is Service
			kind, _, _ := unstructured.NestedString(refMap, "kind")
			if kind == "" || kind == "Service" {
				name, found, _ := unstructured.NestedString(refMap, "name")
				if found {
					serviceNames[name] = true
				}
			}
		}
	}

	if len(serviceNames) == 0 {
		return nil, nil
	}

	// Fetch all referenced services
	items := make([]interface{}, 0)
	for serviceName := range serviceNames {
		service, err := dc.Resource(serviceInfo.GVR).Namespace(namespace).Get(
			context.Background(),
			serviceName,
			metav1.GetOptions{},
		)
		if err == nil {
			items = append(items, service.Object)
		}
	}

	if len(items) == 0 {
		return nil, nil
	}

	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "List",
			"items":      items,
		},
	}, nil
}

func checkGRPCRoute(grpcRoute map[string]interface{}, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error) {
	namespace, _, _ := unstructured.NestedString(grpcRoute, "metadata", "namespace")
	name, _, _ := unstructured.NestedString(grpcRoute, "metadata", "name")

	status := &ResourceStatus{
		Kind:      "GRPCRoute",
		Name:      name,
		Namespace: namespace,
	}

	// Parent references (Gateways)
	parentRefs, found, _ := unstructured.NestedSlice(grpcRoute, "spec", "parentRefs")
	if found && len(parentRefs) > 0 {
		parentStrs := make([]string, 0, len(parentRefs))
		for _, ref := range parentRefs {
			refMap, ok := ref.(map[string]interface{})
			if !ok {
				continue
			}
			refName, _, _ := unstructured.NestedString(refMap, "name")
			refNamespace, found, _ := unstructured.NestedString(refMap, "namespace")
			if found && refNamespace != namespace {
				parentStrs = append(parentStrs, fmt.Sprintf("%s/%s", refNamespace, refName))
			} else {
				parentStrs = append(parentStrs, refName)
			}
		}
		status.AddField("Parents:   ", strings.Join(parentStrs, ", "))
	}

	// Hostnames
	hostnames, found, _ := unstructured.NestedStringSlice(grpcRoute, "spec", "hostnames")
	if found && len(hostnames) > 0 {
		status.AddField("Hostnames: ", strings.Join(hostnames, ", "))
	}

	// Rules count
	rules, found, _ := unstructured.NestedSlice(grpcRoute, "spec", "rules")
	if found && len(rules) > 0 {
		status.AddField("Rules:     ", fmt.Sprintf("%d", len(rules)))

		// Backend services
		backendServices := make(map[string]bool)
		for _, rule := range rules {
			ruleMap, ok := rule.(map[string]interface{})
			if !ok {
				continue
			}
			backendRefs, found, _ := unstructured.NestedSlice(ruleMap, "backendRefs")
			if !found {
				continue
			}
			for _, ref := range backendRefs {
				refMap, ok := ref.(map[string]interface{})
				if !ok {
					continue
				}
				kind, _, _ := unstructured.NestedString(refMap, "kind")
				if kind == "" || kind == "Service" {
					name, found, _ := unstructured.NestedString(refMap, "name")
					if found {
						backendServices[name] = true
					}
				}
			}
		}
		if len(backendServices) > 0 {
			serviceList := make([]string, 0, len(backendServices))
			for svc := range backendServices {
				serviceList = append(serviceList, svc)
			}
			status.AddField("Backends:  ", strings.Join(serviceList, ", "))
		}
	}

	// Status conditions
	conditions, found, _ := unstructured.NestedSlice(grpcRoute, "status", "parents")
	if found && len(conditions) > 0 {
		condStrs := make([]string, 0)
		for _, parent := range conditions {
			parentMap, ok := parent.(map[string]interface{})
			if !ok {
				continue
			}
			parentName, _, _ := unstructured.NestedString(parentMap, "parentRef", "name")
			conds, found, _ := unstructured.NestedSlice(parentMap, "conditions")
			if !found {
				continue
			}
			for _, cond := range conds {
				condMap, ok := cond.(map[string]interface{})
				if !ok {
					continue
				}
				condType, _, _ := unstructured.NestedString(condMap, "type")
				condStatus, _, _ := unstructured.NestedString(condMap, "status")
				reason, _, _ := unstructured.NestedString(condMap, "reason")

				condStr := fmt.Sprintf("%s: %s=%s", parentName, condType, condStatus)
				if reason != "" {
					condStr += fmt.Sprintf(" (%s)", reason)
				}
				condStrs = append(condStrs, condStr)
			}
		}
		if len(condStrs) > 0 {
			status.AddField("Status:    ", strings.Join(condStrs, "; "))
		}
	}

	return status, nil
}
