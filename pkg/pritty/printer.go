package pritty

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type Printer struct {
	IOStreams genericclioptions.IOStreams
	Color     bool
	TTY       bool
}

func (p Printer) SprintHeader(text string) string {
	return p.Sprint(Style(text).Fg(Cyan))
}

func (p Printer) Sprint(ts *TextStyle) string {
	if p.TTY || p.Color {
		return ts.String()
	}
	return ts.Text
}
