package event

import corev1 "k8s.io/api/core/v1"

func FilterWarnEvents(events *corev1.EventList) []corev1.Event {
	var warnEv []corev1.Event
	for _, ev := range events.Items {
		if ev.Type == corev1.EventTypeWarning {
			warnEv = append(warnEv, ev)
		}
	}
	return warnEv
}
