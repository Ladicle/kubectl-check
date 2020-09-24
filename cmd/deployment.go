package cmd

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"github.com/Ladicle/kubectl-diagnose/pkg/deployment"
	dcmdutil "github.com/Ladicle/kubectl-diagnose/pkg/util/cmd"
)

type DeploymentOptions struct {
}

func NewDeploymentCmd(f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	opts := DeploymentOptions{}
	cmd := &cobra.Command{
		Use:                   "deploy <resource>",
		DisableFlagsInUseLine: true,
		Short:                 "Diagnose Deployment resource",
		Run: func(cmd *cobra.Command, args []string) {
			dcmdutil.CheckErr(opts.Validate(cmd, args))
			dcmdutil.CheckErr(opts.Run())
		},
	}
	return cmd
}

func (o *DeploymentOptions) Validate(cmd *cobra.Command, args []string) error {
	return nil
}

func (o *DeploymentOptions) Run() error {
	return deployment.Diagnose()
}
