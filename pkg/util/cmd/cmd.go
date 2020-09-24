package cmd

import (
	"os"
)

func CheckErr(err error) {
	if err == nil {
		return
	}
	os.Exit(1)
}
