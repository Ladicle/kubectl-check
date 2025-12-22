package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Ladicle/kubectl-check/pkg/checker"
	"github.com/Ladicle/kubectl-check/pkg/pritty"
)

func getUsageTemplate(p *pritty.Printer) string {
	// Group resources by their GVR group
	resourcesByGroup := make(map[string][]string)

	for _, info := range checker.Resources {
		// Skip resources without aliases (like Workload which is internal)
		if len(info.Aliases) == 0 {
			continue
		}

		// Use the first alias as the primary name
		primaryName := info.Aliases[0]

		// Use GVR group as-is, or "core" for empty group
		groupName := info.GVR.Group
		if groupName == "" {
			groupName = "core"
		}

		resourcesByGroup[groupName] = append(resourcesByGroup[groupName], primaryName)
	}

	// Sort group names
	var groupNames []string
	for groupName := range resourcesByGroup {
		groupNames = append(groupNames, groupName)
	}
	sort.Strings(groupNames)

	// Sort resources within each group
	for _, resources := range resourcesByGroup {
		sort.Strings(resources)
	}

	// Build grouped resource list
	var resourceList strings.Builder
	for _, groupName := range groupNames {
		resourceList.WriteString(fmt.Sprintf("  %s:\n", groupName))
		resourceList.WriteString(fmt.Sprintf("    %s\n", strings.Join(resourcesByGroup[groupName], ", ")))
	}

	usageTemplate := `%v:
  kubectl {{.UseLine}}

%v:
%v
%v:
  -O, --check-owner    Recursively traverse owner references upward (default true)
      --show-all-logs  Show logs for all containers (not just failed ones)
      --tail int       Number of log lines to display (default 100)
      --version        Version for check
      --options        Show full options of this command
  -h, --help           Show this message
  -R, --color          Enable color output even if stdout is not a terminal

%v:
  completion   Generate shell completion scripts

Use "kubectl {{.CommandPath}} --options" for full information about global flags.
`

	return fmt.Sprintf(usageTemplate,
		p.SprintHeader("Usage"),
		p.SprintHeader("Resources"),
		resourceList.String(),
		p.SprintHeader("Flags"),
		p.SprintHeader("Available Commands"),
	)
}
