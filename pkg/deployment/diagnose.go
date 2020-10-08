package deployment

import (
	"context"
	"errors"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/Ladicle/kubectl-diagnose/pkg/diagnoser"
	"github.com/Ladicle/kubectl-diagnose/pkg/pod"
	"github.com/Ladicle/kubectl-diagnose/pkg/pritty"
	condutil "github.com/Ladicle/kubectl-diagnose/pkg/util/cond"
)

// NewDeploymentDiagnoser creates Deployment Diagnoser resource.
func NewDeploymentDiagnoser(diagnoser *diagnoser.Diagnoser) *DeploymentDiagnoser {
	return &DeploymentDiagnoser{Diagnoser: diagnoser}
}

// DeploymentDiagnoser diagnoses a target deployment resource.
type DeploymentDiagnoser struct {
	*diagnoser.Diagnoser
}

func (d *DeploymentDiagnoser) Diagnose(printer *pritty.Printer) error {
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
	return pod.ReportPodsDetail(d.Clientset, printer, pods.Items)
}

func getDeployment(c *kubernetes.Clientset, nn types.NamespacedName) (*appsv1.Deployment, error) {
	return c.AppsV1().Deployments(nn.Namespace).Get(
		context.Background(), nn.Name, metav1.GetOptions{})
}

func (d *DeploymentDiagnoser) checkDeploymentAvailable(deploy *appsv1.Deployment) (bool, error) {
	for _, cond := range deploy.Status.Conditions {
		if cond.Type == appsv1.DeploymentAvailable {
			return condutil.IsStatusTrue(cond.Status), nil
		}
	}
	return false, nil
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
