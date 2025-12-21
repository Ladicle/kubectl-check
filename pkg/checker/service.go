package checker

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

func checkService(svc corev1.Service, c kubernetes.Interface, _ dynamic.Interface) (*ResourceStatus, error) {
	status := &ResourceStatus{
		Kind:      "Service",
		Name:      svc.Name,
		Namespace: svc.Namespace,
	}

	// Type
	if svc.Spec.Type != "" {
		status.AddField("Type:      ", string(svc.Spec.Type))
	}

	// Cluster IPs (dual-stack aware)
	var clusterIPs []string
	if len(svc.Spec.ClusterIPs) > 0 {
		clusterIPs = append(clusterIPs, svc.Spec.ClusterIPs...)
	} else if svc.Spec.ClusterIP != "" {
		clusterIPs = append(clusterIPs, svc.Spec.ClusterIP)
	}
	if len(clusterIPs) > 0 {
		status.AddField("IP:        ", strings.Join(clusterIPs, ", "))
	}

	// External IPs
	if len(svc.Spec.ExternalIPs) > 0 {
		status.AddField("ExternalIP:", strings.Join(svc.Spec.ExternalIPs, ", "))
	}

	// Ports
	if len(svc.Spec.Ports) > 0 {
		ports := make([]string, 0, len(svc.Spec.Ports))
		for _, p := range svc.Spec.Ports {
			target := p.TargetPort.String()
			node := ""
			if p.NodePort > 0 {
				node = fmt.Sprintf(" (node:%d)", p.NodePort)
			}
			name := ""
			if p.Name != "" {
				name = fmt.Sprintf(" [%s]", p.Name)
			}
			ports = append(ports, fmt.Sprintf("%d/%s->%s%s%s", p.Port, p.Protocol, target, node, name))
		}
		status.AddField("Ports:     ", strings.Join(ports, ", "))
	}

	// Selector
	if len(svc.Spec.Selector) > 0 {
		status.AddField("Selector:  ", labels.Set(svc.Spec.Selector).String())
	}

	// Age
	if !svc.CreationTimestamp.IsZero() {
		status.AddField("Age:       ", formatAge(svc.CreationTimestamp.Time))
	}

	return status, nil
}
