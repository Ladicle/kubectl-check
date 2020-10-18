package cmd

import (
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/Ladicle/kubectl-diagnose/pkg/diagnoser"
	"github.com/Ladicle/kubectl-diagnose/pkg/pritty"
	dcmdutil "github.com/Ladicle/kubectl-diagnose/pkg/util/cmd"
)

func NewStatefulSetCmd(f cmdutil.Factory, printer *pritty.Printer) *cobra.Command {
	opts := CmdOptions{
		Resource: "StatefulSet",
		createDiagnoserFn: func(opts *diagnoser.Options) diagnoser.Diagnoser {
			return diagnoser.NewStatefulSetDiagnoser(opts)
		},
	}
	cmd := &cobra.Command{
		Use:                   "statefulset <name>",
		Aliases:               []string{"sts"},
		DisableFlagsInUseLine: true,
		Short:                 "Diagnose StatefulSet resource",
		Run: func(cmd *cobra.Command, args []string) {
			dcmdutil.CheckErr(opts.Validate(args))
			dcmdutil.CheckErr(opts.Complete(f))
			dcmdutil.CheckErr(opts.Run(printer))
		},
	}
	return cmd
}
