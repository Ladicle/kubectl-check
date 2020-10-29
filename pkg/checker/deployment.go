package checker

import (
	"context"
	"errors"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/Ladicle/kubectl-check/pkg/pod"
	"github.com/Ladicle/kubectl-check/pkg/pritty"
	condutil "github.com/Ladicle/kubectl-check/pkg/util/cond"
)

// NewDeploymentChecker creates Deployment Checkr resource.
func NewDeploymentChecker(opts *Options) Checker {
	return &DeploymentChecker{Options: opts}
}

// DeploymentChecker checks a target deployment resource.
type DeploymentChecker struct {
	*Options
}

func (dc DeploymentChecker) Check(printer *pritty.Printer) error {
	deploy, err := dc.getTarget()
	if err != nil {
		return err
	}

	available, err := dc.checkDeploymentAvailable(deploy)
	if err != nil {
		return err
	}
	if available {
		fmt.Fprintf(printer.IOStreams.Out, "%v is available\n", dc.Target)
		return nil
	}

	fmt.Fprintf(printer.IOStreams.Out, "Deployment %q is not available (%d/%d):\n\n",
		dc.Target, deploy.Status.AvailableReplicas, deploy.Status.Replicas)
	pods, err := dc.getLatestPods(deploy)
	if err != nil {
		return err
	}
	return pod.ReportPodsDetail(dc.Clientset, printer, pods.Items)
}

func (dc *DeploymentChecker) getTarget() (*appsv1.Deployment, error) {
	return dc.Clientset.AppsV1().Deployments(dc.Target.Namespace).Get(
		context.Background(), dc.Target.Name, metav1.GetOptions{})
}

func (dc *DeploymentChecker) checkDeploymentAvailable(deploy *appsv1.Deployment) (bool, error) {
	for _, cond := range deploy.Status.Conditions {
		if cond.Type == appsv1.DeploymentAvailable {
			return condutil.IsStatusTrue(cond.Status), nil
		}
	}
	return false, nil
}

func (dc DeploymentChecker) getLatestPods(deploy *appsv1.Deployment) (*corev1.PodList, error) {
	rss, err := dc.Clientset.AppsV1().ReplicaSets(deploy.Namespace).List(context.Background(), metav1.ListOptions{
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
	return dc.Clientset.CoreV1().Pods(deploy.Namespace).List(context.Background(), opt)
}
