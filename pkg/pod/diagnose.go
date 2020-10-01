package pod

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/scheme"

	"github.com/Ladicle/kubectl-diagnose/pkg/pritty"
	condutil "github.com/Ladicle/kubectl-diagnose/pkg/util/cond"
	eventutil "github.com/Ladicle/kubectl-diagnose/pkg/util/event"
	"github.com/Ladicle/kubectl-diagnose/pkg/util/formatter"
)

func ReportPodsDetail(c *kubernetes.Clientset, printer *pritty.Printer, pods []corev1.Pod) error {
	for i := range pods {
		if err := reportPodDetail(c, printer, &pods[i]); err != nil {
			return err
		}
	}
	return nil
}

func reportPodDetail(c *kubernetes.Clientset, printer *pritty.Printer, pod *corev1.Pod) error {
	readypp := make(map[string]bool, len(pod.Spec.ReadinessGates))
	for _, rg := range pod.Spec.ReadinessGates {
		readypp[string(rg.ConditionType)] = true
	}

	var (
		errMsgList     []string
		notReadyCSList []corev1.ContainerStatus
	)
	for _, cond := range pod.Status.Conditions {
		var errMsg string
		switch cond.Type {
		case corev1.PodReady:
			continue
		case corev1.PodScheduled:
			// noop
		case corev1.PodInitialized:
			notReadyCSList = filterNotReadyContainers(pod.Status.InitContainerStatuses)
			errMsg = formatter.FormatContainerStatuses(pod.Name, notReadyCSList)
		case corev1.ContainersReady:
			notReadyCSList = filterNotReadyContainers(pod.Status.ContainerStatuses)
			errMsg = formatter.FormatContainerStatuses(pod.Name, notReadyCSList)
		}
		if condutil.IsStatusTrue(cond.Status) {
			continue
		}
		if errMsg == "" {
			errMsg = fmt.Sprintf("[%v] Pod/%v: %v", cond.Type, pod.Name, cond.Message)
		}
		errMsgList = append(errMsgList, errMsg)
	}

	events, err := c.CoreV1().Events(pod.Namespace).Search(scheme.Scheme, pod)
	if err != nil {
		return err
	}
	warnEventList := eventutil.FilterWarnEvents(events)

	if len(errMsgList) != 0 {
		fmt.Fprintf(printer.IOStreams.Out,
			"%v\n", strings.Join(errMsgList, "\n"))

		for _, cs := range notReadyCSList {
			if !isContainerStarted(cs) {
				continue
			}
			log, err := getContainerLog(c, pod.Namespace, pod.Name, cs.Name)
			if err != nil {
				return err
			}
			fmt.Fprintf(printer.IOStreams.Out,
				"\nContainer{%q} Log:\n%v\n", cs.Name, log)
		}
	}
	if len(warnEventList) != 0 {
		fmt.Fprintf(printer.IOStreams.Out,
			"\n%v\n", formatter.FormatEvents(warnEventList))
	}
	return nil
}

// isContainerStarted is a function to checks if the current or the last container holds the ContainerID.
// In other words, it is determined whether the container has been started even once.
func isContainerStarted(cs corev1.ContainerStatus) bool {
	switch {
	case cs.Ready && cs.ContainerID != "":
		return true
	case cs.State.Terminated != nil && cs.State.Terminated.ContainerID != "":
		return true
	case cs.LastTerminationState.Terminated != nil && cs.LastTerminationState.Terminated.ContainerID != "":
		return true
	}
	return false
}
