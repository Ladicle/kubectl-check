package cmd

import (
	"errors"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/Ladicle/kubectl-diagnose/pkg/daemonset"
	"github.com/Ladicle/kubectl-diagnose/pkg/diagnoser"
	"github.com/Ladicle/kubectl-diagnose/pkg/pritty"
	dcmdutil "github.com/Ladicle/kubectl-diagnose/pkg/util/cmd"
)

type DaemonSetOptions struct {
	Name      string
	Namespace string

	clientset *kubernetes.Clientset
}

func NewDaemonSetCmd(f cmdutil.Factory, printer *pritty.Printer) *cobra.Command {
	opts := DaemonSetOptions{}
	cmd := &cobra.Command{
		Use:                   "daemonset <name>",
		Aliases:               []string{"ds"},
		DisableFlagsInUseLine: true,
		Short:                 "Diagnose DaemonSet resource",
		Run: func(cmd *cobra.Command, args []string) {
			dcmdutil.CheckErr(opts.Validate(args))
			dcmdutil.CheckErr(opts.Complete(f))
			dcmdutil.CheckErr(opts.Run(printer))
		},
	}
	return cmd
}

func (o *DaemonSetOptions) Validate(args []string) error {
	if len(args) != 1 {
		return errors.New("invalid number of arguments: DaemonSet <name> is a required argument")
	}
	o.Name = args[0]
	return nil
}

func (o *DaemonSetOptions) Complete(f cmdutil.Factory) error {
	c, err := f.KubernetesClientSet()
	if err != nil {
		return err
	}
	o.clientset = c

	k8sCfg := f.ToRawKubeConfigLoader()
	ns, _, err := k8sCfg.Namespace()
	if err != nil {
		return err
	}
	o.Namespace = ns
	return nil
}

func (o *DaemonSetOptions) Run(printer *pritty.Printer) error {
	target := types.NamespacedName{Name: o.Name, Namespace: o.Namespace}
	diagnoser := daemonset.NewDaemonSetDiagnoser(diagnoser.NewDiagnoser(target, o.clientset))
	return diagnoser.Diagnose(printer)
}
