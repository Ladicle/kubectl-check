package diagnoser

import (
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/Ladicle/kubectl-diagnose/pkg/pritty"
)

// NewOptions creates Diagnoser resource.
func NewOptions(target types.NamespacedName, clientset *kubernetes.Clientset) *Options {
	d := &Options{
		Target:    target,
		Clientset: clientset,
	}
	return d
}

// Options diagnoses a target resource.
type Options struct {
	Target types.NamespacedName

	*kubernetes.Clientset
}

type Diagnoser interface {
	Diagnose(printer *pritty.Printer) error
}
