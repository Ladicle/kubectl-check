package cmd

import (
	"fmt"

	"github.com/Ladicle/kubectl-check/pkg/pritty"
)

var usageTemplate = `%v:{{if .Runnable}}
  kubectl {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}

%v:{{range .Commands}}{{if .IsAvailableCommand}}
  - {{.NameAndAliases}}{{end}}{{end}}{{end}}{{if not .HasAvailableSubCommands}}

%v:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

%v:
{{.Example}}{{end}}

%v:{{if .HasAvailableSubCommands}}
  --version    Version for check
  --options    Show full options of this command
  -h, --help   Show this message
  -R, --color  Enable color output even if stdout is not a terminal{{else}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}

Use "kubectl {{.CommandPath}} --options" for full information about global flags.{{if .HasAvailableSubCommands}}
Use "kubectl {{.CommandPath}} <resource> --help" for more information about each resource.{{end}}
`

func getUsageTemplate(p *pritty.Printer) string {
	return fmt.Sprintf(usageTemplate,
		p.SprintHeader("Usage"),
		p.SprintHeader("Resources"),
		p.SprintHeader("Aliases"),
		p.SprintHeader("Example"),
		p.SprintHeader("Flags"),
	)
}
