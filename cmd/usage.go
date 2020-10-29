package cmd

var usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}

Resources:{{range .Commands}}{{if .IsAvailableCommand}}
  - {{.NameAndAliases}}{{end}}{{end}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}

Flags:{{if .HasAvailableSubCommands}}
  --version    Version for check
  --options    Show full options of this command
  -h, --help   Show this message
  -R, --color  Enable color output even if stdout is not a terminal{{else}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}

Use "{{.CommandPath}} --options" for full information about global flags.{{if .HasAvailableSubCommands}}
Use "{{.CommandPath}} <resource> --help" for more information about each resource.{{end}}
`
