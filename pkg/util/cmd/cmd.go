package cmd

import (
	"fmt"
	"os"
)

func CheckErr(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
