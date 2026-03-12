package main

import (
	"os"

	"github.com/infraspecdev/goperf/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
