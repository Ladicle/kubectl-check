package checker

import (
	"context"
	"errors"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Ladicle/kubectl-check/pkg/pod"
	"github.com/Ladicle/kubectl-check/pkg/pritty"
)

// NewStatefulSetChecker creates Statefulset Checker resource.
func NewStatefulSetChecker(opts *Options) Checker {
	return &StatefulSetChecker{Options: opts}
}

// StatefulSetChecker checks a target statefulset resource.
type StatefulSetChecker struct {
	*Options
}

func (ssc *StatefulSetChecker) Check(printer *pritty.Printer) error {
	sts, err := ssc.getTarget()
	if err != nil {
		return err
	}

	if sts.Status.ReadyReplicas == sts.Status.Replicas {
		fmt.Fprintf(printer.IOStreams.Out, "%v is ready\n", ssc.Target)
		return nil
	}

	fmt.Fprintf(printer.IOStreams.Out, "Deployment %q is not ready (%d/%d):\n\n",
		ssc.Target, sts.Status.ReadyReplicas, sts.Status.Replicas)
	pods, err := ssc.getLatestPods(sts)
	if err != nil {
		return err
	}
	return pod.ReportPodsDetail(ssc.Clientset, printer, pods.Items)
}

func (ssc *StatefulSetChecker) getTarget() (*appsv1.StatefulSet, error) {
	return ssc.Clientset.AppsV1().StatefulSets(ssc.Target.Namespace).
		Get(context.Background(), ssc.Target.Name, metav1.GetOptions{})
}

func (ssc *StatefulSetChecker) getLatestPods(sts *appsv1.StatefulSet) (*corev1.PodList, error) {
	if sts.Status.UpdateRevision == "" {
		return nil, errors.New(".state.updateRevision is empty")
	}
	opt := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%v=%v",
			appsv1.ControllerRevisionHashLabelKey,
			sts.Status.CurrentRevision),
	}
	return ssc.Clientset.CoreV1().Pods(sts.Namespace).List(context.Background(), opt)
}
