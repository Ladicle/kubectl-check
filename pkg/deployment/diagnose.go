package deployment

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/Ladicle/kubectl-diagnose/pkg/pritty"
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
		fmt.Fprintf(printer.IOStreams.Out, "%v is available", d.Target)
		return nil
	}

	pods, err := getLatestPods(d.Clientset, deploy)
	if err != nil {
		return err
	}
	for i := range pods.Items {
		pod := &pods.Items[i]
		if _, err := d.checkPodAvailable(printer, pod); err != nil {
			return err
		}
	}
	return nil
}

func isStatusTrue(status corev1.ConditionStatus) bool {
	return status == corev1.ConditionTrue
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

func formatContainerStatuses(css []corev1.ContainerStatus) string {
	const prefix = "  - "
	var statuses []string
	for _, cs := range css {
		switch {
		case cs.Ready:
			statuses = append(statuses,
				fmt.Sprintf("%v[%v is Running]:", prefix, cs.Name))
		case cs.State.Waiting != nil:
			statuses = append(statuses,
				fmt.Sprintf("%v[%v is %v]: %v (restarted x%v)", prefix,
					cs.Name, cs.State.Waiting.Reason, cs.State.Waiting.Message, cs.RestartCount))
		case cs.State.Terminated != nil:
			statuses = append(statuses,
				fmt.Sprintf("%v[%v is %v]: %v (exit-code %v)", prefix,
					cs.Name, cs.State.Terminated.Reason, cs.State.Terminated.Message, cs.State.Terminated.ExitCode))
		}
	}
	return strings.Join(statuses, "\n")
}

func formatEvents(events []corev1.Event) string {
	var buf bytes.Buffer
	tw := tabwriter.NewWriter(&buf, 0, 8, 2, ' ', 0)
	tw.Write([]byte("Reason\tAge\tFrom\tObject\tMessage\n"))
	tw.Write([]byte("------\t----\t----\t------\t-------\n"))
	for _, ev := range events {
		tw.Write([]byte(fmt.Sprintf(
			"%v\t%s\t%v\t%v\t%v\n",
			ev.Reason,
			FormatAge(ev),
			FormatEventSource(ev.Source),
			FormatInvolvedObject(ev.InvolvedObject),
			strings.TrimSpace(ev.Message),
		)))
	}
	tw.Flush()
	return buf.String()
}

func FormatAge(ev corev1.Event) string {
	if ev.Count > 1 {
		return fmt.Sprintf("%s (x%d over %s)", translateTimestampSince(ev.LastTimestamp), ev.Count, translateTimestampSince(ev.FirstTimestamp))
	}
	return translateTimestampSince(ev.FirstTimestamp)
}

// translateTimestampSince returns the elapsed time since timestamp in
// human-readable approximation.
func translateTimestampSince(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}
	return duration.HumanDuration(time.Since(timestamp.Time))
}

// FormatInvolvedObject formats ref.
func FormatInvolvedObject(ref corev1.ObjectReference) string {
	ivo := []string{ref.Kind, ref.Name}
	if ref.FieldPath != "" {
		ivo = append(ivo, ref.FieldPath)
	}
	return strings.Join(ivo, "/")
}

// FormatEventSource formats EventSource as a comma separated string excluding Host when empty
func FormatEventSource(es corev1.EventSource) string {
	EventSourceString := []string{es.Component}
	if len(es.Host) > 0 {
		EventSourceString = append(EventSourceString, es.Host)
	}
	return strings.Join(EventSourceString, ", ")
}

func getContainerLog(c *kubernetes.Clientset, ns, pname, cname string) (string, error) {
	// TODO: get container logs
	return "not implemented yet", nil
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

func (d *Diagnoser) checkPodAvailable(printer *pritty.Printer, pod *corev1.Pod) (bool, error) {
	readypp := make(map[string]bool, len(pod.Spec.ReadinessGates))
	for _, rg := range pod.Spec.ReadinessGates {
		readypp[string(rg.ConditionType)] = true
	}

	var (
		errMsgList     []string
		notImplList    []string
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
			errMsg = formatContainerStatuses(notReadyCSList)
		case corev1.ContainersReady:
			notReadyCSList = filterNotReadyContainers(pod.Status.ContainerStatuses)
			errMsg = formatContainerStatuses(notReadyCSList)
		default:
			if _, ok := readypp[string(cond.Type)]; !ok {
				notImplList = append(notImplList, string(cond.Type))
				continue
			}
		}
		if isStatusTrue(cond.Status) {
			continue
		}
		if errMsg == "" {
			errMsg = fmt.Sprintf("[%v is %v]: %v", cond.Type, cond.Status, cond.Message)
		}
		errMsgList = append(errMsgList, errMsg)
	}

	events, err := d.CoreV1().Events(pod.Namespace).Search(scheme.Scheme, pod)
	if err != nil {
		return false, err
	}
	warnEventList := filterWarnEvents(events)

	if len(notImplList) != 0 {
		fmt.Fprintf(printer.IOStreams.Out,
			"Not Yet Implemented Conditions:\n%v\n", strings.Join(notImplList, "\n"))
	}
	if len(errMsgList) != 0 {
		fmt.Fprintf(printer.IOStreams.Out,
			"Error Conditions:\n%v\n", strings.Join(errMsgList, "\n"))

		for _, cs := range notReadyCSList {
			log, err := getContainerLog(d.Clientset, pod.Namespace, pod.Name, cs.Name)
			if err != nil {
				return false, err
			}
			fmt.Fprintf(printer.IOStreams.Out,
				"Container %q Log:\n%v\n", cs.Name, log)
		}
	}
	if len(warnEventList) != 0 {
		fmt.Fprintf(printer.IOStreams.Out,
			"Warning Events:\n%v\n", formatEvents(warnEventList))
	}
	return len(notImplList) == 0 &&
		len(errMsgList) == 0 &&
		len(warnEventList) == 0, nil
}
