package cond

import corev1 "k8s.io/api/core/v1"

func IsStatusTrue(status corev1.ConditionStatus) bool {
	return status == corev1.ConditionTrue
}
