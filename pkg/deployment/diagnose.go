package deployment

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/Ladicle/kubectl-diagnose/pkg/pritty"
	"github.com/Ladicle/kubectl-diagnose/pkg/util/formatter"
)

// NewDiagnoser creates Deployment Diagnoser resource.
func NewDiagnoser(target types.NamespacedName, clientset *kubernetes.Clientset) *Diagnoser {
	d := &Diagnoser{
		Target:    target,
		Clientset: clientset,
	}
	return d
}

// Diagnoser diagnoses a target deployment resource.
type Diagnoser struct {
	Target types.NamespacedName

	*kubernetes.Clientset
}

func (d *Diagnoser) Diagnose(printer *pritty.Printer) error {
	deploy, err := getDeployment(d.Clientset, d.Target)
	if err != nil {
		return err
	}

	available, err := d.checkDeploymentAvailable(deploy)
	if err != nil {
		return err
	}
	if available {
		fmt.Fprintf(printer.IOStreams.Out, "%v is available\n", d.Target)
		return nil
	}

	fmt.Fprintf(printer.IOStreams.Out, "Deployment %q is not available (%d/%d):\n\n",
		d.Target, deploy.Status.AvailableReplicas, deploy.Status.Replicas)
	pods, err := getLatestPods(d.Clientset, deploy)
	if err != nil {
		return err
	}
	for i := range pods.Items {
		pod := &pods.Items[i]
		if err := d.reportPodDetail(printer, pod); err != nil {
			return err
		}
	}
	return nil
}

func isStatusTrue(status corev1.ConditionStatus) bool {
	return status == corev1.ConditionTrue
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

func filterNotReadyContainers(css []corev1.ContainerStatus) []corev1.ContainerStatus {
	var notReadyContainers []corev1.ContainerStatus
	for _, cs := range css {
		if !cs.Ready {
			notReadyContainers = append(notReadyContainers, cs)
		}
	}
	return notReadyContainers
}

func filterWarnEvents(events *corev1.EventList) []corev1.Event {
	var warnEv []corev1.Event
	for _, ev := range events.Items {
		if ev.Type == corev1.EventTypeWarning {
			warnEv = append(warnEv, ev)
		}
	}
	return warnEv
}

func getContainerLog(c *kubernetes.Clientset, ns, pname, cname string) (string, error) {
	var tailN = int64(15)
	req := c.CoreV1().Pods(ns).GetLogs(pname, &corev1.PodLogOptions{
		TailLines: &tailN,
		Container: cname,
	})

	readCloser, err := req.Stream(context.TODO())
	if err != nil {
		return "", err
	}
	defer readCloser.Close()

	var (
		line int
		buf  bytes.Buffer
		r    = bufio.NewScanner(readCloser)
	)
	for r.Scan() {
		buf.Write(r.Bytes())
		buf.WriteString("\n")
		line++
	}
	if line == 0 {
		buf.WriteString("<none>\n")
	}
	return buf.String(), nil
}

func getDeployment(c *kubernetes.Clientset, nn types.NamespacedName) (*appsv1.Deployment, error) {
	return c.AppsV1().Deployments(nn.Namespace).Get(
		context.Background(), nn.Name, metav1.GetOptions{})
}

func getLatestPods(c *kubernetes.Clientset, deploy *appsv1.Deployment) (*corev1.PodList, error) {
	rss, err := c.AppsV1().ReplicaSets(deploy.Namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labels.Set(deploy.Spec.Selector.MatchLabels).String(),
	})
	if err != nil {
		return nil, err
	}
	if len(rss.Items) == 0 {
		return nil, errors.New("not found ReplicaSet")
	}
	latestRS := &rss.Items[0]
	for i := range rss.Items[1:] {
		rs := &rss.Items[i]
		if latestRS.Status.ObservedGeneration < rs.Status.ObservedGeneration {
			latestRS = rs
		}
	}

	tplhash, ok := latestRS.ObjectMeta.Labels[appsv1.DefaultDeploymentUniqueLabelKey]
	if !ok {
		return nil, errors.New("ReplicaSet does not have pod-template-hash")
	}

	opt := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%v=%v",
			appsv1.DefaultDeploymentUniqueLabelKey,
			tplhash),
	}
	return c.CoreV1().Pods(deploy.Namespace).List(context.Background(), opt)
}

func (d *Diagnoser) checkDeploymentAvailable(deploy *appsv1.Deployment) (bool, error) {
	for _, cond := range deploy.Status.Conditions {
		if cond.Type == appsv1.DeploymentAvailable {
			return isStatusTrue(cond.Status), nil
		}
	}
	return false, nil
}

func (d *Diagnoser) reportPodDetail(printer *pritty.Printer, pod *corev1.Pod) error {
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
		if isStatusTrue(cond.Status) {
			continue
		}
		if errMsg == "" {
			errMsg = fmt.Sprintf("[%v] Pod/%v: %v", cond.Type, pod.Name, cond.Message)
		}
		errMsgList = append(errMsgList, errMsg)
	}

	events, err := d.CoreV1().Events(pod.Namespace).Search(scheme.Scheme, pod)
	if err != nil {
		return err
	}
	warnEventList := filterWarnEvents(events)

	if len(errMsgList) != 0 {
		fmt.Fprintf(printer.IOStreams.Out,
			"%v\n", strings.Join(errMsgList, "\n"))

		for _, cs := range notReadyCSList {
			if !isContainerStarted(cs) {
				continue
			}
			log, err := getContainerLog(d.Clientset, pod.Namespace, pod.Name, cs.Name)
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
