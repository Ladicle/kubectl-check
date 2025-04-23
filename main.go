package main

import (
	"os"

	"k8s.io/klog/v2"

	"github.com/Ladicle/kubectl-check/cmd"
)

func main() {
	klog.InitFlags(nil)
	defer klog.Flush()

	c := cmd.NewCheckCmd()
	if err := c.Execute(); err != nil {
		os.Exit(1)
	}
}
