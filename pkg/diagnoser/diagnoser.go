package diagnoser

import (
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// NewDiagnoser creates Diagnoser resource.
func NewDiagnoser(target types.NamespacedName, clientset *kubernetes.Clientset) *Diagnoser {
	d := &Diagnoser{
		Target:    target,
		Clientset: clientset,
	}
	return d
}

// Diagnoser diagnoses a target resource.
type Diagnoser struct {
	Target types.NamespacedName

	*kubernetes.Clientset
}
