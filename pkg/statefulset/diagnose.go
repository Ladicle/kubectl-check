package statefulset

import (
	"context"
	"errors"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/Ladicle/kubectl-diagnose/pkg/diagnoser"
	"github.com/Ladicle/kubectl-diagnose/pkg/pod"
	"github.com/Ladicle/kubectl-diagnose/pkg/pritty"
)

// NewStatefulSetDiagnoser creates Statefulset Diagnoser resource.
func NewStatefulSetDiagnoser(diagnoser *diagnoser.Diagnoser) *StatefulSetDiagnoser {
	return &StatefulSetDiagnoser{Diagnoser: diagnoser}
}

// StatefulSetDiagnoser diagnoses a target statefulset resource.
type StatefulSetDiagnoser struct {
	*diagnoser.Diagnoser
}

func (d *StatefulSetDiagnoser) Diagnose(printer *pritty.Printer) error {
	sts, err := getStatefulSet(d.Clientset, d.Target)
	if err != nil {
		return err
	}

	if sts.Status.ReadyReplicas == sts.Status.Replicas {
		fmt.Fprintf(printer.IOStreams.Out, "%v is ready\n", d.Target)
		return nil
	}

	fmt.Fprintf(printer.IOStreams.Out, "Deployment %q is not ready (%d/%d):\n\n",
		d.Target, sts.Status.ReadyReplicas, sts.Status.Replicas)
	pods, err := getChildPods(d.Clientset, sts)
	if err != nil {
		return err
	}
	return pod.ReportPodsDetail(d.Clientset, printer, pods.Items)
}

func getStatefulSet(c *kubernetes.Clientset, nn types.NamespacedName) (*appsv1.StatefulSet, error) {
	return c.AppsV1().StatefulSets(nn.Namespace).
		Get(context.Background(), nn.Name, metav1.GetOptions{})
}

func getChildPods(c *kubernetes.Clientset, sts *appsv1.StatefulSet) (*corev1.PodList, error) {
	if sts.Status.CurrentRevision == "" {
		return nil, errors.New(".state.currentRevision is empty")
	}
	opt := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%v=%v",
			appsv1.ControllerRevisionHashLabelKey,
			sts.Status.CurrentRevision),
	}
	return c.CoreV1().Pods(sts.Namespace).List(context.Background(), opt)
}
