package cmd

import (
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/Ladicle/kubectl-diagnose/pkg/diagnoser"
	"github.com/Ladicle/kubectl-diagnose/pkg/pritty"
	dcmdutil "github.com/Ladicle/kubectl-diagnose/pkg/util/cmd"
)

func NewDeploymentCmd(f cmdutil.Factory, printer *pritty.Printer) *cobra.Command {
	opts := CmdOptions{
		Resource: "Deployment",
		createDiagnoserFn: func(opts *diagnoser.Options) diagnoser.Diagnoser {
			return diagnoser.NewDeploymentDiagnoser(opts)
		},
	}
	cmd := &cobra.Command{
		Use:                   "deployment <name>",
		Aliases:               []string{"deploy", "dp"},
		DisableFlagsInUseLine: true,
		Short:                 "Diagnose Deployment resource",
		Run: func(cmd *cobra.Command, args []string) {
			dcmdutil.CheckErr(opts.Validate(args))
			dcmdutil.CheckErr(opts.Complete(f))
			dcmdutil.CheckErr(opts.Run(printer))
		},
	}
	return cmd
}
