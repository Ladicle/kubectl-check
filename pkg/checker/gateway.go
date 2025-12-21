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

func listGatewayChild(gateway map[string]interface{}, dc dynamic.Interface) (*unstructured.Unstructured, error) {
	namespace, _, err := unstructured.NestedString(gateway, "metadata", "namespace")
	if err != nil {
		return nil, err
	}
	name, _, err := unstructured.NestedString(gateway, "metadata", "name")
	if err != nil {
		return nil, err
	}

	// List HTTPRoutes and GRPCRoutes that reference this Gateway
	httpRouteInfo, ok := Resources["HTTPRoute"]
	if !ok {
		return nil, fmt.Errorf("could not resolve HTTPRoute resource")
	}
	grpcRouteInfo, ok := Resources["GRPCRoute"]
	if !ok {
		return nil, fmt.Errorf("could not resolve GRPCRoute resource")
	}

	httpRouteList, err := dc.Resource(httpRouteInfo.GVR).Namespace(namespace).List(
		context.Background(),
		metav1.ListOptions{},
	)
	if err != nil {
		return nil, err
	}

	grpcRouteList, err := dc.Resource(grpcRouteInfo.GVR).Namespace(namespace).List(
		context.Background(),
		metav1.ListOptions{},
	)
	if err != nil {
		return nil, err
	}

	// Filter routes that reference this Gateway
	items := make([]interface{}, 0)
	for i := range httpRouteList.Items {
		route := &httpRouteList.Items[i]
		parentRefs, found, err := unstructured.NestedSlice(route.Object, "spec", "parentRefs")
		if err != nil || !found {
			continue
		}

		for _, ref := range parentRefs {
			refMap, ok := ref.(map[string]interface{})
			if !ok {
				continue
			}
			refName, _, _ := unstructured.NestedString(refMap, "name")
			refNamespace, found, _ := unstructured.NestedString(refMap, "namespace")
			if !found {
				refNamespace = namespace
			}
			if refName == name && refNamespace == namespace {
				items = append(items, route.Object)
				break
			}
		}
	}

	for i := range grpcRouteList.Items {
		route := &grpcRouteList.Items[i]
		parentRefs, found, err := unstructured.NestedSlice(route.Object, "spec", "parentRefs")
		if err != nil || !found {
			continue
		}

		for _, ref := range parentRefs {
			refMap, ok := ref.(map[string]interface{})
			if !ok {
				continue
			}
			refName, _, _ := unstructured.NestedString(refMap, "name")
			refNamespace, found, _ := unstructured.NestedString(refMap, "namespace")
			if !found {
				refNamespace = namespace
			}
			if refName == name && refNamespace == namespace {
				items = append(items, route.Object)
				break
			}
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

func checkGateway(gateway map[string]interface{}, c kubernetes.Interface, dc dynamic.Interface) (*ResourceStatus, error) {
	namespace, _, _ := unstructured.NestedString(gateway, "metadata", "namespace")
	name, _, _ := unstructured.NestedString(gateway, "metadata", "name")

	status := &ResourceStatus{
		Kind:      "Gateway",
		Name:      name,
		Namespace: namespace,
	}

	// GatewayClassName
	gatewayClassName, found, _ := unstructured.NestedString(gateway, "spec", "gatewayClassName")
	if found {
		status.AddField("Class:     ", gatewayClassName)
	}

	// Listeners
	listeners, found, _ := unstructured.NestedSlice(gateway, "spec", "listeners")
	if found && len(listeners) > 0 {
		listenerStrs := make([]string, 0, len(listeners))
		for _, l := range listeners {
			lMap, ok := l.(map[string]interface{})
			if !ok {
				continue
			}
			listenerName, _, _ := unstructured.NestedString(lMap, "name")
			protocol, _, _ := unstructured.NestedString(lMap, "protocol")
			port, _, _ := unstructured.NestedInt64(lMap, "port")
			hostname, _, _ := unstructured.NestedString(lMap, "hostname")

			listenerStr := fmt.Sprintf("%s:%s:%d", listenerName, protocol, port)
			if hostname != "" {
				listenerStr += fmt.Sprintf(" (%s)", hostname)
			}
			listenerStrs = append(listenerStrs, listenerStr)
		}
		status.AddField("Listeners: ", strings.Join(listenerStrs, ", "))
	}

	// Addresses
	addresses, found, _ := unstructured.NestedSlice(gateway, "status", "addresses")
	if found && len(addresses) > 0 {
		addrStrs := make([]string, 0, len(addresses))
		for _, a := range addresses {
			aMap, ok := a.(map[string]interface{})
			if !ok {
				continue
			}
			addrType, _, _ := unstructured.NestedString(aMap, "type")
			value, _, _ := unstructured.NestedString(aMap, "value")
			addrStrs = append(addrStrs, fmt.Sprintf("%s:%s", addrType, value))
		}
		status.AddField("Addresses: ", strings.Join(addrStrs, ", "))
	}

	// Status conditions
	conditions, found, _ := unstructured.NestedSlice(gateway, "status", "conditions")
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
