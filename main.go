package main

import (
	"flag"
	"os"

	"k8s.io/kubectl/pkg/util/logs"

	"github.com/Ladicle/kubectl-check/cmd"
)

func init() {
	flag.Set("logtostderr", "false")
	flag.Set("log_file", "/dev/null")
}

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	c := cmd.NewCheckCmd()
	if err := c.Execute(); err != nil {
		os.Exit(1)
	}
}
