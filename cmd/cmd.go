package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	// Initialize all known client auth plugins.
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/term"

	"github.com/Ladicle/kubectl-check/pkg/checker"
	"github.com/Ladicle/kubectl-check/pkg/client"
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
	var showAllLogs bool
	var logTailLines int
	var checkOwner bool

	kubeConfigFlags := genericclioptions.NewConfigFlags(true)
	matchVersionFlags := cmdutil.NewMatchVersionFlags(kubeConfigFlags)
	f := cmdutil.NewFactory(matchVersionFlags)
	cfg, err := f.ToRESTConfig()
	if err != nil {
		_, _ = fmt.Fprintf(ioStreams.ErrOut, "Error creating REST config: %v\n", err)
		os.Exit(1)
	}
	clientManager, err := client.NewManager(cfg)
	if err != nil {
		_, _ = fmt.Fprintf(ioStreams.ErrOut, "Error creating client manager: %v\n", err)
		os.Exit(1)
	}

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

			// Get namespace from flags or kubeconfig
			ns := *kubeConfigFlags.Namespace
			if ns == "" {
				k8sCfg := f.ToRawKubeConfigLoader()
				ns, _, err = k8sCfg.Namespace()
				if err != nil {
					return fmt.Errorf("failed to get namespace: %w", err)
				}
			}

			target := types.NamespacedName{Name: name, Namespace: ns}
			chk := checker.NewChecker(&checker.Config{
				Target:        target,
				ClientManager: clientManager,
				ShowAllLogs:   showAllLogs,
				LogTailLines:  int64(logTailLines),
				CheckOwner:    checkOwner,
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
	cmds.PersistentFlags().BoolVarP(&showAllLogs, "show-all-logs", "L", false, "Show logs for all containers, not just failed ones")
	cmds.PersistentFlags().IntVarP(&logTailLines, "tail", "t", 20, "Number of lines to show from the end of the logs")
	cmds.PersistentFlags().BoolVarP(&checkOwner, "check-owner", "O", true, "Recursively check and display owner resources")
	cmds.SetUsageTemplate(getUsageTemplate(printer))

	// Register namespace flag completion
	_ = cmds.RegisterFlagCompletionFunc("namespace", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Use metadata client to fetch only names
		namespaces, err := clientManager.MetadataClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}).
			List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		var completions []string
		for _, ns := range namespaces.Items {
			if strings.HasPrefix(ns.Name, toComplete) {
				completions = append(completions, ns.Name)
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	})

	// Add shell completion
	cmds.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			// Complete resource types (one primary name per resource, no aliases)
			var completions []string
			seen := make(map[string]bool)
			for _, resInfo := range checker.Resources {
				// Use the primary alias (first one) if available
				if len(resInfo.Aliases) > 0 {
					primaryName := resInfo.Aliases[0]
					if strings.HasPrefix(primaryName, toComplete) && !seen[primaryName] {
						completions = append(completions, primaryName)
						seen[primaryName] = true
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

			// Get namespace from flag or default (only for namespaced resources)
			var list *metav1.PartialObjectMetadataList
			var err error

			if resInfo.Namespaced {
				namespace, _ := cmd.Flags().GetString("namespace")
				if namespace == "" {
					k8sCfg := f.ToRawKubeConfigLoader()
					ns, _, err := k8sCfg.Namespace()
					if err != nil {
						return nil, cobra.ShellCompDirectiveError
					}
					namespace = ns
				}
				// List namespaced resources
				list, err = clientManager.MetadataClient.Resource(resInfo.GVR).Namespace(namespace).List(context.Background(), metav1.ListOptions{})
			} else {
				// List cluster-scoped resources
				list, err = clientManager.MetadataClient.Resource(resInfo.GVR).List(context.Background(), metav1.ListOptions{})
			}

			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}

			var completions []string
			for _, item := range list.Items {
				if strings.HasPrefix(item.Name, toComplete) {
					completions = append(completions, item.Name)
				}
			}
			return completions, cobra.ShellCompDirectiveNoFileComp
		}

		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return cmds
}
