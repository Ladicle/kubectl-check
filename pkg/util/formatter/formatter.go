package formatter

import (
	"bytes"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
)

func FormatContainerStatuses(podName string, css []corev1.ContainerStatus) string {
	var statuses []string
	for _, cs := range css {
		switch {
		case cs.Ready:
			statuses = append(statuses,
				fmt.Sprintf("[Running] Pod/%v/%v:", podName, cs.Name))
		case cs.State.Waiting != nil:
			statuses = append(statuses,
				fmt.Sprintf("[%v] Pod/%v/%v: %v (restarted x%v)",
					cs.State.Waiting.Reason, podName, cs.Name,
					cs.State.Waiting.Message, cs.RestartCount))
		case cs.State.Terminated != nil:
			statuses = append(statuses,
				fmt.Sprintf("[%v] Pod/%v/%v: %v (exit-code %v)",
					cs.State.Terminated.Reason, podName, cs.Name,
					cs.State.Terminated.Message, cs.State.Terminated.ExitCode))
		}
	}
	return strings.Join(statuses, "\n")
}

func FormatEvents(events []corev1.Event) string {
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
		return fmt.Sprintf("%s (x%d over %s)",
			translateTimestampSince(ev.LastTimestamp),
			ev.Count,
			translateTimestampSince(ev.FirstTimestamp))
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
		path := strings.TrimPrefix(ref.FieldPath, "spec.containers{")
		path = strings.TrimSuffix(path, "}")
		ivo = append(ivo, path)
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
