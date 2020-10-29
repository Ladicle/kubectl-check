package checker

import (
	"context"
	"errors"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Ladicle/kubectl-check/pkg/pod"
	"github.com/Ladicle/kubectl-check/pkg/pritty"
)

// NewDaemonSetChecker creates Statefulset Checkr resource.
func NewDaemonSetChecker(opts *Options) Checker {
	return &DaemonSetChecker{Options: opts}
}

// DaemonSetChecker checks a target statefulset resource.
type DaemonSetChecker struct {
	*Options
}

func (dsc DaemonSetChecker) Check(printer *pritty.Printer) error {
	ds, err := dsc.getTarget()
	if err != nil {
		return err
	}

	if ds.Status.NumberReady == ds.Status.DesiredNumberScheduled {
		fmt.Fprintf(printer.IOStreams.Out, "%v is ready\n", dsc.Target)
		return nil
	}

	fmt.Fprintf(printer.IOStreams.Out, "DaemonSet %q is not ready (%d/%d):\n\n",
		dsc.Target, ds.Status.NumberReady, ds.Status.DesiredNumberScheduled)
	pods, err := dsc.getLatestPods(ds)
	if err != nil {
		return err
	}
	return pod.ReportPodsDetail(dsc.Clientset, printer, pods.Items)
}

func (dsc DaemonSetChecker) getTarget() (*appsv1.DaemonSet, error) {
	return dsc.Clientset.AppsV1().DaemonSets(dsc.Target.Namespace).
		Get(context.Background(), dsc.Target.Name, metav1.GetOptions{})
}

func (dsc DaemonSetChecker) getLatestPods(ds *appsv1.DaemonSet) (*corev1.PodList, error) {
	if ds.Status.ObservedGeneration == 0 {
		return nil, errors.New(".state.observedGeneration is empty")
	}

	var labelSelector []string
	for k, v := range ds.Spec.Selector.MatchLabels {
		labelSelector = append(labelSelector, fmt.Sprintf("%v=%v", k, v))
	}
	opt := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%v=%v,%v",
			"pod-template-generation",
			ds.Status.ObservedGeneration,
			strings.Join(labelSelector, ",")),
	}
	return dsc.Clientset.CoreV1().Pods(ds.Namespace).List(context.Background(), opt)
}
