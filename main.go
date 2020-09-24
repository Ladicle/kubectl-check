package main

import (
	"flag"
	"os"

	"k8s.io/kubectl/pkg/util/logs"

	"github.com/Ladicle/kubectl-diagnose/cmd"
)

func init() {
	flag.Set("logtostderr", "false")
	flag.Set("log_file", "/dev/null")
}

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	c := cmd.NewDiagnoseCmd()
	if err := c.Execute(); err != nil {
		os.Exit(1)
	}
}
