package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"

	// Initialize all known client auth plugins.
	"k8s.io/apimachinery/pkg/types"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/term"

	"github.com/Ladicle/kubectl-check/pkg/checker"
	"github.com/Ladicle/kubectl-check/pkg/pritty"
)

var (
	// set values via build flags
	version string
	commit  string
)

func NewCheckCmd() *cobra.Command {
	ioStreams := genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}

	var optionsFlag bool
	cmds := &cobra.Command{
		Use:                   "check [flags...] <resource> <name>",
		Version:               fmt.Sprintf("%v @%v", version, commit),
		DisableFlagsInUseLine: true,
		Short:                 "Check Kubernetes resource status",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if optionsFlag {
				fmt.Fprint(ioStreams.Out, "The following options can be passed to any command:\n\n"+cmd.Flags().FlagUsages())
				os.Exit(0)
			}
		},
		Run: cmdutil.DefaultSubCommandRun(os.Stderr),
	}

	flags := cmds.PersistentFlags()
	flags.AddGoFlagSet(flag.CommandLine)

	kubeConfigFlags := genericclioptions.NewConfigFlags(true)
	kubeConfigFlags.AddFlags(flags)
	matchVersionFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	matchVersionFlags.AddFlags(flags)

	f := cmdutil.NewFactory(matchVersionFlags)

	printer := &pritty.Printer{IOStreams: ioStreams}
	cmds.PersistentFlags().BoolVarP(&printer.Color, "color", "R", false, "Enable color output even if stdout is not a terminal")
	printer.TTY = term.TTY{Out: ioStreams.Out}.IsTerminalOut()

	cmds.AddCommand(NewDeploymentCmd(f, printer))
	cmds.AddCommand(NewStatefulSetCmd(f, printer))
	cmds.AddCommand(NewDaemonSetCmd(f, printer))

	cmds.PersistentFlags().BoolVarP(&optionsFlag, "options", "", false, "Show full options of this command")
	cmds.SetUsageTemplate(usageTemplate)

	return cmds
}

type CmdOptions struct {
	Resource string
	Name     string

	checker         checker.Checker
	createCheckerFn func(opts *checker.Options) checker.Checker
}

func (o *CmdOptions) Validate(args []string) error {
	if len(args) != 1 {
		return errors.New(
			fmt.Sprintf("invalid number of arguments: %v <name> is a required argument", o.Resource))
	}
	o.Name = args[0]
	return nil
}

func (o *CmdOptions) Complete(f cmdutil.Factory) error {
	c, err := f.KubernetesClientSet()
	if err != nil {
		return err
	}

	k8sCfg := f.ToRawKubeConfigLoader()
	ns, _, err := k8sCfg.Namespace()
	if err != nil {
		return err
	}

	target := types.NamespacedName{Name: o.Name, Namespace: ns}
	opts := checker.NewOptions(target, c)
	o.checker = o.createCheckerFn(opts)
	return nil
}

func (o *CmdOptions) Run(printer *pritty.Printer) error {
	return o.checker.Check(printer)
}
