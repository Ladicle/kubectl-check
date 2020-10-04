package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/term"

	"github.com/Ladicle/kubectl-diagnose/pkg/pritty"
)

var (
	// set values via build flags
	version string
	commit  string
)

func NewDiagnoseCmd() *cobra.Command {
	cmds := &cobra.Command{
		Use:                   "diagnose",
		Version:               fmt.Sprintf("%v @%v", version, commit),
		DisableFlagsInUseLine: true,
		Short:                 "Diagnose Kubernetes resource status",
		Run:                   cmdutil.DefaultSubCommandRun(os.Stderr),
	}

	flags := cmds.PersistentFlags()
	flags.AddGoFlagSet(flag.CommandLine)

	kubeConfigFlags := genericclioptions.NewConfigFlags(true)
	kubeConfigFlags.AddFlags(flags)
	matchVersionFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	matchVersionFlags.AddFlags(flags)

	f := cmdutil.NewFactory(matchVersionFlags)
	ioStreams := genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}

	printer := &pritty.Printer{IOStreams: ioStreams}
	cmds.PersistentFlags().BoolVarP(&printer.Color, "color", "R", false, "Enable color output even if stdout is not a terminal")
	printer.TTY = term.TTY{Out: ioStreams.Out}.IsTerminalOut()

	cmds.AddCommand(NewDeploymentCmd(f, printer))
	cmds.AddCommand(NewStatefulSetCmd(f, printer))
	cmds.AddCommand(NewDaemonSetCmd(f, printer))

	return cmds
}
