package checker

import (
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/Ladicle/kubectl-check/pkg/pritty"
)

// NewOptions creates Checkr resource.
func NewOptions(target types.NamespacedName, clientset *kubernetes.Clientset) *Options {
	d := &Options{
		Target:    target,
		Clientset: clientset,
	}
	return d
}

// Options checks a target resource.
type Options struct {
	Target types.NamespacedName

	*kubernetes.Clientset
}

type Checker interface {
	Check(printer *pritty.Printer) error
}
