package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Ladicle/kubectl-check/pkg/checker"
	"github.com/Ladicle/kubectl-check/pkg/pritty"
)

func getUsageTemplate(p *pritty.Printer) string {
	// Build resource list from checker.Resources
	var resources []string
	for _, info := range checker.Resources {
		resources = append(resources, fmt.Sprintf("  - %s", strings.Join(info.Aliases, ", ")))
	}
	sort.Strings(resources)

	usageTemplate := `%v:
  kubectl {{.UseLine}}

%v:
%v

%v:
  --version    Version for check
  --options    Show full options of this command
  -h, --help   Show this message
  -R, --color  Enable color output even if stdout is not a terminal

%v:
  completion   Generate shell completion scripts

Use "kubectl {{.CommandPath}} --options" for full information about global flags.
`

	return fmt.Sprintf(usageTemplate,
		p.SprintHeader("Usage"),
		p.SprintHeader("Resources"),
		strings.Join(resources, "\n"),
		p.SprintHeader("Flags"),
		p.SprintHeader("Available Commands"),
	)
}
