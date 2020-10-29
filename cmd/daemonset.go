package cmd

import (
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/Ladicle/kubectl-check/pkg/checker"
	"github.com/Ladicle/kubectl-check/pkg/pritty"
	dcmdutil "github.com/Ladicle/kubectl-check/pkg/util/cmd"
)

func NewDaemonSetCmd(f cmdutil.Factory, printer *pritty.Printer) *cobra.Command {
	opts := CmdOptions{
		Resource: "DaemonSet",
		createCheckerFn: func(opts *checker.Options) checker.Checker {
			return checker.NewDaemonSetChecker(opts)
		},
	}
	cmd := &cobra.Command{
		Use:                   "daemonset [flags...] <name>",
		Aliases:               []string{"ds"},
		DisableFlagsInUseLine: true,
		Short:                 "Check DaemonSet resource",
		Run: func(cmd *cobra.Command, args []string) {
			dcmdutil.CheckErr(opts.Validate(args))
			dcmdutil.CheckErr(opts.Complete(f))
			dcmdutil.CheckErr(opts.Run(printer))
		},
	}
	return cmd
}
