package deployment

import (
	"context"
	"fmt"

	"github.com/Ladicle/kubectl-diagnose/pkg/pritty"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
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

	fmt.Fprintf(printer.IOStreams.Out, "%v is not available", d.Target)
	// TODO: dig Pods

	return nil
}

func getDeployment(c *kubernetes.Clientset, nn types.NamespacedName) (*appsv1.Deployment, error) {
	return c.AppsV1().Deployments(nn.Namespace).Get(
		context.Background(), nn.Name, metav1.GetOptions{})
}

func (d *Diagnoser) checkDeploymentAvailable(deploy *appsv1.Deployment) (bool, error) {
	for _, cond := range deploy.Status.Conditions {
		if cond.Type == appsv1.DeploymentAvailable {
			return true, nil
		}
	}
	return false, nil
}
