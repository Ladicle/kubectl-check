package cmd

import (
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/Ladicle/kubectl-check/pkg/checker"
	"github.com/Ladicle/kubectl-check/pkg/pritty"
	dcmdutil "github.com/Ladicle/kubectl-check/pkg/util/cmd"
)

func NewStatefulSetCmd(f cmdutil.Factory, printer *pritty.Printer) *cobra.Command {
	opts := CmdOptions{
		Resource: "StatefulSet",
		createCheckerFn: func(opts *checker.Options) checker.Checker {
			return checker.NewStatefulSetChecker(opts)
		},
	}
	cmd := &cobra.Command{
		Use:                   "statefulset <name>",
		Aliases:               []string{"sts"},
		DisableFlagsInUseLine: true,
		Short:                 "Check StatefulSet resource",
		Run: func(cmd *cobra.Command, args []string) {
			dcmdutil.CheckErr(opts.Validate(args))
			dcmdutil.CheckErr(opts.Complete(f))
			dcmdutil.CheckErr(opts.Run(printer))
		},
	}
	return cmd
}
