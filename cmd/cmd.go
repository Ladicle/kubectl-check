package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	// Initialize all known client auth plugins.
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
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

	kubeConfigFlags := genericclioptions.NewConfigFlags(true)
	matchVersionFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	f := cmdutil.NewFactory(matchVersionFlags)

	printer := &pritty.Printer{IOStreams: ioStreams}
	printer.TTY = term.TTY{Out: ioStreams.Out}.IsTerminalOut()

	// Build alias to kind mapping
	aliasToKind := make(map[string]string)
	for kind, info := range checker.Resources {
		aliasToKind[strings.ToLower(kind)] = kind
		for _, alias := range info.Aliases {
			aliasToKind[alias] = kind
		}
	}

	cmds := &cobra.Command{
		Use:                   "check [flags...] <resource> <name>",
		Version:               fmt.Sprintf("%v @%v", version, commit),
		DisableFlagsInUseLine: true,
		Short:                 "Check Kubernetes resource status",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if optionsFlag {
				_, _ = fmt.Fprint(ioStreams.Out, "The following options can be passed to any command:\n\n"+cmd.Flags().FlagUsages())
				os.Exit(0)
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("<resource> and <name> are required arguments")
			}
			resourceArg := args[0]
			name := args[1]

			// Find kind from alias
			kind, ok := aliasToKind[resourceArg]
			if !ok {
				return fmt.Errorf("unsupported resource type: %s", resourceArg)
			}

			c, err := f.KubernetesClientSet()
			if err != nil {
				return fmt.Errorf("failed to create Kubernetes client: %w", err)
			}
			cfg, err := f.ToRESTConfig()
			if err != nil {
				return fmt.Errorf("failed to get REST config: %w", err)
			}
			dc, err := dynamic.NewForConfig(cfg)
			if err != nil {
				return fmt.Errorf("failed to create dynamic client: %w", err)
			}
			k8sCfg := f.ToRawKubeConfigLoader()
			ns, _, err := k8sCfg.Namespace()
			if err != nil {
				return fmt.Errorf("failed to get namespace: %w", err)
			}

			target := types.NamespacedName{Name: name, Namespace: ns}
			chk := checker.NewChecker(&checker.Config{
				Target:        target,
				Clientset:     c,
				DynamicClient: dc,
			}, kind)
			if err := chk.Check(printer); err != nil {
				return err
			}
			return nil
		},
	}

	flags := cmds.PersistentFlags()
	flags.AddGoFlagSet(flag.CommandLine)
	kubeConfigFlags.AddFlags(flags)
	matchVersionFlags.AddFlags(flags)

	cmds.PersistentFlags().BoolVarP(&printer.Color, "color", "R", false, "Enable color output even if stdout is not a terminal")
	cmds.PersistentFlags().BoolVarP(&optionsFlag, "options", "", false, "Show full options of this command")
	cmds.SetUsageTemplate(getUsageTemplate(printer))

	// Register namespace flag completion
	_ = cmds.RegisterFlagCompletionFunc("namespace", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		c, err := f.KubernetesClientSet()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		namespaces, err := c.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var completions []string
		for _, ns := range namespaces.Items {
			if strings.HasPrefix(ns.Name, toComplete) {
				// Add status as description if available
				status := string(ns.Status.Phase)
				if status != "" {
					completions = append(completions, ns.Name+"\t"+status)
				} else {
					completions = append(completions, ns.Name)
				}
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	})

	// Add shell completion
	cmds.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			// Complete resource types (aliases) with descriptions
			var completions []string
			for alias, kind := range aliasToKind {
				if strings.HasPrefix(alias, toComplete) {
					// Add description from resource info if available
					if resInfo, ok := checker.Resources[kind]; ok {
						completions = append(completions, alias+"\t"+resInfo.GVR.Resource)
					} else {
						completions = append(completions, alias)
					}
				}
			}
			return completions, cobra.ShellCompDirectiveNoFileComp
		}

		if len(args) == 1 {
			// Complete resource names
			kind, ok := aliasToKind[args[0]]
			if !ok {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			resInfo, ok := checker.Resources[kind]
			if !ok {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			// Get namespace from flag or default
			namespace, _ := cmd.Flags().GetString("namespace")
			if namespace == "" {
				k8sCfg := f.ToRawKubeConfigLoader()
				ns, _, err := k8sCfg.Namespace()
				if err != nil {
					return nil, cobra.ShellCompDirectiveError
				}
				namespace = ns
			}

			// Get dynamic client
			dc, err := f.DynamicClient()
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			// List resources
			list, err := dc.Resource(resInfo.GVR).Namespace(namespace).List(context.Background(), metav1.ListOptions{})
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			var completions []string
			for _, item := range list.Items {
				name := item.GetName()
				if strings.HasPrefix(name, toComplete) {
					completions = append(completions, name)
				}
			}
			return completions, cobra.ShellCompDirectiveNoFileComp
		}

		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return cmds
}
