package pritty

import "k8s.io/cli-runtime/pkg/genericclioptions"

type Printer struct {
	IOStreams genericclioptions.IOStreams
	Color     bool
	TTY       bool
}
